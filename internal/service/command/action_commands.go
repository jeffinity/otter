package command

import (
	"fmt"

	"github.com/spf13/cobra"

	disablecmd "github.com/jeffinity/otter/internal/service/command/disable"
	enablecmd "github.com/jeffinity/otter/internal/service/command/enable"
	reloadcmd "github.com/jeffinity/otter/internal/service/command/reload"
	restartcmd "github.com/jeffinity/otter/internal/service/command/restart"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
	stopcmd "github.com/jeffinity/otter/internal/service/command/stop"
)

func (b *commandBuilder) startCommand() *cobra.Command {
	opts := &actionOptions{}
	cmd := b.actionCommand("start service [services...]", "启动服务", "start", opts)
	cmd.Flags().BoolVar(&opts.reload, "reload", false, "run daemon-reload before start")
	cmd.Flags().DurationVar(&opts.stopAfter, "stop-after", 0, "在指定时间以后自动停止服务")
	cmd.Flags().BoolVarP(&opts.trace, "trace", "t", false, "在启动成功后展示日志（仅限单个服务，等同于 log -f）")
	return cmd
}

func (b *commandBuilder) stopCommand() *cobra.Command {
	return b.actionCommand("stop service [services...]", "停止服务", "stop", &actionOptions{})
}

func (b *commandBuilder) restartCommand() *cobra.Command {
	opts := &actionOptions{}
	cmd := b.actionCommand("restart service [services...]", "重启服务", "restart", opts)
	cmd.Flags().BoolVar(&opts.reload, "reload", false, "run daemon-reload before restart")
	cmd.Flags().BoolVarP(&opts.trace, "trace", "t", false, "在启动成功后展示日志（仅限单个服务，等同于 log -f）")
	return cmd
}

func (b *commandBuilder) reloadCommand() *cobra.Command {
	opts := &actionOptions{}
	cmd := &cobra.Command{
		Use:               "reload service [services...]",
		Short:             "重载服务",
		Args:              reloadArgs,
		ValidArgsFunction: completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runAction(cmd, args, "reload", opts)
		},
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	return cmd
}

func (b *commandBuilder) enableCommand() *cobra.Command {
	opts := &enableDisableOptions{}
	cmd := &cobra.Command{
		Use:                   "enable service [services...]",
		Short:                 "启用服务",
		DisableFlagsInUseLine: true,
		Args:                  actionArgs,
		ValidArgsFunction:     completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runEnable(cmd, args, opts)
		},
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	cmd.Flags().BoolVar(&opts.startAfterEnable, "start", false, "start service after enable")
	return cmd
}

func (b *commandBuilder) disableCommand() *cobra.Command {
	opts := &enableDisableOptions{}
	cmd := &cobra.Command{
		Use:                   "disable service [services...]",
		Short:                 "禁用服务",
		DisableFlagsInUseLine: true,
		Args:                  actionArgs,
		ValidArgsFunction:     completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runDisable(cmd, args, opts)
		},
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	cmd.Flags().BoolVar(&opts.stopAfterDisable, "stop", false, "stop service after disable")
	return cmd
}

func (b *commandBuilder) actionCommand(use string, short string, action string, opts *actionOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   use,
		Short:                 short,
		DisableFlagsInUseLine: true,
		Args:                  actionArgs,
		ValidArgsFunction:     completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runAction(cmd, args, action, opts)
		},
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	return cmd
}

func (b *commandBuilder) runAction(
	cmd *cobra.Command,
	args []string,
	action string,
	opts *actionOptions,
) error {
	store := b.deps.ActionStore
	if store == nil {
		store = b.deps.StatusStore
	}
	if store == nil {
		store = statuscmd.NewManagedSystemdStore(b.deps.StatusRunner, b.deps.FS)
	}
	out := b.deps.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}
	errOut := b.deps.ErrOut
	if errOut == nil {
		errOut = cmd.ErrOrStderr()
	}
	in := b.deps.In
	if in == nil {
		in = cmd.InOrStdin()
	}

	actionOpts := startcmd.Options{
		ExcludeEnabled:  opts.excludeEnabled,
		IncludeDisabled: opts.includeDisabled,
		Reload:          opts.reload,
		StopAfter:       opts.stopAfter,
		Trace:           opts.trace,
		NoColor:         b.deps.NoColor || b.deps.Out != nil,
	}
	deps := startcmd.Dependencies{
		Store:       store,
		Runner:      b.deps.ActionRunner,
		AutoStopper: b.deps.ActionAutoStopper,
		TraceRunner: b.deps.ActionTraceRunner,
		Executable:  b.deps.ActionExecutable,
		Environ:     b.deps.ActionEnviron,
		Out:         out,
		ErrOut:      errOut,
		In:          in,
	}

	switch action {
	case "start":
		return startcmd.Run(cmd.Context(), args, actionOpts, deps)
	case "stop":
		return stopcmd.Run(cmd.Context(), args, actionOpts, deps)
	case "restart":
		return restartcmd.Run(cmd.Context(), args, actionOpts, deps)
	case "reload":
		return reloadcmd.Run(cmd.Context(), args, actionOpts, deps)
	default:
		return fmt.Errorf("unknown service action %s", action)
	}
}

func (b *commandBuilder) runEnable(cmd *cobra.Command, args []string, opts *enableDisableOptions) error {
	return enablecmd.Run(cmd.Context(), args, enablecmd.Options{
		ExcludeEnabled:  opts.excludeEnabled,
		IncludeDisabled: opts.includeDisabled,
		Start:           opts.startAfterEnable,
	}, enablecmd.Dependencies(b.actionDeps(cmd)))
}

func (b *commandBuilder) runDisable(cmd *cobra.Command, args []string, opts *enableDisableOptions) error {
	return disablecmd.Run(cmd.Context(), args, disablecmd.Options{
		ExcludeEnabled:  opts.excludeEnabled,
		IncludeDisabled: opts.includeDisabled,
		Stop:            opts.stopAfterDisable,
	}, disablecmd.Dependencies(b.actionDeps(cmd)))
}

func actionArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one service should be provided")
	}
	return nil
}

func (b *commandBuilder) actionDeps(cmd *cobra.Command) startcmd.Dependencies {
	store := b.deps.ActionStore
	if store == nil {
		store = b.deps.StatusStore
	}
	if store == nil {
		store = statuscmd.NewManagedSystemdStore(b.deps.StatusRunner, b.deps.FS)
	}
	out := b.deps.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}
	errOut := b.deps.ErrOut
	if errOut == nil {
		errOut = cmd.ErrOrStderr()
	}
	in := b.deps.In
	if in == nil {
		in = cmd.InOrStdin()
	}
	return startcmd.Dependencies{
		Store:       store,
		Runner:      b.deps.ActionRunner,
		AutoStopper: b.deps.ActionAutoStopper,
		TraceRunner: b.deps.ActionTraceRunner,
		Executable:  b.deps.ActionExecutable,
		Environ:     b.deps.ActionEnviron,
		Out:         out,
		ErrOut:      errOut,
		In:          in,
	}
}

func reloadArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one service should be provided, if you want to reload service file please use `otter service daemon-reload`")
	}
	return nil
}
