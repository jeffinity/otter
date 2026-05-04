package stop

import (
	"context"

	startcmd "github.com/jeffinity/otter/internal/service/command/start"
)

type Options = startcmd.Options
type Dependencies = startcmd.Dependencies
type Runner = startcmd.Runner
type AutoStopper = startcmd.AutoStopper
type TraceRunner = startcmd.TraceRunner

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	return startcmd.RunAction(ctx, "stop", args, opts, deps)
}
