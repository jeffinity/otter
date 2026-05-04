package selfcheck

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestRunExecsSelfCheck(t *testing.T) {
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"--deep", "api"}, Dependencies{
		Runner:     runner,
		Executable: func() (string, error) { return "/usr/bin/otter", nil },
		Environ:    func() []string { return []string{"PATH=/bin", "AS=old", "HOME=/root"} },
	})
	if err != nil {
		t.Fatalf("run self-check: %v", err)
	}
	if got, want := runner.file, "/usr/bin/otter"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if got, want := runner.args, []string{"otter-self-check", "--deep", "api"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
	if got, want := runner.env, []string{"PATH=/bin", "HOME=/root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("env = %#v, want %#v", got, want)
	}
}

func TestRunReturnsExecutableError(t *testing.T) {
	err := Run(context.Background(), nil, Dependencies{
		Runner:     &fakeRunner{},
		Executable: func() (string, error) { return "", errors.New("missing") },
	})
	if err == nil || err.Error() != "cannot found self executable: missing" {
		t.Fatalf("expected executable error, got %v", err)
	}
}

func TestRunReturnsRunnerError(t *testing.T) {
	err := Run(context.Background(), nil, Dependencies{
		Runner:     &fakeRunner{err: errors.New("boom")},
		Executable: func() (string, error) { return "/usr/bin/otter", nil },
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected runner error, got %v", err)
	}
}

type fakeRunner struct {
	file string
	args []string
	env  []string
	err  error
}

func (f *fakeRunner) Exec(ctx context.Context, file string, args []string, env []string) error {
	f.file = file
	f.args = append([]string(nil), args...)
	f.env = append([]string(nil), env...)
	return f.err
}

var _ Runner = (*fakeRunner)(nil)
