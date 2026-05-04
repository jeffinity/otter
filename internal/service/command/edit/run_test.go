package edit

import (
	"context"
	"io"
	"testing"

	"github.com/jeffinity/otter/internal/service/command/servicefile"
)

func TestRunEditsClassicAndRestarts(t *testing.T) {
	runner := &fakeRunner{}
	system := &fakeSystemRunner{}
	err := Run(context.Background(), "api.service", Dependencies{
		Finder: servicefileFinder{file: servicefile.File{Name: "api", Path: "/tmp/api.service", Source: servicefile.SourceClassic}},
		Prompter: fakePrompter{
			confirm: true,
		},
		Runner:       runner,
		SystemRunner: system,
		LookPath:     func(file string) (string, error) { return "/usr/bin/" + file, nil },
		Getenv:       func(string) string { return "vim" },
		Out:          io.Discard,
	})
	if err != nil {
		t.Fatalf("run edit: %v", err)
	}
	if runner.command != "/usr/bin/vim" || runner.args[0] != "/tmp/api.service" {
		t.Fatalf("editor call = %s %v", runner.command, runner.args)
	}
	if len(system.calls) != 2 || system.calls[0][0] != "daemon-reload" || system.calls[1][0] != "restart" {
		t.Fatalf("system calls = %#v", system.calls)
	}
}

type servicefileFinder struct {
	file servicefile.File
}

func (f servicefileFinder) Find(ctx context.Context, serviceName string) (servicefile.File, error) {
	return f.file, nil
}

type fakeRunner struct {
	command string
	args    []string
}

func (f *fakeRunner) Run(ctx context.Context, command string, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.command = command
	f.args = append([]string(nil), args...)
	return nil
}

type fakeSystemRunner struct {
	calls [][]string
}

func (f *fakeSystemRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	return nil
}

type fakePrompter struct {
	confirm bool
}

func (f fakePrompter) Choose(options []string) (string, error) {
	return "y", nil
}

func (f fakePrompter) Confirm(prompt string) (bool, error) {
	return f.confirm, nil
}

func (f fakePrompter) Input(prompt string) (string, error) {
	return "message", nil
}
