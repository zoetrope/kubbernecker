package sub

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zoetrope/kubbernecker/pkg/client"
	"github.com/zoetrope/kubbernecker/pkg/cobwrap"
	"github.com/zoetrope/kubbernecker/pkg/watch"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type watchOptions struct {
	resources     []string
	allNamespaces bool
	allResources  bool

	kube     *client.KubeClient
	watchers []*watch.Watcher
}

func newWatchCmd() *cobwrap.Command[*watchOptions] {

	cmd := &cobwrap.Command[*watchOptions]{
		Command: &cobra.Command{
			Use:   "watch",
			Short: "",
			Long:  ``,
		},
		Options: &watchOptions{},
	}

	cmd.Command.Flags().BoolVarP(&cmd.Options.allResources, "all-resources", "a", false, "If true, watch all resources in the specified namespaces.")
	cmd.Command.Flags().BoolVarP(&cmd.Options.allNamespaces, "all-namespaces", "A", false, "If true, watch the resources in all namespaces.")

	return cmd
}

func (o *watchOptions) Fill(cmd *cobra.Command, args []string) error {
	root := cobwrap.GetOpt[*rootOpts](cmd)

	kube, err := client.MakeKubeClient(root.config, o.allNamespaces)
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

func (o *watchOptions) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	root := cobwrap.GetOpt[*rootOpts](cmd)

	err := o.kube.Start(ctx)
	if err != nil {
		return err
	}

	resources, err := o.targetResources()
	if err != nil {
		return err
	}

	for _, res := range resources {
		klog.V(2).Info("create watcher", res)
		watcher := watch.NewWatcher(root.logger, o.kube, res)
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
			statistics := w.Statistics()
			b, err := json.MarshalIndent(statistics, "", "  ")
			if err != nil {
				klog.Errorf("failed to marshal json: %v", err)
			}
			fmt.Fprint(root.streams.Out, string(b))
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
				if client.IsExcludedResource(gvk) {
					continue
				}
				targets = append(targets, gvk)
			}
		}
	} else {
		for _, res := range o.resources {
			gvk, err := o.kube.DetectGVK(res)
			if err != nil {
				return nil, err
			}
			targets = append(targets, *gvk)
		}
	}

	return targets, nil
}
