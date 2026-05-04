package grouprestart

import (
	"context"

	groupstartcmd "github.com/jeffinity/otter/internal/service/command/groupstart"
)

type Options = groupstartcmd.Options
type Dependencies = groupstartcmd.Dependencies

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	return groupstartcmd.RunAction(ctx, "restart", args, opts, deps)
}
