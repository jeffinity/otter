package servicecmd

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

const UnsupportedPlatformMessage = "otter service is only supported on linux"

var ErrUnsupportedPlatform = errors.New(UnsupportedPlatformMessage)

type Dependencies struct {
	RuntimeOS string
}

type filterOptions struct {
	excludeEnabled  bool
	includeDisabled bool
	onlyPackage     bool
	onlyClassic     bool
}

type statusOptions struct {
	filterOptions
	includeTimeInfo bool
	sortAsc         bool
	sortDesc        bool
	since           time.Duration
	noMono          bool
}

type listOptions struct {
	filterOptions
	oneLine bool
}

type detailOptions struct {
	filterOptions
	noPager bool
}

type logOptions struct {
	follow          bool
	lines           int
	since           string
	until           string
	output          string
	pagerEnd        bool
	reverse         bool
	forceJournalctl bool
}

type actionOptions struct {
	filterOptions
	reload    bool
	trace     bool
	stopAfter time.Duration
}

type enableDisableOptions struct {
	filterOptions
	startAfterEnable bool
	stopAfterDisable bool
}

type installOptions struct {
	name     string
	noEnable bool
	noStart  bool
}

type installCommandOptions struct {
	installOptions
	workingDirectory string
	noInstall        bool
}

type installDockerComposeOptions struct {
	installOptions
	baseDir string
	force   bool
}

type groupListOptions struct {
	oneLine         bool
	includeServices bool
}

type reGenerateOptions struct {
	restart    bool
	notRestart bool
}

type auditOptions struct {
	serviceName string
	actionName  string
}

type commandBuilder struct {
	deps Dependencies
}

func New(deps Dependencies) *cobra.Command {
	if deps.RuntimeOS == "" {
		deps.RuntimeOS = runtime.GOOS
	}
	b := &commandBuilder{deps: deps}
	return b.serviceCommand()
}

func (b *commandBuilder) serviceCommand() *cobra.Command {
	var clusterMode bool

	cmd := &cobra.Command{
		Use:                   "service (status) <services...>",
		Short:                 "Manage otter services",
		TraverseChildren:      true,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     completeServices,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if b.deps.RuntimeOS != "linux" {
				return ErrUnsupportedPlatform
			}
			if clusterMode {
				return notImplemented(cmd, nil)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if arg == "--help" || arg == "-h" {
					return cmd.Help()
				}
			}
			statusCmd, _, err := cmd.Find([]string{"status"})
			if err != nil {
				return err
			}
			return notImplemented(statusCmd, nil)
		},
	}

	cmd.PersistentFlags().BoolVarP(&clusterMode, "cluster", "c", false, "集群模式")

	b.addServiceCommands(cmd)

	return cmd
}

func (b *commandBuilder) addServiceCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		b.statusCommand(),
		b.listCommand(),
		b.detailCommand(),
		b.showPropertyCommand(),
		b.viewCommand(),
		b.logCommand(),
		b.showPidsCommand(),
		b.showPortsCommand(),
		b.startCommand(),
		b.stopCommand(),
		b.restartCommand(),
		b.reloadCommand(),
		b.enableCommand(),
		b.disableCommand(),
		b.daemonReloadCommand(),
		b.groupListCommand(),
		b.groupStartCommand(),
		b.groupStopCommand(),
		b.groupRestartCommand(),
		b.installServiceCommand(),
		b.installCommandCommand(),
		b.installDockerComposeCommand(),
		b.linkServiceCommand(),
		b.editCommand(),
		b.reGenerateCommand(),
		b.auditCommand(),
		b.selfCheckCommand(),
		b.installHiddenCommand(),
		b.upsertSelfCommand(),
		b.upsertClusterCommand(),
	)
}

