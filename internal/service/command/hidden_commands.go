package command

import (
	"github.com/spf13/cobra"

	auditcmd "github.com/jeffinity/otter/internal/service/command/audit"
	installcmd "github.com/jeffinity/otter/internal/service/command/install"
	selfcheckcmd "github.com/jeffinity/otter/internal/service/command/selfcheck"
	upsertclustercmd "github.com/jeffinity/otter/internal/service/command/upsertcluster"
	upsertselfcmd "github.com/jeffinity/otter/internal/service/command/upsertself"
)

func (b *commandBuilder) auditCommand() *cobra.Command {
	opts := &auditOptions{}
	cmd := &cobra.Command{
		Use:               "audit",
		Args:              cobra.NoArgs,
		ValidArgsFunction: completeEmpty,
		Hidden:            true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return auditcmd.Run(cmd.Context(), auditcmd.Options{
				ServiceName: opts.serviceName,
				ActionName:  opts.actionName,
			}, auditcmd.Dependencies{
				Writer:  b.deps.AuditWriter,
				Environ: b.deps.AuditEnviron,
				Out:     output(cmd, b.deps.Out),
				NoColor: b.deps.NoColor || b.deps.Out != nil,
				Now:     b.deps.Now,
				Path:    b.deps.AuditLogPath,
			})
		},
	}
	cmd.Flags().StringVarP(&opts.serviceName, "service-name", "s", "", "service name")
	cmd.Flags().StringVarP(&opts.actionName, "action-name", "a", "", "action name")
	return cmd
}

func (b *commandBuilder) selfCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "self-check",
		Aliases:            []string{"check"},
		Short:              "自检",
		ValidArgsFunction:  completeEmpty,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return selfcheckcmd.Run(cmd.Context(), args, selfcheckcmd.Dependencies{
				Runner:     b.deps.SelfCheckRunner,
				Executable: b.deps.SelfCheckExecutable,
				Environ:    b.deps.SelfCheckEnviron,
			})
		},
	}
}

func (b *commandBuilder) installHiddenCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "install",
		Short:              "安装一些依赖服务",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installcmd.Run(cmd.Context(), args, installcmd.Dependencies{
				Runner:     b.deps.InstallRunner,
				Executable: b.deps.InstallExecutable,
				Environ:    b.deps.InstallEnviron,
			})
		},
	}
}

func (b *commandBuilder) upsertSelfCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "upsert-self",
		Aliases:            []string{"install-self", "self-install", "self-update", "update-self", "us"},
		Short:              "将自己安装到当前服务器",
		ValidArgsFunction:  completeEmpty,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return upsertselfcmd.Run(cmd.Context(), args, upsertselfcmd.Dependencies{
				Runner:   b.deps.UpsertSelfRunner,
				Self:     b.deps.UpsertSelfPath,
				Environ:  b.deps.UpsertSelfEnviron,
				MkdirAll: b.deps.UpsertSelfMkdirAll,
			})
		},
	}
}

func (b *commandBuilder) upsertClusterCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "upsert-cluster",
		Aliases:            []string{"uc", "i-c", "ii-c", "i-cluster", "install-cluster", "update-cluster"},
		Short:              "将自己安装/更新到集群",
		ValidArgsFunction:  completeEmpty,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return upsertclustercmd.Run(cmd.Context(), args, upsertclustercmd.Dependencies{
				Runner:   b.deps.UpsertClusterRunner,
				Self:     b.deps.UpsertClusterPath,
				Environ:  b.deps.UpsertClusterEnv,
				MkdirAll: b.deps.UpsertClusterMkdir,
			})
		},
	}
}
