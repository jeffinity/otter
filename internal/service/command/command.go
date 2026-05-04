package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffinity/otter/internal/otterfs"
	auditcmd "github.com/jeffinity/otter/internal/service/command/audit"
	daemonreloadcmd "github.com/jeffinity/otter/internal/service/command/daemonreload"
	detailcmd "github.com/jeffinity/otter/internal/service/command/detail"
	editcmd "github.com/jeffinity/otter/internal/service/command/edit"
	grouplistcmd "github.com/jeffinity/otter/internal/service/command/grouplist"
	installcmd "github.com/jeffinity/otter/internal/service/command/install"
	installservicecmd "github.com/jeffinity/otter/internal/service/command/installservice"
	listcmd "github.com/jeffinity/otter/internal/service/command/list"
	logcmd "github.com/jeffinity/otter/internal/service/command/log"
	regeneratecmd "github.com/jeffinity/otter/internal/service/command/regenerate"
	selfcheckcmd "github.com/jeffinity/otter/internal/service/command/selfcheck"
	"github.com/jeffinity/otter/internal/service/command/servicefile"
	showpidscmd "github.com/jeffinity/otter/internal/service/command/showpids"
	showportscmd "github.com/jeffinity/otter/internal/service/command/showports"
	showpropertycmd "github.com/jeffinity/otter/internal/service/command/showproperty"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
	upsertclustercmd "github.com/jeffinity/otter/internal/service/command/upsertcluster"
	upsertselfcmd "github.com/jeffinity/otter/internal/service/command/upsertself"
)

const UnsupportedPlatformMessage = "otter service is only supported on linux"

var ErrUnsupportedPlatform = errors.New(UnsupportedPlatformMessage)

type Dependencies struct {
	RuntimeOS           string
	FS                  otterfs.Provider
	StatusStore         statuscmd.Store
	ListStore           statuscmd.Store
	DetailStore         statuscmd.Store
	StatusRunner        statuscmd.Runner
	DetailRunner        detailcmd.Runner
	ShowPropertyGetter  showpropertycmd.Getter
	ShowPropertyRunner  showpropertycmd.Runner
	DaemonReloadRunner  daemonreloadcmd.Runner
	ActionStore         statuscmd.Store
	ActionRunner        startcmd.Runner
	ActionAutoStopper   startcmd.AutoStopper
	ActionTraceRunner   startcmd.TraceRunner
	ActionExecutable    func() (string, error)
	ActionEnviron       func() []string
	GroupListStore      grouplistcmd.Store
	ShowPidsFinder      showpidscmd.Finder
	ShowPortsPidFinder  showpidscmd.Finder
	ShowPortsConnFinder showportscmd.ConnFinder
	SelfCheckRunner     selfcheckcmd.Runner
	SelfCheckExecutable func() (string, error)
	SelfCheckEnviron    func() []string
	InstallRunner       installcmd.Runner
	InstallExecutable   func() (string, error)
	InstallEnviron      func() []string
	InstallSvcInstaller installservicecmd.Installer
	InstallCmdLookPath  func(string) (string, error)
	InstallCmdGetwd     func() (string, error)
	InstallDCGetwd      func() (string, error)
	InstallDCLookPath   func(string) (string, error)
	LinkStore           statuscmd.Store
	EditFinder          servicefile.Finder
	EditPackageFinder   regeneratecmd.Finder
	EditRegenerator     regeneratecmd.Generator
	EditPrompter        editcmd.Prompter
	EditRunner          editcmd.Runner
	EditSystemRunner    startcmd.Runner
	EditLookPath        func(string) (string, error)
	EditGetenv          func(string) string
	EditTempFile        func(pattern string) (*os.File, error)
	ReGenerateFinder    regeneratecmd.Finder
	ReGenerateGenerator regeneratecmd.Generator
	ReGeneratePrompter  regeneratecmd.Prompter
	AuditWriter         auditcmd.Writer
	AuditEnviron        func() []string
	AuditLogPath        string
	UpsertSelfRunner    upsertselfcmd.Runner
	UpsertSelfPath      func() string
	UpsertSelfEnviron   func() []string
	UpsertSelfMkdirAll  func(path string, perm os.FileMode) error
	UpsertClusterRunner upsertclustercmd.Runner
	UpsertClusterPath   func() string
	UpsertClusterEnv    func() []string
	UpsertClusterMkdir  func(path string, perm os.FileMode) error
	ViewFinder          servicefile.Finder
	LogFinder           servicefile.Finder
	LogRunner           logcmd.Runner
	LogLookPath         func(file string) (string, error)
	Out                 io.Writer
	ErrOut              io.Writer
	In                  io.Reader
	NoColor             bool
	Now                 func() time.Time
	MonoNow             func() int64
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

type showPropertyOptions struct {
	showAll bool
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
	if deps.FS.Config().SystemdServicePath == "" {
		deps.FS = otterfs.Default()
	}
	b := &commandBuilder{deps: deps}
	return b.serviceCommand()
}

func (b *commandBuilder) serviceCommand() *cobra.Command {
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
			opts, ok := statusCmdOptions(statusCmd)
			if !ok {
				return notImplemented(statusCmd, nil)
			}
			return b.runStatus(cmd, args, opts)
		},
	}

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
			if opts.onlyPackage && opts.onlyClassic {
				return fmt.Errorf("--only-package and --only-classic cannot be apply in the meantime")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runStatus(cmd, args, opts)
		},
	}
	addFilterFlags(cmd, &opts.filterOptions)
	cmd.Flags().BoolVarP(&opts.includeTimeInfo, "time-info", "t", false, "显示时间信息")
	cmd.Flags().BoolVar(&opts.sortAsc, "asc", false, "按照变化时间距离当前的时间差升序排列")
	cmd.Flags().BoolVar(&opts.sortDesc, "desc", false, "按照变化时间距离当前的时间差倒序排列")
	cmd.Flags().DurationVarP(&opts.since, "since", "s", 0, "仅展示距当前多久之内有过启动/停止的服务")
	cmd.Flags().BoolVarP(&opts.noMono, "no-mono", "M", false, "不使用 monotonic 时间")
	cmd.Annotations = map[string]string{"status-options": "true"}
	return cmd
}