func (b *commandBuilder) statusCommand() *cobra.Command {
	opts := &statusOptions{}
	cmd := &cobra.Command{
		Use:               "status [services...]",
		Short:             "查看服务状态",
		ValidArgsFunction: completeServices,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.sortAsc && opts.sortDesc {
				return fmt.Errorf("--asc and --desc cannot be apply in the meantime")
			}
			return nil
		},
		RunE: notImplemented,
	}
	addFilterFlags(cmd, &opts.filterOptions)
	cmd.Flags().BoolVarP(&opts.includeTimeInfo, "time-info", "t", false, "显示时间信息")
	cmd.Flags().BoolVar(&opts.sortAsc, "asc", false, "按照变化时间距离当前的时间差升序排列")
	cmd.Flags().BoolVar(&opts.sortDesc, "desc", false, "按照变化时间距离当前的时间差倒序排列")
	cmd.Flags().DurationVarP(&opts.since, "since", "s", 0, "仅展示距当前多久之内有过启动/停止的服务")
	cmd.Flags().BoolVarP(&opts.noMono, "no-mono", "M", false, "不使用 monotonic 时间")
	return cmd
}

func (b *commandBuilder) listCommand() *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:               "list [services...]",
		Short:             "列出服务信息",
		ValidArgsFunction: completeServices,
		RunE:              notImplemented,
	}
	cmd.Flags().BoolVarP(&opts.oneLine, "one", "1", false, "在一行中展示")
	addFilterFlags(cmd, &opts.filterOptions)
	return cmd
}

func (b *commandBuilder) detailCommand() *cobra.Command {
	opts := &detailOptions{}
	cmd := &cobra.Command{
		Use:                   "detail service [services...]",
		Short:                 "获取服务的详细信息",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		ValidArgsFunction:     completeServices,
		RunE:                  notImplemented,
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	cmd.Flags().BoolVarP(&opts.noPager, "no-pager", "P", false, "不使用翻页")
	return cmd
}

func (b *commandBuilder) showPropertyCommand() *cobra.Command {
	var showAll bool
	cmd := &cobra.Command{
		Use:               "show-property [service]",
		Aliases:           []string{"show"},
		Short:             "查看服务参数",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE:              notImplemented,
	}
	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "展示所有参数")
	return cmd
}

func (b *commandBuilder) viewCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "view [service]",
		Short:             "展示服务文件",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE:              notImplemented,
	}
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
		RunE:                  notImplemented,
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
	return cmd
}

func (b *commandBuilder) showPidsCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "show-pids [services...]",
		Aliases:           []string{"show-pid", "pids", "pid"},
		Short:             "查看服务对应的进程的 PID",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeServices,
		RunE:              notImplemented,
	}
}

func (b *commandBuilder) showPortsCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "show-ports [services...]",
		Aliases:           []string{"show-port", "ports", "port"},
		Short:             "查看服务对应的进程的监听端口",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeServices,
		RunE:              notImplemented,
	}
}

func (b *commandBuilder) startCommand() *cobra.Command {
	opts := &actionOptions{}
	cmd := actionCommand("start service [services...]", "启动服务", opts)
	cmd.Flags().BoolVar(&opts.reload, "reload", false, "run daemon-reload before start")
	cmd.Flags().DurationVar(&opts.stopAfter, "stop-after", 0, "在指定时间以后自动停止服务")
	cmd.Flags().BoolVarP(&opts.trace, "trace", "t", false, "在启动成功后展示日志（仅限单个服务，等同于 log -f）")
	return cmd
}

func (b *commandBuilder) stopCommand() *cobra.Command {
	return actionCommand("stop service [services...]", "停止服务", &actionOptions{})
}

func (b *commandBuilder) restartCommand() *cobra.Command {
	opts := &actionOptions{}
	cmd := actionCommand("restart service [services...]", "重启服务", opts)
	cmd.Flags().BoolVar(&opts.reload, "reload", false, "run daemon-reload before restart")
	cmd.Flags().BoolVarP(&opts.trace, "trace", "t", false, "在启动成功后展示日志（仅限单个服务，等同于 log -f）")
	return cmd
}

