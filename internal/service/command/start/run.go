package start

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

type Options struct {
	ExcludeEnabled  bool
	IncludeDisabled bool
	Reload          bool
	StopAfter       time.Duration
	Trace           bool
	NoColor         bool
}

type Dependencies struct {
	Store       statuscmd.Store
	Runner      Runner
	AutoStopper AutoStopper
	TraceRunner TraceRunner
	Executable  func() (string, error)
	Environ     func() []string
	Out         io.Writer
	ErrOut      io.Writer
	In          io.Reader
}

type Runner interface {
	Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error
}

type AutoStopper interface {
	StopAfter(ctx context.Context, services []string, duration time.Duration) error
}

type TraceRunner interface {
	Exec(ctx context.Context, file string, args []string, env []string) error
}

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	return RunAction(ctx, "start", args, opts, deps)
}

func RunAction(ctx context.Context, action string, args []string, opts Options, deps Dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one service should be provided")
	}

	services, err := selectServices(ctx, args, opts, deps)
	if err != nil {
		return err
	}
	names := serviceNames(services)
	in, out, errOut := streams(deps)

	if opts.Reload {
		if err := runSystemctl(ctx, []string{"daemon-reload"}, deps, in, out, errOut); err != nil {
			return err
		}
	}
	if err := runSystemctl(ctx, append([]string{action}, names...), deps, in, out, errOut); err != nil {
		return err
	}
	if opts.StopAfter != 0 {
		if err := autoStopper(deps).StopAfter(ctx, names, opts.StopAfter); err != nil {
			return fmt.Errorf("Cannot setup auto-stop: %w", err)
		}
	}
	if opts.Trace {
		return trace(ctx, names, deps, out, opts)
	}
	return nil
}

func selectServices(
	ctx context.Context,
	args []string,
	opts Options,
	deps Dependencies,
) ([]statuscmd.Service, error) {
	if deps.Store == nil {
		return nil, errors.New("service store is required")
	}
	services, err := deps.Store.List(ctx)
	if err != nil {
		return nil, err
	}
	patterns, err := normalizePatterns(args)
	if err != nil {
		return nil, err
	}
	if err := checkPatterns(services, patterns); err != nil {
		return nil, err
	}
	services = filterServices(services, patterns, opts)
	if len(services) == 0 {
		return nil, fmt.Errorf("no services after filter")
	}
	sort.SliceStable(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	return services, nil
}

func normalizePatterns(args []string) ([]string, error) {
	patterns := make([]string, 0, len(args))
	includeAll := false
	for _, arg := range args {
		pattern := strings.TrimSuffix(strings.TrimSpace(arg), ".service")
		if pattern == "" {
			continue
		}
		if pattern == "all" || pattern == "*" {
			includeAll = true
		}
		patterns = append(patterns, pattern)
	}
	if includeAll && len(patterns) != 1 {
		return nil, fmt.Errorf("if use `all` or `*`, it must be the only one service pattern")
	}
	if includeAll {
		return nil, nil
	}
	return patterns, nil
}

func checkPatterns(services []statuscmd.Service, patterns []string) error {
	for _, pattern := range patterns {
		matched := false
		for _, service := range services {
			if matchPattern(pattern, service.Name) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("service pattern %s cannot be match by any service", pattern)
		}
	}
	return nil
}

func filterServices(
	services []statuscmd.Service,
	patterns []string,
	opts Options,
) []statuscmd.Service {
	filtered := make([]statuscmd.Service, 0, len(services))
	explicit := len(patterns) != 0
	for _, service := range services {
		if explicit {
			if matchAny(patterns, service.Name) {
				filtered = append(filtered, service)
			}
			continue
		}

		switch {
		case !opts.ExcludeEnabled && !opts.IncludeDisabled:
			if service.Enabled {
				filtered = append(filtered, service)
			}
		case !opts.ExcludeEnabled && opts.IncludeDisabled:
			filtered = append(filtered, service)
		default:
			if !service.Enabled {
				filtered = append(filtered, service)
			}
		}
	}
	return filtered
}

func matchAny(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if matchPattern(pattern, name) {
			return true
		}
	}
	return false
}

