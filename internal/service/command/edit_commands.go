package command

import (
	"github.com/spf13/cobra"

	editcmd "github.com/jeffinity/otter/internal/service/command/edit"
	regeneratecmd "github.com/jeffinity/otter/internal/service/command/regenerate"
)

func (b *commandBuilder) editCommandReal() *cobra.Command {
	return &cobra.Command{
		Use:               "edit <service>",
		Short:             "编辑服务",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return editcmd.Run(cmd.Context(), args[0], editcmd.Dependencies{
				Finder:       b.deps.EditFinder,
				PackageFind:  b.deps.EditPackageFinder,
				Regenerator:  b.deps.EditRegenerator,
				Prompter:     b.deps.EditPrompter,
				Runner:       b.deps.EditRunner,
				SystemRunner: b.deps.EditSystemRunner,
				LookPath:     b.deps.EditLookPath,
				Getenv:       b.deps.EditGetenv,
				TempFile:     b.deps.EditTempFile,
				FS:           b.deps.FS,
				Out:          output(cmd, b.deps.Out),
				ErrOut:       errOutput(cmd, b.deps.ErrOut),
				In:           input(cmd, b.deps.In),
			})
		},
	}
}

func (b *commandBuilder) reGenerateCommandReal() *cobra.Command {
	opts := &reGenerateOptions{}
	cmd := &cobra.Command{
		Use:               "re-generate <service>",
		Aliases:           []string{"regen"},
		Short:             "刷新指定 service 文件的配置",
		Hidden:            true,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return regeneratecmd.Run(cmd.Context(), args[0], regeneratecmd.Options{
				Restart:    opts.restart,
				NotRestart: opts.notRestart,
			}, regeneratecmd.Dependencies{
				Finder:    b.deps.ReGenerateFinder,
				Generator: b.deps.ReGenerateGenerator,
				Prompter:  b.deps.ReGeneratePrompter,
				FS:        b.deps.FS,
				Runner:    b.deps.ActionRunner,
				Out:       output(cmd, b.deps.Out),
				ErrOut:    errOutput(cmd, b.deps.ErrOut),
				In:        input(cmd, b.deps.In),
			})
		},
	}
	cmd.Flags().BoolVarP(&opts.restart, "restart", "r", false, "restart service after regen")
	cmd.Flags().BoolVarP(&opts.notRestart, "not-restart", "R", false, "do not restart service after regen")
	return cmd
}