func (b *commandBuilder) reloadCommand() *cobra.Command {
	opts := &actionOptions{}
	cmd := &cobra.Command{
		Use:               "reload service [services...]",
		Short:             "重载服务",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeServices,
		RunE:              notImplemented,
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	return cmd
}

func (b *commandBuilder) enableCommand() *cobra.Command {
	opts := &enableDisableOptions{}
	cmd := enableDisableCommand("enable service [services...]", "启用服务", opts)
	cmd.Flags().BoolVar(&opts.startAfterEnable, "start", false, "start service after enable")
	return cmd
}

func (b *commandBuilder) disableCommand() *cobra.Command {
	opts := &enableDisableOptions{}
	cmd := enableDisableCommand("disable service [services...]", "禁用服务", opts)
	cmd.Flags().BoolVar(&opts.stopAfterDisable, "stop", false, "stop service after disable")
	return cmd
}

func (b *commandBuilder) daemonReloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "daemon-reload",
		Short:             "重载系统 service 树",
		Args:              cobra.NoArgs,
		ValidArgsFunction: completeEmpty,
		RunE:              notImplemented,
	}
}

func (b *commandBuilder) groupListCommand() *cobra.Command {
	opts := &groupListOptions{}
	cmd := &cobra.Command{
		Use:                   "group-list [services...]",
		Aliases:               []string{"list-group"},
		Short:                 "列出组信息",
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     completeServices,
		RunE:                  notImplemented,
	}
	cmd.Flags().BoolVarP(&opts.oneLine, "one", "1", false, "在一行中展示")
	cmd.Flags().BoolVarP(&opts.includeServices, "services", "s", false, "同时展示包含的服务名称")
	return cmd
}

func (b *commandBuilder) groupStartCommand() *cobra.Command {
	opts := &actionOptions{}
	cmd := groupActionCommand("group-start group [groups...]", []string{"start-group"}, "启动一组服务")
	cmd.Flags().DurationVar(&opts.stopAfter, "stop-after", 0, "在指定时间以后自动停止服务")
	return cmd
}

func (b *commandBuilder) groupStopCommand() *cobra.Command {
	return groupActionCommand("group-stop group [groups...]", []string{"stop-group"}, "停止一组服务")
}

func (b *commandBuilder) groupRestartCommand() *cobra.Command {
	return groupActionCommand("group-restart group [groups...]", []string{"restart-group"}, "重启一组服务")
}

func (b *commandBuilder) installServiceCommand() *cobra.Command {
	opts := &installOptions{}
	cmd := &cobra.Command{
		Use:     "install-service <file>",
		Aliases: []string{"iiiii"},
		Short:   "安装一个 service 文件，接收一个参数为文件路径",
		Args:    cobra.ExactArgs(1),
		RunE:    notImplemented,
	}
	addInstallFlags(cmd, opts)
	return cmd
}

func (b *commandBuilder) installCommandCommand() *cobra.Command {
	opts := &installCommandOptions{}
	cmd := &cobra.Command{
		Use:   "install-command -n service_name -- command...",
		Short: "将一个 command 生成为 service 并安装",
		Args:  cobra.MinimumNArgs(1),
		RunE:  notImplemented,
	}
	addInstallFlags(cmd, &opts.installOptions)
	cmd.Flags().StringVarP(&opts.workingDirectory, "wd", "w", "", "指定工作目录，- 代表使用当前目录")
	cmd.Flags().BoolVarP(&opts.noInstall, "no-install", "I", false, "仅展示生成的服务（不安装）")
	return cmd
}

func (b *commandBuilder) installDockerComposeCommand() *cobra.Command {
	opts := &installDockerComposeOptions{}
	cmd := &cobra.Command{
		Use:     "exp-install-docker-compose [file]",
		Aliases: []string{"install-docker-compose", "idc"},
		Short:   "安装一个 docker-compose 文件作为 service",
		Args:    cobra.MaximumNArgs(1),
		RunE:    notImplemented,
	}
	addInstallFlags(cmd, &opts.installOptions)
	cmd.Flags().StringVarP(&opts.baseDir, "dir", "d", "", "服务的基础路径，默认为 /opt/服务名")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "强制覆盖基础路径中已经存在的 docker-compose.yml 文件")
	return cmd
}

func (b *commandBuilder) linkServiceCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "link-service <service>",
		Aliases: []string{"link", "install-fake", "fake", "install-fake-service"},
		Short:   "将当前已经存在的 service 纳入 otter service 管理",
		Args:    cobra.ExactArgs(1),
		RunE:    notImplemented,
	}
}

func (b *commandBuilder) editCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "edit <service>",
		Short:             "编辑服务",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE:              notImplemented,
	}
}

