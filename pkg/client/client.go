package client

import (
	"fmt"
	"time"

	"k8s.io/utils/pointer"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type KubeClient struct {
	Cluster   cluster.Cluster
	Discovery *discovery.DiscoveryClient
}

func NewCachingClient(cache cache.Cache, config *rest.Config, options client.Options, uncachedObjects ...client.Object) (client.Client, error) {
	c, err := client.New(config, options)
	if err != nil {
		return nil, err
	}

	return client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader:       cache,
		Client:            c,
		UncachedObjects:   uncachedObjects,
		CacheUnstructured: true,
	})
}

var _ cluster.NewClientFunc = NewCachingClient

func MakeKubeClientFromCluster(c cluster.Cluster) (*KubeClient, error) {
	disco, err := discovery.NewDiscoveryClientForConfig(c.GetConfig())
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		Cluster:   c,
		Discovery: disco,
	}, nil
}

func MakeKubeClientFromRestConfig(cfg *rest.Config, namespace string) (*KubeClient, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c, err := cluster.New(cfg, func(opts *cluster.Options) {
		opts.Namespace = namespace
		opts.Scheme = scheme
		opts.SyncPeriod = pointer.Duration(0 * time.Second)
		opts.NewClient = NewCachingClient
	})

	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		Cluster:   c,
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
	{
		Group:   "",
		Version: "v1",
		Kind:    "ComponentStatus",
	},
	{
		Group:   "events.k8s.io",
		Version: "v1",
		Kind:    "Event",
	},
	{
		Group:   "authentication.k8s.io",
		Version: "v1",
		Kind:    "TokenReview",
	},
	{
		Group:   "authorization.k8s.io",
		Version: "v1",
		Kind:    "SubjectAccessReview",
	},
	{
		Group:   "authorization.k8s.io",
		Version: "v1",
		Kind:    "SelfSubjectAccessReview",
	},
	{
		Group:   "authorization.k8s.io",
		Version: "v1",
		Kind:    "SelfSubjectRulesReview",
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

func (k *KubeClient) DetectGVK(arg string) (*schema.GroupVersionKind, error) {
	mapper := k.Cluster.GetRESTMapper()
	gr := schema.ParseGroupResource(arg)
	gvk, err := mapper.KindFor(gr.WithVersion(""))
	if err == nil {
		return &gvk, nil
	}

	gvr, gr := schema.ParseResourceArg(arg)
	if gvr != nil {
		gvk, err := mapper.KindFor(*gvr)
		if err == nil {
			return &gvk, nil
		}
	}
	gvk, err = mapper.KindFor(gr.WithVersion(""))
	if err == nil {
		return &gvk, nil
	}
	return nil, err
}
func (k *KubeClient) IsValidGVK(gvk *schema.GroupVersionKind) error {
	_, err := k.Cluster.GetRESTMapper().RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
	if err != nil {
		return fmt.Errorf("invalid gvk %s: %w", gvk.String(), err)
	}
	return nil
}
