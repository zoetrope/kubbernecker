package watch

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/zoetrope/kubbernecker/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Watcher struct {
	logger      logr.Logger
	kube        *client.KubeClient
	gvk         schema.GroupVersionKind
	nsSelector  labels.Selector
	resSelector labels.Selector

	startTime time.Time

	mu         sync.RWMutex
	statistics Statistics
}

func NewWatcher(logger logr.Logger, kube *client.KubeClient, gvk schema.GroupVersionKind, nsSelector labels.Selector, resSelector labels.Selector) *Watcher {
	statistics := Statistics{}
	statistics.GroupVersionKind = gvk
	statistics.Namespaces = make(map[string]*NamespaceStatistics)

	return &Watcher{
		logger:      logger,
		kube:        kube,
		statistics:  statistics,
		gvk:         gvk,
		nsSelector:  nsSelector,
		resSelector: resSelector,
	}
}

func (w *Watcher) handle(obj interface{}, event string) {
	meta := obj.(*metav1.PartialObjectMetadata)

	w.logger.V(3).Info("Event", "event", event, "gvk", meta.GroupVersionKind(), "namespace", meta.Namespace, "name", meta.Name)
	if event == "add" {
		if meta.CreationTimestamp.Time.Before(w.startTime) {
			// Ignore add events for resources created before start of watching
			w.logger.V(3).Info("Ignore resources created before start of watch", "start", w.startTime, "creation", meta.CreationTimestamp)
			return
		}
	}

	if !w.resSelector.Matches(labels.Set(meta.Labels)) {
		return
	}
	if !w.nsSelector.Empty() && meta.Namespace != "" {
		ns := &corev1.Namespace{}
		err := w.kube.Cluster.GetClient().Get(context.TODO(), ctrlclient.ObjectKey{Name: meta.Namespace}, ns)
		if err != nil {
			w.logger.Error(err, "failed to get namespace", "namespace", meta.Namespace)
			return
		}
		if !w.resSelector.Matches(labels.Set(ns.Labels)) {
			return
		}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.statistics.Namespaces[meta.Namespace]; !ok {
		ks := &NamespaceStatistics{}
		ks.Resources = make(map[string]*ResourceStatistics)
		w.statistics.Namespaces[meta.Namespace] = ks
	}
	info := w.statistics.Namespaces[meta.Namespace]

	if _, ok := info.Resources[meta.Name]; !ok {
		info.Resources[meta.Name] = &ResourceStatistics{}
	}
	resInfo := info.Resources[meta.Name]

	switch event {
	case "add":
		info.AddCount += 1
	case "update":
		info.UpdateCount += 1
		resInfo.UpdateCount += 1
	case "delete":
		info.DeleteCount += 1
	}
}

func (w *Watcher) Statistics() *Statistics {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.statistics.DeepCopy()
}

func (w *Watcher) Start(ctx context.Context) error {
	w.logger.Info("start watcher", "gvk", w.gvk.String(), "nsSelector", w.nsSelector.String(), "resSelector", w.resSelector.String())
	w.startTime = time.Now()

	meta := &metav1.PartialObjectMetadata{}
	meta.SetGroupVersionKind(w.gvk)
	informer, err := w.kube.Cluster.GetCache().GetInformer(ctx, meta)
	if err != nil {
		return err
	}
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.handle(obj, "add")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.handle(newObj, "update")
		},
		DeleteFunc: func(obj interface{}) {
			w.handle(obj, "delete")
		},
	})

	return err
}
