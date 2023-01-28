package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/zoetrope/kubbernecker/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	streams genericclioptions.IOStreams
	kube    *client.KubeClient
	gvk     schema.GroupVersionKind

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

func NewWatcher(streams genericclioptions.IOStreams, kube *client.KubeClient, resource schema.GroupVersionKind) *Watcher {
	return &Watcher{
		streams:    streams,
		kube:       kube,
		statistics: map[string]*Statistics{},
		gvk:        resource,
	}
}

func printMetadata(out io.Writer, event string, obj interface{}) {
	meta := obj.(*metav1.PartialObjectMetadata)
	fmt.Fprintf(out, "%s: %s/%s/%s\n", event, meta.GroupVersionKind(), meta.Namespace, meta.Name)
	for _, m := range meta.ManagedFields {
		fmt.Fprintf(out, "  - manager: %s(%s)\n", m.Manager, m.Time.Format(time.RFC3339))
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

func (w *Watcher) PrintStatistics() {
	w.mu.RLock()
	defer w.mu.RUnlock()

	b, err := json.MarshalIndent(w.statistics, "", "  ")
	if err != nil {
		fmt.Fprintf(w.streams.ErrOut, "failed to marshal json: %v\n", err)
	}
	fmt.Fprintf(w.streams.Out, "%s", string(b))
}

func (w *Watcher) Start(ctx context.Context) error {
	meta := &metav1.PartialObjectMetadata{}
	meta.SetGroupVersionKind(w.gvk)
	informer, err := w.kube.Cache.GetInformer(ctx, meta)
	if err != nil {
		return err
	}
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.collect(obj, "add")
			printMetadata(w.streams.Out, "add", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.collect(newObj, "update")
			printMetadata(w.streams.Out, "update", newObj)
		},
		DeleteFunc: func(obj interface{}) {
			w.collect(obj, "delete")
			printMetadata(w.streams.Out, "delete", obj)
		},
	})

	return err
}
