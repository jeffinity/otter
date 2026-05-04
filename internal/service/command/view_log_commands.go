package command

import (
	"github.com/spf13/cobra"

	logcmd "github.com/jeffinity/otter/internal/service/command/log"
	viewcmd "github.com/jeffinity/otter/internal/service/command/view"
)

func (b *commandBuilder) viewCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "view [service]",
		Short:             "展示服务文件",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runView(cmd, args[0])
		},
	}
}

func (b *commandBuilder) runView(cmd *cobra.Command, serviceName string) error {
	out := b.deps.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}
	return viewcmd.Run(cmd.Context(), serviceName, viewcmd.Options{
		NoColor: b.deps.NoColor || b.deps.Out != nil,
	}, viewcmd.Dependencies{
		Finder: b.deps.ViewFinder,
		FS:     b.deps.FS,
		Out:    out,
	})
}

func (b *commandBuilder) logCommand() *cobra.Command {
	opts := &logOptions{}
	cmd := &cobra.Command{
		Use:                   "log service",
		Aliases:               []string{"logs"},
		Short:                 "查看服务日志",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		ValidArgsFunction:     completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runLog(cmd, args[0], opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.follow, "follow", "f", false, "实时跟踪日志（无法和 --until 同时使用）")
	cmd.Flags().IntVarP(&opts.lines, "lines", "n", 0, "日志行数（不指定时间参数时默认为 80 行，-1 代表不限制）")
	cmd.Flags().StringVarP(&opts.since, "since", "S", "", "筛选开始时间，支持确定性时间、时间偏移和一定的自然语言描述能力")
	cmd.Flags().StringVarP(&opts.until, "until", "U", "", "筛选结束时间")
	cmd.Flags().
		StringVarP(&opts.output, "output", "o", "cat", "修改 journalctl 输出模式 (short,short-full,short-iso,short-iso-precise,short-precise,short-monotonic,verbose,export,json,json-pretty,json-sse,cat,with-unit)")
	cmd.Flags().BoolVarP(&opts.pagerEnd, "pager-end", "e", false, "Immediately jump to the end in the pager")
	cmd.Flags().BoolVarP(&opts.reverse, "reverse", "r", false, "Show the newest entries first")
	cmd.Flags().BoolVarP(&opts.forceJournalctl, "force-journalctl", "F", false, "Force to use journalctl to see log for systemd unit")
	_ = cmd.Flags().MarkHidden("force-journalctl")
	_ = cmd.RegisterFlagCompletionFunc("output", logOutputCompletion)
	return cmd
}

func logOutputCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"short",
		"short-full",
		"short-iso",
		"short-iso-precise",
		"short-precise",
		"short-monotonic",
		"verbose",
		"export",
		"json",
		"json-pretty",
		"json-sse",
		"cat",
		"with-unit",
	}, cobra.ShellCompDirectiveDefault
}

func (b *commandBuilder) runLog(cmd *cobra.Command, serviceName string, opts *logOptions) error {
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
	return logcmd.Run(cmd.Context(), serviceName, logcmd.Options{
		Follow:          opts.follow,
		Lines:           opts.lines,
		Since:           opts.since,
		Until:           opts.until,
		Output:          opts.output,
		PagerEnd:        opts.pagerEnd,
		Reverse:         opts.reverse,
		ForceJournalctl: opts.forceJournalctl,
		NoColor:         b.deps.NoColor || b.deps.Out != nil,
	}, logcmd.Dependencies{
		Finder:   b.deps.LogFinder,
		Runner:   b.deps.LogRunner,
		LookPath: b.deps.LogLookPath,
		FS:       b.deps.FS,
		Out:      out,
		ErrOut:   errOut,
		In:       in,
		Now:      b.deps.Now,
	})
}
