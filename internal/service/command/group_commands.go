package command

import (
	"fmt"

	"github.com/spf13/cobra"

	grouprestartcmd "github.com/jeffinity/otter/internal/service/command/grouprestart"
	groupstartcmd "github.com/jeffinity/otter/internal/service/command/groupstart"
	groupstopcmd "github.com/jeffinity/otter/internal/service/command/groupstop"
)

func (b *commandBuilder) groupActionCommand(use string, aliases []string, short string, action string) *cobra.Command {
	opts := &actionOptions{}
	cmd := &cobra.Command{
		Use:                   use,
		Aliases:               aliases,
		Short:                 short,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		ValidArgsFunction:     completeGroups,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runGroupAction(cmd, args, action, opts)
		},
	}
	if action == "start" {
		cmd.Flags().DurationVar(&opts.stopAfter, "stop-after", 0, "在指定时间以后自动停止服务")
	}
	return cmd
}

func (b *commandBuilder) runGroupAction(
	cmd *cobra.Command,
	args []string,
	action string,
	opts *actionOptions,
) error {
	actionDeps := b.actionDeps(cmd)
	deps := groupstartcmd.Dependencies{
		GroupStore:  b.deps.GroupListStore,
		FS:          b.deps.FS,
		ActionStore: actionDeps.Store,
		Runner:      actionDeps.Runner,
		AutoStopper: actionDeps.AutoStopper,
		TraceRunner: actionDeps.TraceRunner,
		Executable:  actionDeps.Executable,
		Environ:     actionDeps.Environ,
		Out:         actionDeps.Out,
		ErrOut:      actionDeps.ErrOut,
		In:          actionDeps.In,
	}
	switch action {
	case "start":
		return groupstartcmd.Run(cmd.Context(), args, groupstartcmd.Options{StopAfter: opts.stopAfter}, deps)
	case "stop":
		return groupstopcmd.Run(cmd.Context(), args, groupstopcmd.Options{}, groupstopcmd.Dependencies(deps))
	case "restart":
		return grouprestartcmd.Run(cmd.Context(), args, grouprestartcmd.Options{}, grouprestartcmd.Dependencies(deps))
	default:
		return fmt.Errorf("unknown group action %s", action)
	}
}
