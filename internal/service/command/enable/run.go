package enable

import (
	"context"

	startcmd "github.com/jeffinity/otter/internal/service/command/start"
)

type Options struct {
	ExcludeEnabled  bool
	IncludeDisabled bool
	Start           bool
}

type Dependencies = startcmd.Dependencies
type Runner = startcmd.Runner

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	actionOpts := startcmd.Options{
		ExcludeEnabled:  opts.ExcludeEnabled,
		IncludeDisabled: opts.IncludeDisabled,
	}
	if err := startcmd.RunAction(ctx, "enable", args, actionOpts, deps); err != nil {
		return err
	}
	if !opts.Start {
		return nil
	}
	return startcmd.RunAction(ctx, "start", args, actionOpts, deps)
}
