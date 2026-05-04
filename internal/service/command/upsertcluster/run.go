package upsertcluster

import (
	"context"
	"os"
	"runtime"
	"strings"
	"syscall"
)

const (
	AppName    = "otter-upsert-cluster"
	LogBaseDir = "/data/log/otter"
)

type Dependencies struct {
	Runner   Runner
	Self     func() string
	Environ  func() []string
	MkdirAll func(path string, perm os.FileMode) error
}

type Runner interface {
	Exec(ctx context.Context, file string, args []string, env []string) error
}

func Run(ctx context.Context, args []string, deps Dependencies) error {
	_ = mkdirAll(deps)(LogBaseDir, 0o755|os.ModeSticky)

	execArgs := append([]string{AppName}, args...)
	return runner(deps).Exec(ctx, self(deps), execArgs, env(deps))
}

func self(deps Dependencies) string {
	if deps.Self != nil {
		return deps.Self()
	}
	if runtime.GOOS == "linux" {
		return "/proc/self/exe"
	}
	self, err := os.Executable()
	if err != nil {
		panic("cannot get self executable: " + err.Error())
	}
	return self
}

func env(deps Dependencies) []string {
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
	return append(env, "AS="+AppName)
}

func mkdirAll(deps Dependencies) func(path string, perm os.FileMode) error {
	if deps.MkdirAll != nil {
		return deps.MkdirAll
	}
	return os.MkdirAll
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
