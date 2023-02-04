package watch

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/zoetrope/kubbernecker/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	logger logr.Logger
	kube   *client.KubeClient
	gvk    schema.GroupVersionKind

	startTime  time.Time
	mu         sync.RWMutex
	statistics Statistics
}

func NewWatcher(logger logr.Logger, kube *client.KubeClient, resource schema.GroupVersionKind) *Watcher {
	statistics := Statistics{}
	statistics.GroupVersionKind = resource.String()
	statistics.Namespaces = make(map[string]*NamespaceStatistics)

	return &Watcher{
		logger:     logger,
		kube:       kube,
		statistics: statistics,
		gvk:        resource,
	}
}

func (w *Watcher) printMetadata(event string, obj interface{}) {
	meta := obj.(*metav1.PartialObjectMetadata)
	w.logger.V(3).Info("Event", "event", event, "gvk", meta.GroupVersionKind(), "namespace", meta.Namespace, "name", meta.Name)
	for _, m := range meta.ManagedFields {
		w.logger.V(5).Info("Manager", "manager", m.Manager, "managedTime", m.Time.Format(time.RFC3339))
	}
}

func (w *Watcher) collect(obj interface{}, event string) {
	meta := obj.(*metav1.PartialObjectMetadata)

	if event == "add" {
		if meta.CreationTimestamp.Time.Before(w.startTime) {
			// Ignore add events for resources created before start of watch
			w.logger.V(3).Info("Ignore resources created before start of watch", "start", w.startTime, "creation", meta.CreationTimestamp)
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
	w.logger.Info("start watcher")
	w.startTime = time.Now()

	meta := &metav1.PartialObjectMetadata{}
	meta.SetGroupVersionKind(w.gvk)
	informer, err := w.kube.Cluster.GetCache().GetInformer(ctx, meta)
	if err != nil {
		return err
	}
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.collect(obj, "add")
			w.printMetadata("add", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.collect(newObj, "update")
			w.printMetadata("update", newObj)
		},
		DeleteFunc: func(obj interface{}) {
			w.collect(obj, "delete")
			w.printMetadata("delete", obj)
		},
	})

	return err
}