func matchPattern(pattern, name string) bool {
	ok, err := filepath.Match(pattern, name)
	if err != nil {
		return pattern == name
	}
	return ok
}

func serviceNames(services []statuscmd.Service) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Name)
	}
	return names
}

func runSystemctl(
	ctx context.Context,
	args []string,
	deps Dependencies,
	in io.Reader,
	out io.Writer,
	errOut io.Writer,
) error {
	if _, err := fmt.Fprintln(out, "systemctl "+strings.Join(args, " ")); err != nil {
		return err
	}
	return runner(deps).Run(ctx, args, in, out, errOut)
}

func trace(ctx context.Context, names []string, deps Dependencies, out io.Writer, opts Options) error {
	if len(names) != 1 {
		msg := "Trace can only be used if start exactly one service, you can run the following command manually if need:"
		if _, err := fmt.Fprintln(out, colorize(msg, ansiYellow, opts.NoColor)); err != nil {
			return err
		}
		for _, name := range names {
			if _, err := fmt.Fprintln(out, "> otter service log -f "+name); err != nil {
				return err
			}
		}
		return nil
	}

	executable := deps.Executable
	if executable == nil {
		executable = os.Executable
	}
	self, err := executable()
	if err != nil {
		return fmt.Errorf("cannot found self executable: %w", err)
	}
	return traceRunner(deps).Exec(ctx, self, []string{"otter", "service", "log", "-f", names[0]}, cleanEnv(deps))
}

type ansiColor string

const (
	ansiYellow ansiColor = "\x1b[33m"
	ansiReset  ansiColor = "\x1b[0m"
)

func colorize(text string, color ansiColor, noColor bool) string {
	if noColor {
		return text
	}
	return string(color) + text + string(ansiReset)
}

func streams(deps Dependencies) (io.Reader, io.Writer, io.Writer) {
	in := deps.In
	if in == nil {
		in = os.Stdin
	}
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := deps.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}
	return in, out, errOut
}

func runner(deps Dependencies) Runner {
	if deps.Runner != nil {
		return deps.Runner
	}
	return execRunner{}
}

func traceRunner(deps Dependencies) TraceRunner {
	if deps.TraceRunner != nil {
		return deps.TraceRunner
	}
	return execTraceRunner{}
}

func autoStopper(deps Dependencies) AutoStopper {
	if deps.AutoStopper != nil {
		return deps.AutoStopper
	}
	return systemdRunAutoStopper{}
}

func cleanEnv(deps Dependencies) []string {
	environ := deps.Environ
	if environ == nil {
		environ = os.Environ
	}

	env := make([]string, 0)
	for _, item := range environ() {
		if strings.HasPrefix(item, "AS=") {
			continue
		}
		env = append(env, item)
	}
	return env
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

type execTraceRunner struct{}

func (execTraceRunner) Exec(ctx context.Context, file string, args []string, env []string) error {
	_ = ctx
	return syscall.Exec(file, args, env)
}

type systemdRunAutoStopper struct{}

func (systemdRunAutoStopper) StopAfter(ctx context.Context, services []string, duration time.Duration) error {
	seconds := int64(duration / time.Second)
	args := []string{
		"--on-active=" + strconv.FormatInt(seconds, 10) + "s",
		"--unit=" + autoStopUnit(services),
		"systemctl",
		"stop",
	}
	args = append(args, services...)
	return exec.CommandContext(ctx, "systemd-run", args...).Run()
}

func autoStopUnit(services []string) string {
	name := strings.Join(services, "-")
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "otter-auto-stop"
	}
	return "otter-auto-stop-" + b.String()
}
