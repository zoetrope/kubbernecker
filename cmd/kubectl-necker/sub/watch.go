package sub

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"
	"github.com/zoetrope/kubbernecker/pkg"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type watchOptions struct {
	*rootOpts

	resources     []string
	allNamespaces bool
	allResources  bool

	kube     *pkg.KubeClient
	watchers []*pkg.Watcher
}

func newWatchCmd(parentOpts *rootOpts) *cobra.Command {
	opts := &watchOptions{
		rootOpts: parentOpts,
	}

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Fill(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVarP(&opts.allResources, "all-resources", "a", false, "If true, watch all resources in the specified namespaces.")
	cmd.Flags().BoolVarP(&opts.allNamespaces, "all-namespaces", "A", false, "If true, watch the resources in all namespaces.")

	return cmd
}

func (o *watchOptions) Fill(cmd *cobra.Command, args []string) error {
	kube, err := pkg.MakeKubeClient(o.config, o.allNamespaces)
	if err != nil {
		return err
	}
	o.kube = kube
	o.resources = args

	if len(o.resources) > 0 && o.allResources {
		return errors.New("resources and --all-resources cannot be used together")
	}
	if len(o.resources) == 0 && !o.allResources {
		return errors.New("resources or --all-resources is required but not provided")
	}

	return nil
}

func (o *watchOptions) Run(ctx context.Context) error {

	go func() {
		klog.V(3).Info("starting cache")
		err := o.kube.Cache.Start(ctx)
		if err != nil {
			klog.Error("failed to start cache", err)
			return
		}
		klog.V(1).Info("cache was closed")
	}()

	klog.V(3).Info("waiting for cache sync")
	ok := o.kube.Cache.WaitForCacheSync(ctx)
	if !ok {
		klog.Errorf("could not sync cache")
		return errors.New("could not sync cache")
	}

	resources, err := o.targetResources()
	if err != nil {
		return err
	}

	for _, res := range resources {
		klog.V(2).Info("create watcher", res)
		watcher := pkg.NewWatcher(o.streams, o.kube, res)
		o.watchers = append(o.watchers, watcher)
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
	case <-time.After(1 * time.Minute):
		klog.V(3).Info("timed out")
		for _, w := range o.watchers {
			w.PrintStatistics()
		}
		return nil
	}
}

func (o *watchOptions) targetResources() ([]schema.GroupVersionKind, error) {
	targets := make([]schema.GroupVersionKind, 0)

	if o.allResources {
		serverResources, err := o.kube.Discovery.ServerPreferredNamespacedResources()
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
				if o.isExcludedResource(gvk) {
					continue
				}
				targets = append(targets, gvk)
			}
		}
	} else {
		for _, res := range o.resources {
			gvk, err := o.detectGVK(res)
			if err != nil {
				return nil, err
			}
			targets = append(targets, *gvk)
		}
	}

	return targets, nil
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

func (o *watchOptions) isExcludedResource(gvk schema.GroupVersionKind) bool {
	for _, er := range excludedResources {
		if er == gvk {
			return true
		}
	}
	return false
}

func (o *watchOptions) detectGVK(arg string) (*schema.GroupVersionKind, error) {
	gr := schema.ParseGroupResource(arg)
	gvk, err := o.kube.Mapper.KindFor(gr.WithVersion(""))
	if err != nil {
		return nil, err
	}
	return &gvk, nil
}
