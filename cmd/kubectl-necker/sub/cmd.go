package sub

import (
	"flag"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewCmd creates the root *cobra.Command of `kubectl-necker`.
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubbernecker",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
		},
	}

	config := genericclioptions.NewConfigFlags(true)
	config.AddFlags(cmd.PersistentFlags())

	klog.InitFlags(nil)
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	ctrl.SetLogger(klogr.New())

	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if f.Name != "v" { // hide all klog flags except for -v
			cmd.PersistentFlags().MarkHidden(f.Name)
		}
	})

	cmd.AddCommand(newWatchCmd(streams, config))

	return cmd
}

// rootCmd represents the base command when called without any subcommands

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer klog.Flush()
	cmd := NewCmd(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
