package cobwrap

import (
	"context"

	"github.com/spf13/cobra"
)

type fillable interface {
	getParent() fillable
	fill(cmd *cobra.Command, args []string) error
}

type Command[T Option] struct {
	Command *cobra.Command
	Options T

	parent fillable
}

type Option interface {
	Fill(cmd *cobra.Command, args []string) error
	Run(cmd *cobra.Command, args []string) error
}

func GetOpt[T any](cmd *cobra.Command) T {
	v := cmd.Context().Value(*new(*T))
	if v == nil {
		return *new(T)
	}
	return v.(T)
}

func (w Command[T]) getParent() fillable {
	return w.parent
}

func (w Command[T]) fill(cmd *cobra.Command, args []string) error {
	err := w.Options.Fill(cmd, args)
	if err != nil {
		return err
	}
	ctx := context.WithValue(cmd.Context(), *new(*T), w.Options)
	cmd.SetContext(ctx)
	return nil
}

func AddCommand[P, S Option](parent *Command[P], sub *Command[S]) {
	sub.parent = parent

	if sub.Command.RunE != nil || sub.Command.Run != nil {
		panic("error")
	}

	sub.Command.RunE = func(cmd *cobra.Command, args []string) error {
		var parents []fillable
		for p := sub.getParent(); p != nil; p = p.getParent() {
			parents = append(parents, p)
		}
		for i := len(parents) - 1; i >= 0; i-- {
			err := parents[i].fill(cmd, args)
			if err != nil {
				return err
			}
		}
		err := sub.Options.Fill(cmd, args)
		if err != nil {
			return err
		}
		return sub.Options.Run(cmd, args)
	}

	parent.Command.AddCommand(sub.Command)
}

func (w Command[T]) Execute() error {
	return w.Command.Execute()
}
