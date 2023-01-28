package client

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type KubeClient struct {
	Mapper    meta.RESTMapper
	Cache     cache.Cache
	Client    client.Client
	Discovery *discovery.DiscoveryClient
}

func MakeKubeClient(config *genericclioptions.ConfigFlags, allNamespaces bool) (*KubeClient, error) {
	cfg, err := config.ToRESTConfig()
	if err != nil {
		return nil, err
	}

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

	cacheOpts := cache.Options{
		Scheme: scheme,
		Mapper: mapper,
	}
	if !allNamespaces {
		ns, _, err := config.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return nil, err
		}
		cacheOpts.Namespace = ns
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
