package daemonreload

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Dependencies struct {
	Runner Runner
	Out    io.Writer
	ErrOut io.Writer
	In     io.Reader
}

type Runner interface {
	Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error
}

func Run(ctx context.Context, deps Dependencies) error {
	in, out, errOut := streams(deps)
	if _, err := fmt.Fprintln(out, "systemctl daemon-reload "); err != nil {
		return err
	}
	return runner(deps).Run(ctx, []string{"daemon-reload"}, in, out, errOut)
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

type execRunner struct{}

func (execRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
