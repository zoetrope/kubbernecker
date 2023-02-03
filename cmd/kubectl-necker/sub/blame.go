package sub

import (
	"github.com/spf13/cobra"
	"github.com/zoetrope/kubbernecker/pkg/client"
	"github.com/zoetrope/kubbernecker/pkg/cobwrap"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type blameOptions struct {
	kube *client.KubeClient
}

func newBlameCmd() *cobwrap.Command[*blameOptions] {

	cmd := &cobwrap.Command[*blameOptions]{
		Command: &cobra.Command{
			Use:   "blame",
			Short: "",
			Long:  ``,
		},
		Options: &blameOptions{},
	}

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
	ctx := cmd.Context()
	root := cobwrap.GetOpt[*rootOpts](cmd)

	root.logger.Info("run")

	err := o.kube.Start(ctx)
	if err != nil {
		return err
	}

	serverResources, err := o.kube.Discovery.ServerPreferredNamespacedResources()
	if err != nil {
		return err
	}
	for _, resList := range serverResources {

		root.logger.Info("apiversion", "groupversion", resList.GroupVersion, "apiversion", resList.APIVersion)

		for _, res := range resList.APIResources {
			gv, err := schema.ParseGroupVersion(resList.GroupVersion)
			if err != nil {
				gv = schema.GroupVersion{}
			}
			gvk := gv.WithKind(res.Kind)
			if client.IsExcludedResource(gvk) {
				continue
			}
			root.logger.Info("resource", "group", res.Group, "version", res.Version, "kind", res.Kind)
		}
	}

	groups, apiresources, err := o.kube.Discovery.ServerGroupsAndResources()
	if err != nil {
		return err
	}

	for _, g := range groups {
		for _, v := range g.Versions {
			root.logger.Info("group-version", "version", v.GroupVersion)
		}
	}

	for _, resList := range apiresources {

		root.logger.Info("apiversion", "groupversion", resList.GroupVersion, "apiversion", resList.APIVersion)

		for _, res := range resList.APIResources {
			gv, err := schema.ParseGroupVersion(resList.GroupVersion)
			if err != nil {
				gv = schema.GroupVersion{}
			}
			gvk := gv.WithKind(res.Kind)
			if client.IsExcludedResource(gvk) {
				continue
			}
			root.logger.Info("resource", "group", res.Group, "version", res.Version, "kind", res.Kind)
		}
	}
	return nil
}