func statusCmdOptions(cmd *cobra.Command) (*statusOptions, bool) {
	return &statusOptions{}, cmd.Annotations["status-options"] == "true"
}

func (b *commandBuilder) runStatus(cmd *cobra.Command, args []string, opts *statusOptions) error {
	store := b.deps.StatusStore
	if store == nil {
		store = statuscmd.NewManagedSystemdStore(b.deps.StatusRunner, b.deps.FS)
	}
	out := b.deps.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}
	return statuscmd.Run(cmd.Context(), args, statuscmd.Options{
		ExcludeEnabled:  opts.excludeEnabled,
		IncludeDisabled: opts.includeDisabled,
		OnlyPackage:     opts.onlyPackage,
		OnlyClassic:     opts.onlyClassic,
		IncludeTimeInfo: opts.includeTimeInfo,
		SortAsc:         opts.sortAsc,
		SortDesc:        opts.sortDesc,
		Since:           opts.since,
		NoMono:          opts.noMono,
		NoColor:         b.deps.NoColor,
	}, statuscmd.Dependencies{
		Store:   store,
		Out:     out,
		Now:     b.deps.Now,
		MonoNow: b.deps.MonoNow,
	})
}

func (b *commandBuilder) listCommand() *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:               "list [services...]",
		Short:             "列出服务信息",
		ValidArgsFunction: completeServices,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.onlyPackage && opts.onlyClassic {
				return fmt.Errorf("--only-package and --only-classic cannot be apply in the meantime")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runList(cmd, args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.oneLine, "one", "1", false, "在一行中展示")
	addFilterFlags(cmd, &opts.filterOptions)
	return cmd
}

func (b *commandBuilder) runList(cmd *cobra.Command, args []string, opts *listOptions) error {
	store := b.deps.ListStore
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
	return listcmd.Run(cmd.Context(), args, listcmd.Options{
		OneLine:         opts.oneLine,
		ExcludeEnabled:  opts.excludeEnabled,
		IncludeDisabled: opts.includeDisabled,
		OnlyPackage:     opts.onlyPackage,
		OnlyClassic:     opts.onlyClassic,
	}, listcmd.Dependencies{
		Store: store,
		Out:   out,
	})
}

func (b *commandBuilder) detailCommand() *cobra.Command {
	opts := &detailOptions{}
	cmd := &cobra.Command{
		Use:                   "detail service [services...]",
		Short:                 "获取服务的详细信息",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		ValidArgsFunction:     completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runDetail(cmd, args, opts)
		},
	}
	addEnabledFilterFlags(cmd, &opts.filterOptions)
	cmd.Flags().BoolVarP(&opts.noPager, "no-pager", "P", false, "不使用翻页")
	return cmd
}

func (b *commandBuilder) runDetail(cmd *cobra.Command, args []string, opts *detailOptions) error {
	store := b.deps.DetailStore
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
	return detailcmd.Run(cmd.Context(), args, detailcmd.Options{
		ExcludeEnabled:  opts.excludeEnabled,
		IncludeDisabled: opts.includeDisabled,
		NoPager:         opts.noPager,
	}, detailcmd.Dependencies{
		Store:  store,
		Runner: b.deps.DetailRunner,
		Out:    out,
		ErrOut: errOut,
		In:     in,
	})
}

