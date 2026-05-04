package detail

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

type Options struct {
	ExcludeEnabled  bool
	IncludeDisabled bool
	NoPager         bool
}

type Dependencies struct {
	Store  statuscmd.Store
	Runner Runner
	Out    io.Writer
	ErrOut io.Writer
	In     io.Reader
}

type Runner interface {
	Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error
}

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	services, err := statuscmd.Select(ctx, args, statuscmd.Options{
		ExcludeEnabled:  opts.ExcludeEnabled,
		IncludeDisabled: opts.IncludeDisabled,
	}, statuscmd.Dependencies{Store: deps.Store})
	if err != nil {
		return err
	}

	names := serviceNames(services)
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	if _, err := fmt.Fprintln(out, "systemctl status "+strings.Join(names, " ")); err != nil {
		return err
	}

	runner := deps.Runner
	if runner == nil {
		runner = execRunner{}
	}
	runArgs := append([]string{"status"}, names...)
	if opts.NoPager {
		runArgs = append(runArgs, "--no-pager")
	}
	return runner.Run(ctx, runArgs, deps.In, out, deps.ErrOut)
}

func serviceNames(services []statuscmd.Service) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Name)
	}
	return names
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
