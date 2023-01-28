package watch

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/zoetrope/kubbernecker/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	logger *logr.Logger
	kube   *client.KubeClient
	gvk    schema.GroupVersionKind

	mu         sync.RWMutex
	statistics map[string]*Statistics
}

type Statistics struct {
	Resources        map[string]*ResourceStatistics
	AddCount         int
	DeleteCount      int
	UpdateTotalCount int
}

type ResourceStatistics struct {
	UpdateCount int
}

func NewWatcher(logger *logr.Logger, kube *client.KubeClient, resource schema.GroupVersionKind) *Watcher {
	return &Watcher{
		logger:     logger,
		kube:       kube,
		statistics: map[string]*Statistics{},
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
	w.mu.Lock()
	defer w.mu.Unlock()

	meta := obj.(*metav1.PartialObjectMetadata)
	if _, ok := w.statistics[meta.Namespace]; !ok {
		w.statistics[meta.Namespace] = &Statistics{
			Resources: map[string]*ResourceStatistics{},
		}
	}
	info := w.statistics[meta.Namespace]

	if _, ok := info.Resources[meta.Name]; !ok {
		info.Resources[meta.Name] = &ResourceStatistics{}
	}
	resInfo := info.Resources[meta.Name]

	switch event {
	case "add":
		info.AddCount += 1
	case "update":
		resInfo.UpdateCount += 1
	case "delete":
		info.DeleteCount += 1
	}
}

func (w *Watcher) PrintStatistics() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	b, err := json.MarshalIndent(w.statistics, "", "  ")
	if err != nil {
		w.logger.Error(err, "failed to marshal json")
	}
	return string(b)
}

func (w *Watcher) Start(ctx context.Context) error {
	w.logger.Info("start watcher")
	meta := &metav1.PartialObjectMetadata{}
	meta.SetGroupVersionKind(w.gvk)
	informer, err := w.kube.Cache.GetInformer(ctx, meta)
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
