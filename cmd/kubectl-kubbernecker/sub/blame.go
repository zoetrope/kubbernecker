package sub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zoetrope/kubbernecker/pkg/client"
	"github.com/zoetrope/kubbernecker/pkg/cobwrap"
	"github.com/zoetrope/kubbernecker/pkg/watch"
	"k8s.io/klog/v2"
)

type blameOptions struct {
	kube     *client.KubeClient
	duration time.Duration
}

func newBlameCmd() *cobwrap.Command[*blameOptions] {

	cmd := &cobwrap.Command[*blameOptions]{
		Command: &cobra.Command{
			Use:   "blame TYPE[.VERSION][.GROUP] NAME",
			Short: "Print the name of managers that updated the given resource",
			Long: `Print the name of managers that updated the given resource.

Examples:
  # Print managers that updated "test" ConfigMap resource
  kubectl kubbernecker blame configmap test
`,
			Args: cobra.ExactArgs(2),
		},
		Options: &blameOptions{},
	}

	cmd.Command.Flags().DurationVarP(&cmd.Options.duration, "duration", "d", 1*time.Minute, "")

	return cmd
}

func (o *blameOptions) Fill(cmd *cobra.Command, args []string) error {
	root := cobwrap.GetOpt[*rootOpts](cmd)

	kube, err := client.MakeKubeClient(root.config, true)
	if err != nil {
		return err
	}
	o.kube = kube
	return nil
}

func (o *blameOptions) Run(cmd *cobra.Command, args []string) error {
	root := cobwrap.GetOpt[*rootOpts](cmd)

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	root.logger.Info("run")

	go func() {
		err := o.kube.Cluster.Start(ctx)
		if err != nil {
			root.logger.Error(err, "failed to start cluster")
		}
	}()

	gvk, err := o.kube.DetectGVK(args[0])
	if err != nil {
		return err
	}

	klog.V(2).Info("create watcher", *gvk)
	watcher := watch.NewBlameWatcher(root.logger, o.kube, *gvk, args[1])
	klog.V(2).Info("start watcher", *gvk)
	err = watcher.Start(ctx)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		klog.V(3).Info("done")
	case <-time.After(o.duration):
		klog.V(3).Info("timed out")
		statistics := watcher.Statistics()
		b, err := json.MarshalIndent(statistics, "", "  ")
		if err != nil {
			klog.Errorf("failed to marshal json: %v", err)
		}
		fmt.Fprint(root.streams.Out, string(b))
	}
	return nil
}
