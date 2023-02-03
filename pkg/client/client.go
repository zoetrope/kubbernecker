package client

import (
	"context"
	"errors"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type KubeClient struct {
	Mapper    meta.RESTMapper
	Cache     cache.Cache
	Client    client.Client
	Discovery *discovery.DiscoveryClient

	cancel context.CancelFunc
}

func MakeKubeClientFromRestConfig(cfg *rest.Config, namespace string) (*KubeClient, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return nil, err
	}
	resync := 30 * time.Second
	cacheOpts := cache.Options{
		Scheme: scheme,
		Mapper: mapper,
		Resync: &resync,
	}
	if namespace != "" {
		cacheOpts.Namespace = namespace
	}

	ca, err := cache.New(cfg, cacheOpts)
	if err != nil {
		return nil, err
	}

	cli, err := client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader:       ca,
		Client:            c,
		CacheUnstructured: true,
	})
	if err != nil {
		return nil, err
	}

	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		Mapper:    mapper,
		Cache:     ca,
		Client:    cli,
		Discovery: disco,
	}, nil
}

func MakeKubeClient(config *genericclioptions.ConfigFlags, allNamespaces bool) (*KubeClient, error) {
	cfg, err := config.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	namespace := ""
	if !allNamespaces {
		namespace, _, err = config.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return nil, err
		}
	}

	return MakeKubeClientFromRestConfig(cfg, namespace)
}

var excludedResources = []schema.GroupVersionKind{
	{
		Group:   "",
		Version: "v1",
		Kind:    "Binding",
	},
	{
		Group:   "authorization.k8s.io",
		Version: "v1",
		Kind:    "LocalSubjectAccessReview",
	},
	{
		Group:   "metrics.k8s.io",
		Version: "v1beta1",
		Kind:    "PodMetrics",
	},
	{
		Group:   "",
		Version: "v1",
		Kind:    "Event",
	},
	{
		Group:   "",
		Version: "v1",
		Kind:    "Endpoints",
	},
	{
		Group:   "discovery.k8s.io",
		Version: "v1",
		Kind:    "EndpointSlice",
	},
	{
		Group:   "coordination.k8s.io",
		Version: "v1",
		Kind:    "Lease",
	},
	{
		Group:   "",
		Version: "v1",
		Kind:    "Node",
	},
}

func IsExcludedResource(gvk schema.GroupVersionKind) bool {
	for _, er := range excludedResources {
		if er == gvk {
			return true
		}
	}
	return false
}

func (k *KubeClient) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	k.cancel = cancel
	go func() {
		klog.V(3).Info("starting cache")
		err := k.Cache.Start(ctx)
		if err != nil {
			klog.Error("failed to start cache", err)
			return
		}
		klog.V(1).Info("cache was closed")
	}()

	klog.V(3).Info("waiting for cache sync")
	ok := k.Cache.WaitForCacheSync(ctx)
	if !ok {
		klog.Errorf("could not sync cache")
		return errors.New("could not sync cache")
	}
	return nil
}

func (k *KubeClient) Stop() {
	k.cancel()
}

func (k *KubeClient) DetectGVK(arg string) (*schema.GroupVersionKind, error) {
	gr := schema.ParseGroupResource(arg)
	gvk, err := k.Mapper.KindFor(gr.WithVersion(""))
	if err == nil {
		return &gvk, nil
	}

	gvr, gr := schema.ParseResourceArg(arg)
	if gvr != nil {
		gvk, err := k.Mapper.KindFor(*gvr)
		if err == nil {
			return &gvk, nil
		}
	}
	gvk, err = k.Mapper.KindFor(gr.WithVersion(""))
	if err == nil {
		return &gvk, nil
	}
	return nil, err
}
