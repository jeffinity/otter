package regenerate

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestRunRegeneratesAndRestarts(t *testing.T) {
	gen := &fakeGenerator{}
	runner := &fakeRunner{}
	var out bytes.Buffer
	err := Run(context.Background(), "api.service", Options{Restart: true, NotRestart: true}, Dependencies{
		Finder:    fakeFinder{service: PackageService{Name: "api"}},
		Generator: gen,
		Runner:    runner,
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("run regen: %v", err)
	}
	if !gen.called {
		t.Fatalf("generator not called")
	}
	if got, want := strings.Join(runner.args, " "), "restart api"; got != want {
		t.Fatalf("runner args = %q, want %q", got, want)
	}
	assertContains(t, out.String(), "Re-generate service api success\n", "Will restart service api\n", "Success\n")
}

func TestRunPromptsWhenNoRestartFlag(t *testing.T) {
	runner := &fakeRunner{}
	err := Run(context.Background(), "api", Options{}, Dependencies{
		Finder:    fakeFinder{service: PackageService{Name: "api"}},
		Generator: &fakeGenerator{},
		Prompter:  fakePrompter{confirm: false},
		Runner:    runner,
		Out:       io.Discard,
	})
	if err != nil {
		t.Fatalf("run regen: %v", err)
	}
	if runner.args != nil {
		t.Fatalf("runner should not be called")
	}
}

func assertContains(t *testing.T, text string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(text, value) {
			t.Fatalf("expected %q to contain %q", text, value)
		}
	}
}

type fakeFinder struct {
	service PackageService
}

func (f fakeFinder) FindPackage(ctx context.Context, serviceName string) (PackageService, error) {
	return f.service, nil
}

type fakeGenerator struct {
	called bool
}

func (f *fakeGenerator) Generate(ctx context.Context, service PackageService) error {
	f.called = true
	return nil
}

type fakePrompter struct {
	confirm bool
}

func (f fakePrompter) Confirm(prompt string) (bool, error) {
	return f.confirm, nil
}

type fakeRunner struct {
	args []string
}

func (f *fakeRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.args = append([]string(nil), args...)
	return nil
}