func (b *commandBuilder) showPropertyCommand() *cobra.Command {
	opts := &showPropertyOptions{}
	cmd := &cobra.Command{
		Use:               "show-property [service]",
		Aliases:           []string{"show"},
		Short:             "查看服务参数",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runShowProperty(cmd, args[0], opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.showAll, "all", "a", false, "展示所有参数")
	return cmd
}

func (b *commandBuilder) runShowProperty(cmd *cobra.Command, serviceName string, opts *showPropertyOptions) error {
	out := b.deps.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}
	return showpropertycmd.Run(cmd.Context(), serviceName, showpropertycmd.Options{
		All:     opts.showAll,
		NoColor: b.deps.NoColor || b.deps.Out != nil,
	}, showpropertycmd.Dependencies{
		Getter: b.deps.ShowPropertyGetter,
		Runner: b.deps.ShowPropertyRunner,
		Out:    out,
		Now:    b.deps.Now,
	})
}

func (b *commandBuilder) showPidsCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "show-pids [services...]",
		Aliases:           []string{"show-pid", "pids", "pid"},
		Short:             "查看服务对应的进程的 PID",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runShowPids(cmd, args)
		},
	}
}

func (b *commandBuilder) runShowPids(cmd *cobra.Command, args []string) error {
	out := b.deps.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}
	return showpidscmd.Run(cmd.Context(), args, showpidscmd.Dependencies{
		Finder: b.deps.ShowPidsFinder,
		Out:    out,
	})
}

func (b *commandBuilder) showPortsCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "show-ports [services...]",
		Aliases:           []string{"show-port", "ports", "port"},
		Short:             "查看服务对应的进程的监听端口",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runShowPorts(cmd, args)
		},
	}
}

func (b *commandBuilder) runShowPorts(cmd *cobra.Command, args []string) error {
	out := b.deps.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}
	pidFinder := b.deps.ShowPortsPidFinder
	if pidFinder == nil {
		pidFinder = b.deps.ShowPidsFinder
	}
	return showportscmd.Run(cmd.Context(), args, showportscmd.Dependencies{
		PidFinder:  pidFinder,
		ConnFinder: b.deps.ShowPortsConnFinder,
		Out:        out,
	})
}

func (b *commandBuilder) daemonReloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "daemon-reload",
		Short:             "重载系统 service 树",
		Args:              cobra.NoArgs,
		ValidArgsFunction: completeEmpty,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runDaemonReload(cmd)
		},
	}
}

func (b *commandBuilder) runDaemonReload(cmd *cobra.Command) error {
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
	return daemonreloadcmd.Run(cmd.Context(), daemonreloadcmd.Dependencies{
		Runner: b.deps.DaemonReloadRunner,
		Out:    out,
		ErrOut: errOut,
		In:     in,
	})
}

func (b *commandBuilder) groupListCommand() *cobra.Command {
	opts := &groupListOptions{}
	cmd := &cobra.Command{
		Use:                   "group-list [services...]",
		Aliases:               []string{"list-group"},
		Short:                 "列出组信息",
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.runGroupList(cmd, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.oneLine, "one", "1", false, "在一行中展示")
	cmd.Flags().BoolVarP(&opts.includeServices, "services", "s", false, "同时展示包含的服务名称")
	return cmd
}

func (b *commandBuilder) runGroupList(cmd *cobra.Command, opts *groupListOptions) error {
	out := b.deps.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}
	return grouplistcmd.Run(cmd.Context(), grouplistcmd.Options{
		OneLine:         opts.oneLine,
		IncludeServices: opts.includeServices,
	}, grouplistcmd.Dependencies{
		Store: b.deps.GroupListStore,
		FS:    b.deps.FS,
		Out:   out,
	})
}

func (b *commandBuilder) groupStartCommand() *cobra.Command {
	return b.groupActionCommand("group-start group [groups...]", []string{"start-group"}, "启动一组服务", "start")
}

func (b *commandBuilder) groupStopCommand() *cobra.Command {
	return b.groupActionCommand("group-stop group [groups...]", []string{"stop-group"}, "停止一组服务", "stop")
}

func (b *commandBuilder) groupRestartCommand() *cobra.Command {
	return b.groupActionCommand("group-restart group [groups...]", []string{"restart-group"}, "重启一组服务", "restart")
}

func (b *commandBuilder) installServiceCommand() *cobra.Command {
	return b.installServiceCommandReal()
}

func (b *commandBuilder) installCommandCommand() *cobra.Command {
	return b.installCommandCommandReal()
}

func (b *commandBuilder) installDockerComposeCommand() *cobra.Command {
	return b.installDockerComposeCommandReal()
}

func (b *commandBuilder) linkServiceCommand() *cobra.Command {
	return b.linkServiceCommandReal()
}

func (b *commandBuilder) editCommand() *cobra.Command {
	return b.editCommandReal()
}

func (b *commandBuilder) reGenerateCommand() *cobra.Command {
	return b.reGenerateCommandReal()
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
