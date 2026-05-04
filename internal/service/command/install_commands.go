package command

import (
	"github.com/spf13/cobra"

	installcommandcmd "github.com/jeffinity/otter/internal/service/command/installcommand"
	installdockercomposecmd "github.com/jeffinity/otter/internal/service/command/installdockercompose"
	installservicecmd "github.com/jeffinity/otter/internal/service/command/installservice"
	linkservicecmd "github.com/jeffinity/otter/internal/service/command/linkservice"
)

func (b *commandBuilder) installServiceCommandReal() *cobra.Command {
	opts := &installOptions{}
	cmd := &cobra.Command{
		Use:     "install-service <file>",
		Aliases: []string{"iiiii"},
		Short:   "安装一个 service 文件，接收一个参数为文件路径",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installservicecmd.Run(cmd.Context(), args[0], installservicecmd.Options{
				Name:     opts.name,
				NoEnable: opts.noEnable,
				NoStart:  opts.noStart,
			}, b.installServiceDeps(cmd))
		},
	}
	addInstallFlags(cmd, opts)
	return cmd
}

func (b *commandBuilder) installCommandCommandReal() *cobra.Command {
	opts := &installCommandOptions{}
	cmd := &cobra.Command{
		Use:   "install-command -n service_name -- command...",
		Short: "将一个 command 生成为 service 并安装",
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.name == "" {
				return errServiceNameRequired()
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return installcommandcmd.Run(cmd.Context(), args, installcommandcmd.Options{
				Name:             opts.name,
				WorkingDirectory: opts.workingDirectory,
				NoInstall:        opts.noInstall,
				NoEnable:         opts.noEnable,
				NoStart:          opts.noStart,
			}, installcommandcmd.Dependencies{
				Installer: b.deps.InstallSvcInstaller,
				FS:        b.deps.FS,
				Runner:    b.deps.ActionRunner,
				LookPath:  b.deps.InstallCmdLookPath,
				Getwd:     b.deps.InstallCmdGetwd,
				Out:       output(cmd, b.deps.Out),
				ErrOut:    errOutput(cmd, b.deps.ErrOut),
				In:        input(cmd, b.deps.In),
			})
		},
	}
	addInstallFlags(cmd, &opts.installOptions)
	cmd.Flags().StringVarP(&opts.workingDirectory, "wd", "w", "", "指定工作目录，- 代表使用当前目录")
	cmd.Flags().BoolVarP(&opts.noInstall, "no-install", "I", false, "仅展示生成的服务（不安装）")
	return cmd
}

func (b *commandBuilder) installDockerComposeCommandReal() *cobra.Command {
	opts := &installDockerComposeOptions{}
	cmd := &cobra.Command{
		Use:     "exp-install-docker-compose [file]",
		Aliases: []string{"install-docker-compose", "idc"},
		Short:   "安装一个 docker-compose 文件作为 service",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installdockercomposecmd.Run(cmd.Context(), args, installdockercomposecmd.Options{
				Name:     opts.name,
				BaseDir:  opts.baseDir,
				Force:    opts.force,
				NoEnable: opts.noEnable,
				NoStart:  opts.noStart,
			}, installdockercomposecmd.Dependencies{
				Installer: b.deps.InstallSvcInstaller,
				FS:        b.deps.FS,
				Runner:    b.deps.ActionRunner,
				LookPath:  b.deps.InstallDCLookPath,
				Getwd:     b.deps.InstallDCGetwd,
				Out:       output(cmd, b.deps.Out),
				ErrOut:    errOutput(cmd, b.deps.ErrOut),
				In:        input(cmd, b.deps.In),
			})
		},
	}
	addInstallFlags(cmd, &opts.installOptions)
	cmd.Flags().StringVarP(&opts.baseDir, "dir", "d", "", "服务的基础路径，默认为 /opt/服务名")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "强制覆盖基础路径中已经存在的 docker-compose.yml 文件")
	return cmd
}

func (b *commandBuilder) linkServiceCommandReal() *cobra.Command {
	return &cobra.Command{
		Use:     "link-service <service>",
		Aliases: []string{"link", "install-fake", "fake", "install-fake-service"},
		Short:   "将当前已经存在的 service 纳入 otter service 管理",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := b.deps.LinkStore
			if store == nil {
				store = b.deps.StatusStore
			}
			return linkservicecmd.Run(cmd.Context(), args[0], linkservicecmd.Dependencies{
				Store: store,
				FS:    b.deps.FS,
				Out:   output(cmd, b.deps.Out),
			})
		},
	}
}

func (b *commandBuilder) installServiceDeps(cmd *cobra.Command) installservicecmd.Dependencies {
	return installservicecmd.Dependencies{
		Installer: b.deps.InstallSvcInstaller,
		FS:        b.deps.FS,
		Runner:    b.deps.ActionRunner,
		Out:       output(cmd, b.deps.Out),
		ErrOut:    errOutput(cmd, b.deps.ErrOut),
		In:        input(cmd, b.deps.In),
	}
}
