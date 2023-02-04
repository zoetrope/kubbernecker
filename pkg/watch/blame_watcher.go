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

type BlameWatcher struct {
	logger   logr.Logger
	kube     *client.KubeClient
	gvk      schema.GroupVersionKind
	resource string

	mu         sync.RWMutex
	statistics BlameStatistics
}

type ManagerStatistics struct {
	UpdateCount int
}

type BlameStatistics struct {
	Managers     map[string]*ManagerStatistics
	LatestUpdate time.Time
}

func (in *BlameStatistics) DeepCopy() *BlameStatistics {
	if in == nil {
		return nil
	}
	out := new(BlameStatistics)
	in.DeepCopyInto(out)
	return out
}

func (in *BlameStatistics) DeepCopyInto(out *BlameStatistics) {
	*out = *in

	if in.Managers != nil {
		in, out := &in.Managers, &out.Managers
		*out = make(map[string]*ManagerStatistics, len(*in))
		for key, val := range *in {
			var outVal *ManagerStatistics
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ManagerStatistics)
				*out = *in
			}
			(*out)[key] = outVal
		}
	}
	return
}
func NewBlameWatcher(logger logr.Logger, kube *client.KubeClient, gvk schema.GroupVersionKind, resource string) *BlameWatcher {
	statistics := BlameStatistics{}
	statistics.Managers = make(map[string]*ManagerStatistics)
	statistics.LatestUpdate = time.Now()

	return &BlameWatcher{
		logger:     logger,
		kube:       kube,
		statistics: statistics,
		gvk:        gvk,
		resource:   resource,
	}
}

func (w *BlameWatcher) collect(obj interface{}) {
	meta := obj.(*metav1.PartialObjectMetadata)

	if meta.Name != w.resource {
		w.logger.V(10).Info("no target", "res", meta.Name)
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	latest := w.statistics.LatestUpdate
	for _, field := range meta.ManagedFields {
		if field.Time == nil {
			continue
		}
		if (*field.Time).Time.After(w.statistics.LatestUpdate) {
			if _, ok := w.statistics.Managers[field.Manager]; !ok {
				w.statistics.Managers[field.Manager] = &ManagerStatistics{}
			}
			latest = (*field.Time).Time
			w.statistics.Managers[field.Manager].UpdateCount += 1
		}
	}
	w.statistics.LatestUpdate = latest
}

func (w *BlameWatcher) Statistics() *BlameStatistics {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.statistics.DeepCopy()
}

func (w *BlameWatcher) Start(ctx context.Context) error {
	w.logger.Info("start watcher")

	meta := &metav1.PartialObjectMetadata{}
	meta.SetGroupVersionKind(w.gvk)
	informer, err := w.kube.Cluster.GetCache().GetInformer(ctx, meta)
	if err != nil {
		return err
	}
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.collect(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.collect(newObj)
		},
		DeleteFunc: func(obj interface{}) {
		},
	})

	return err
}
