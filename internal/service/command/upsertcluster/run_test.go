package upsertcluster

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestRunExecsUpsertCluster(t *testing.T) {
	runner := &fakeRunner{}
	mkdir := &fakeMkdirAll{}

	err := Run(context.Background(), []string{"--target", "api"}, Dependencies{
		Runner:   runner,
		Self:     func() string { return "/proc/self/exe" },
		Environ:  func() []string { return []string{"PATH=/bin", "AS=old", "HOME=/root"} },
		MkdirAll: mkdir.mkdirAll,
	})
	if err != nil {
		t.Fatalf("run upsert-cluster: %v", err)
	}
	if got, want := mkdir.path, LogBaseDir; got != want {
		t.Fatalf("mkdir path = %q, want %q", got, want)
	}
	if got, want := mkdir.perm, os.FileMode(0o755)|os.ModeSticky; got != want {
		t.Fatalf("mkdir perm = %v, want %v", got, want)
	}
	if got, want := runner.file, "/proc/self/exe"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if got, want := runner.args, []string{"otter-upsert-cluster", "--target", "api"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
	if got, want := runner.env, []string{"PATH=/bin", "HOME=/root", "AS=otter-upsert-cluster"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("env = %#v, want %#v", got, want)
	}
}

func TestRunIgnoresMkdirError(t *testing.T) {
	runner := &fakeRunner{}

	err := Run(context.Background(), nil, Dependencies{
		Runner:   runner,
		Self:     func() string { return "/proc/self/exe" },
		MkdirAll: func(path string, perm os.FileMode) error { return errors.New("denied") },
	})
	if err != nil {
		t.Fatalf("expected mkdir error to be ignored, got %v", err)
	}
	if got, want := runner.args, []string{"otter-upsert-cluster"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestRunReturnsRunnerError(t *testing.T) {
	err := Run(context.Background(), nil, Dependencies{
		Runner: &fakeRunner{err: errors.New("boom")},
		Self:   func() string { return "/proc/self/exe" },
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

type fakeMkdirAll struct {
	path string
	perm os.FileMode
}

func (f *fakeMkdirAll) mkdirAll(path string, perm os.FileMode) error {
	f.path = path
	f.perm = perm
	return nil
}
