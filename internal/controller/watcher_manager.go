package controller

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"github.com/zoetrope/kubbernecker/pkg/client"
	"github.com/zoetrope/kubbernecker/pkg/config"
	"github.com/zoetrope/kubbernecker/pkg/watch"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type WatcherManager struct {
	kube     *client.KubeClient
	watchers []*watch.Watcher
	logger   logr.Logger
	config   *config.Config
}

func NewWatcherManager(logger logr.Logger, kubeClient *client.KubeClient, cfg *config.Config) *WatcherManager {
	return &WatcherManager{
		logger: logger,
		kube:   kubeClient,
		config: cfg,
	}
}

func (m *WatcherManager) Start(ctx context.Context) error {
	resources, err := m.targetResources()
	if err != nil {
		return err
	}

	for _, res := range resources {
		klog.V(2).Info("create watcher", res)
		nsSelector, resSelector, err := m.config.SelectorFor(metav1.GroupVersionKind{
			Group:   res.Group,
			Version: res.Version,
			Kind:    res.Kind,
		})
		if err != nil {
			return err
		}
		watcher := watch.NewWatcher(m.logger, m.kube, res, nsSelector, resSelector)
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
	}
	return nil
}

func (m *WatcherManager) targetResources() ([]schema.GroupVersionKind, error) {
	targets := make([]schema.GroupVersionKind, 0)

	if len(m.config.TargetResources) > 0 {
		for _, target := range m.config.TargetResources {
			gvk := schema.GroupVersionKind{
				Group:   target.Group,
				Version: target.Version,
				Kind:    target.Kind,
			}
			err := m.kube.IsValidGVK(&gvk)
			if err != nil {
				return nil, err
			}
			targets = append(targets, gvk)
		}
	} else {
		serverResources, err := m.kube.Discovery.ServerPreferredResources()
		if err != nil {
			return nil, err
		}
		for _, resList := range serverResources {
			for _, res := range resList.APIResources {
				if !m.config.EnableClusterResources && !res.Namespaced {
					continue
				}
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
	}

	return targets, nil
}