func (b *commandBuilder) reGenerateCommand() *cobra.Command {
	opts := &reGenerateOptions{}
	cmd := &cobra.Command{
		Use:               "re-generate <service>",
		Aliases:           []string{"regen"},
		Short:             "刷新指定 service 文件的配置",
		Hidden:            true,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE:              notImplemented,
	}
	cmd.Flags().BoolVarP(&opts.restart, "restart", "r", false, "restart service after regen")
	cmd.Flags().BoolVarP(&opts.notRestart, "not-restart", "R", false, "do not restart service after regen")
	return cmd
}

func (b *commandBuilder) auditCommand() *cobra.Command {
	opts := &auditOptions{}
	cmd := &cobra.Command{
		Use:               "audit",
		Args:              cobra.NoArgs,
		ValidArgsFunction: completeEmpty,
		Hidden:            true,
		RunE:              notImplemented,
	}
	cmd.Flags().StringVarP(&opts.serviceName, "service-name", "s", "", "service name")
	cmd.Flags().StringVarP(&opts.actionName, "action-name", "a", "", "action name")
	_ = cmd.MarkFlagRequired("service-name")
	_ = cmd.MarkFlagRequired("action-name")
	return cmd
}

func (b *commandBuilder) selfCheckCommand() *cobra.Command {
	return hiddenPassThroughCommand("self-check", []string{"check"}, "自检")
}

func (b *commandBuilder) installHiddenCommand() *cobra.Command {
	return hiddenPassThroughCommand("install", nil, "安装一些依赖服务")
}

func (b *commandBuilder) upsertSelfCommand() *cobra.Command {
	return hiddenPassThroughCommand("upsert-self", []string{"install-self", "self-install", "self-update", "update-self", "us"}, "将自己安装到当前服务器")
}

func (b *commandBuilder) upsertClusterCommand() *cobra.Command {
	return hiddenPassThroughCommand("upsert-cluster", []string{"uc", "i-c", "ii-c", "i-cluster", "install-cluster", "update-cluster"}, "将自己安装/更新到集群")
}

func addEnabledFilterFlags(cmd *cobra.Command, opts *filterOptions) {
	cmd.Flags().BoolVarP(&opts.excludeEnabled, "no-enabled", "E", false, "不包含启用的服务列表")
	cmd.Flags().BoolVarP(&opts.includeDisabled, "disabled", "d", false, "包含未启用的服务列表")
}

func addFilterFlags(cmd *cobra.Command, opts *filterOptions) {
	addEnabledFilterFlags(cmd, opts)
	cmd.Flags().BoolVar(&opts.onlyPackage, "only-package", false, "仅包含来源于 package 的服务")
	cmd.Flags().BoolVar(&opts.onlyClassic, "only-classic", false, "仅包含来源非 package 的服务")
}

func addInstallFlags(cmd *cobra.Command, opts *installOptions) {
	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "服务名")
	cmd.Flags().BoolVarP(&opts.noEnable, "no-enable", "E", false, "不 enable 服务")
	cmd.Flags().BoolVarP(&opts.noStart, "no-start", "S", false, "不 start 服务")
}

func actionCommand(use string, short string, opts *actionOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   use,
		Short:                 short,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		ValidArgsFunction:     completeServices,
		RunE:                  notImplemented,
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	return cmd
}

func enableDisableCommand(use string, short string, opts *enableDisableOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   use,
		Short:                 short,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		ValidArgsFunction:     completeServices,
		RunE:                  notImplemented,
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	return cmd
}

func groupActionCommand(use string, aliases []string, short string) *cobra.Command {
	return &cobra.Command{
		Use:                   use,
		Aliases:               aliases,
		Short:                 short,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		ValidArgsFunction:     completeGroups,
		RunE:                  notImplemented,
	}
}

func hiddenPassThroughCommand(use string, aliases []string, short string) *cobra.Command {
	return &cobra.Command{
		Use:                use,
		Aliases:            aliases,
		Short:              short,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE:               notImplemented,
	}
}

func completeEmpty(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func completeServices(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func completeGroups(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func notImplemented(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("otter service %s is not implemented yet", cmd.Name())
}
