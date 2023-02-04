package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/zoetrope/kubbernecker/pkg/client"
	"github.com/zoetrope/kubbernecker/pkg/watch"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type WatcherManager struct {
	kube     *client.KubeClient
	watchers []*watch.Watcher
	logger   logr.Logger
}

func NewWatcherManager(logger logr.Logger, c *client.KubeClient) *WatcherManager {
	return &WatcherManager{
		logger: logger,
		kube:   c,
	}
}

func (m *WatcherManager) Start(ctx context.Context) error {
	resources, err := m.targetResources()
	if err != nil {
		return err
	}

	for _, res := range resources {
		klog.V(2).Info("create watcher", res)
		watcher := watch.NewWatcher(m.logger, m.kube, res)
		m.watchers = append(m.watchers, watcher)
		klog.V(2).Info("start watcher", res)
		err = watcher.Start(ctx)
		if err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		klog.V(3).Info("done")
		return nil
	}
	return nil
}

func (m *WatcherManager) targetResources() ([]schema.GroupVersionKind, error) {
	targets := make([]schema.GroupVersionKind, 0)

	serverResources, err := m.kube.Discovery.ServerPreferredNamespacedResources()
	if err != nil {
		return nil, err
	}
	for _, resList := range serverResources {
		for _, res := range resList.APIResources {
			gv, err := schema.ParseGroupVersion(resList.GroupVersion)
			if err != nil {
				gv = schema.GroupVersion{}
			}
			gvk := gv.WithKind(res.Kind)
			if client.IsExcludedResource(gvk) {
				continue
			}
			targets = append(targets, gvk)
		}
	}

	return targets, nil
}
