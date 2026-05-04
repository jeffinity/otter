package installservice

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
)

type Options struct {
	Name     string
	NoEnable bool
	NoStart  bool
}

type Dependencies struct {
	Installer Installer
	FS        otterfs.Provider
	Runner    startcmd.Runner
	Out       io.Writer
	ErrOut    io.Writer
	In        io.Reader
}

type Installer interface {
	Install(ctx context.Context, name string, data []byte) error
}

func Run(ctx context.Context, file string, opts Options, deps Dependencies) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("cannot read file %s: %w", file, err)
	}
	name := serviceName(opts.Name, file)
	if err := Install(ctx, name, data, opts, deps); err != nil {
		return err
	}
	return nil
}

func Install(ctx context.Context, name string, data []byte, opts Options, deps Dependencies) error {
	name = serviceName(name, "")
	if name == "" {
		return fmt.Errorf("service name is required")
	}
	if err := installer(deps).Install(ctx, name, data); err != nil {
		return fmt.Errorf("cannot install service: %w", err)
	}

	in, out, errOut := streams(deps)
	if _, err := fmt.Fprintf(out, "Service %s install success\n", name); err != nil {
		return err
	}
	_ = run(ctx, []string{"daemon-reload"}, deps, in, out, errOut)
	if !opts.NoEnable {
		_ = run(ctx, []string{"enable", name}, deps, in, out, errOut)
	}
	if !opts.NoStart {
		_ = run(ctx, []string{"start", name}, deps, in, out, errOut)
	}
	return nil
}

func serviceName(name string, file string) string {
	if name == "" && file != "" {
		name = filepath.Base(file)
	}
	return strings.TrimSuffix(name, ".service")
}

func run(
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

func installer(deps Dependencies) Installer {
	if deps.Installer != nil {
		return deps.Installer
	}
	fs := deps.FS
	if fs.Config().ClassicServicePath == "" {
		fs = otterfs.Default()
	}
	return FSInstaller{FS: fs}
}

func runner(deps Dependencies) startcmd.Runner {
	if deps.Runner != nil {
		return deps.Runner
	}
	return execRunner{}
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

type execRunner struct{}

func (execRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
