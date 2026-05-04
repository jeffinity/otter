package selfcheck

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
)

const AppName = "otter-self-check"

type Dependencies struct {
	Runner     Runner
	Executable func() (string, error)
	Environ    func() []string
}

type Runner interface {
	Exec(ctx context.Context, file string, args []string, env []string) error
}

func Run(ctx context.Context, args []string, deps Dependencies) error {
	executable := deps.Executable
	if executable == nil {
		executable = os.Executable
	}
	self, err := executable()
	if err != nil {
		return fmt.Errorf("cannot found self executable: %w", err)
	}

	execArgs := append([]string{AppName}, args...)
	return runner(deps).Exec(ctx, self, execArgs, cleanEnv(deps))
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

func runner(deps Dependencies) Runner {
	if deps.Runner != nil {
		return deps.Runner
	}
	return execRunner{}
}

type execRunner struct{}

func (execRunner) Exec(ctx context.Context, file string, args []string, env []string) error {
	_ = ctx
	return syscall.Exec(file, args, env)
}
