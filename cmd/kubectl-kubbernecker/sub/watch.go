package sub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zoetrope/kubbernecker/pkg/client"
	"github.com/zoetrope/kubbernecker/pkg/cobwrap"
	"github.com/zoetrope/kubbernecker/pkg/watch"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type watchOptions struct {
	resources     []string
	allNamespaces bool
	allResources  bool
	duration      time.Duration

	kube     *client.KubeClient
	watchers []*watch.Watcher
}

func newWatchCmd() *cobwrap.Command[*watchOptions] {

	cmd := &cobwrap.Command[*watchOptions]{
		Command: &cobra.Command{
			Use:   "watch (TYPE[.VERSION][.GROUP]...)",
			Short: "Print the number of times a resource is updated",
			Long: `Print the number of times a resource is updated.

Examples:
  # Watch Pod resources in "default" namespace
  kubectl kubbernecker watch pods -n default

  # Watch Pod resources in all namespaces
  kubectl kubbernecker watch pods --all-namespaces

  # Watch all resources in all namespaces
  kubectl kubbernecker watch --all-resources --all-namespaces
`,
		},
		Options: &watchOptions{},
	}

	cmd.Command.Flags().BoolVarP(&cmd.Options.allResources, "all-resources", "a", false, "If true, watch all resources in the specified namespaces.")
	cmd.Command.Flags().BoolVarP(&cmd.Options.allNamespaces, "all-namespaces", "A", false, "If true, watch the resources in all namespaces.")
	cmd.Command.Flags().DurationVarP(&cmd.Options.duration, "duration", "d", 1*time.Minute, "")

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
		return errors.New("the type of resource and `--all-namespaces` flag cannot be used together")
	}
	if len(o.resources) == 0 && !o.allResources {
		return errors.New("you must specify the type of resource to get or `--all-namespaces` flag")
	}

	return nil
}

func (o *watchOptions) Run(cmd *cobra.Command, args []string) error {
	klog.V(1).Info("run watch")
	root := cobwrap.GetOpt[*rootOpts](cmd)

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	go func() {
		err := o.kube.Cluster.Start(ctx)
		if err != nil {
			root.logger.Error(err, "failed to start cluster")
		}
	}()

	resources, err := o.targetResources()
	if err != nil {
		return err
	}

	for _, res := range resources {
		klog.V(2).Info("create watcher", res)
		watcher := watch.NewWatcher(root.logger, o.kube, res, labels.Everything(), labels.Everything())
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
	case <-time.After(o.duration):
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
