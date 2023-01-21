package sub

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type watchOptions struct {
	targets       []string
	allNamespaces bool

	streams genericclioptions.IOStreams
	config  *genericclioptions.ConfigFlags

	kube *kubeClient
}

func newWatchCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &watchOptions{}

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(streams, config, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().StringSliceVar(&opts.targets, "targets", nil, "targets")
	cmd.Flags().BoolVarP(&opts.allNamespaces, "all-namespaces", "A", false, "If true, watch the resources in all namespaces.")

	return cmd
}

func (o *watchOptions) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.config = config
	o.streams = streams

	kube, err := makeKubeClient(config, o.allNamespaces)
	if err != nil {
		return err
	}
	o.kube = kube

	return nil
}

func (o *watchOptions) Run(ctx context.Context) error {

	go func() {
		klog.V(3).Info("starting cache")
		err := o.kube.cache.Start(ctx)
		if err != nil {
			klog.Error("failed to start cache", err)
			return
		}
		klog.V(1).Info("cache was closed")
	}()

	klog.V(3).Info("waiting for cache sync")
	ok := o.kube.cache.WaitForCacheSync(ctx)
	if !ok {
		klog.Errorf("could not sync cache")
		return errors.New("could not sync cache")
	}

	for _, target := range o.targets {
		err := o.start(ctx, target)
		if err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		klog.V(3).Info("done")
		return nil
	case <-time.After(5 * time.Minute):
		klog.V(3).Info("timed out")
		return nil
	}
}

func (o *watchOptions) start(ctx context.Context, target string) error {
	gvk, err := o.detectGVK(target)
	if err != nil {
		return err
	}
	fmt.Printf("gvk: %s\n", gvk.String())

	err = o.watchResource(ctx, *gvk)
	if err != nil {
		return err
	}
	return nil
}

func printMetadata(out io.Writer, event string, obj interface{}) {
	meta := obj.(*metav1.PartialObjectMetadata)
	fmt.Fprintf(out, "%s: %s/%s/%s\n", event, meta.GroupVersionKind(), meta.Namespace, meta.Name)
	for _, m := range meta.ManagedFields {
		fmt.Fprintf(out, "  - manager: %s(%s)\n", m.Manager, m.Time.Format(time.RFC3339))
	}
}
func (o *watchOptions) watchResource(ctx context.Context, gvk schema.GroupVersionKind) error {
	meta := &metav1.PartialObjectMetadata{}
	meta.SetGroupVersionKind(gvk)
	informer, err := o.kube.cache.GetInformer(ctx, meta)
	if err != nil {
		return err
	}
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			printMetadata(o.streams.Out, "add", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			printMetadata(o.streams.Out, "update", newObj)
		},
		DeleteFunc: func(obj interface{}) {
			printMetadata(o.streams.Out, "delete", obj)
		},
	})

	return err
}

func (o *watchOptions) detectGVK(arg string) (*schema.GroupVersionKind, error) {
	gvk := schema.GroupVersionKind{}
	var err error

	gvr, gr := schema.ParseResourceArg(arg)
	if gvr != nil {
		gvk, err = o.kube.mapper.KindFor(*gvr)
		if err != nil {
			return nil, err
		}
	}
	if gvk.Empty() {
		gvk, err = o.kube.mapper.KindFor(gr.WithVersion(""))
		if err != nil {
			return nil, err
		}
	}
	if !gvk.Empty() {
		return &gvk, nil
	}

	gvk2, gk := schema.ParseKindArg(arg)
	if gvk2 != nil {
		gvk = gk.WithVersion("")
	} else {
		gvk = *gvk2
	}
	if gvk.Empty() {
		return nil, fmt.Errorf("failed to detect GroupVersionKind: %s", arg)
	}
	return &gvk, nil
}
