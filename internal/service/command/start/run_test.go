package start

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

func TestRunStartsSelectedServices(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"api.service", "worker"}, Options{}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{
			{Name: "worker", Enabled: false},
			{Name: "api", Enabled: true},
		}},
		Runner: runner,
		Out:    &out,
	})
	if err != nil {
		t.Fatalf("run start: %v", err)
	}
	if got, want := runner.calls, [][]string{{"start", "api", "worker"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
	if got, want := out.String(), "systemctl start api worker\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunAllDefaultsToEnabledOnly(t *testing.T) {
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"all"}, Options{}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{
			{Name: "api", Enabled: true},
			{Name: "disabled", Enabled: false, Running: true},
		}},
		Runner: runner,
		Out:    io.Discard,
	})
	if err != nil {
		t.Fatalf("run start all: %v", err)
	}
	if got, want := runner.calls, [][]string{{"start", "api"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
}

func TestRunAllIncludeDisabledAndExcludeEnabled(t *testing.T) {
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"*"}, Options{
		ExcludeEnabled:  true,
		IncludeDisabled: true,
	}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{
			{Name: "api", Enabled: true},
			{Name: "disabled", Enabled: false},
		}},
		Runner: runner,
		Out:    io.Discard,
	})
	if err != nil {
		t.Fatalf("run start disabled: %v", err)
	}
	if got, want := runner.calls, [][]string{{"start", "disabled"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
}

func TestRunRejectsAllWithOtherPatterns(t *testing.T) {
	err := Run(context.Background(), []string{"all", "api"}, Options{}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{{Name: "api", Enabled: true}}},
		Out:   io.Discard,
	})
	if err == nil || err.Error() != "if use `all` or `*`, it must be the only one service pattern" {
		t.Fatalf("expected all exclusivity error, got %v", err)
	}
}

func TestRunRejectsUnmatchedGlob(t *testing.T) {
	err := Run(context.Background(), []string{"api*"}, Options{}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{{Name: "worker", Enabled: true}}},
		Out:   io.Discard,
	})
	if err == nil || err.Error() != "service pattern api* cannot be match by any service" {
		t.Fatalf("expected unmatched glob error, got %v", err)
	}
}

func TestRunReloadsBeforeStart(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"api"}, Options{Reload: true}, Dependencies{
		Store:  fakeStore{services: []statuscmd.Service{{Name: "api", Enabled: true}}},
		Runner: runner,
		Out:    &out,
	})
	if err != nil {
		t.Fatalf("run start --reload: %v", err)
	}
	wantCalls := [][]string{{"daemon-reload"}, {"start", "api"}}
	if got := runner.calls; !reflect.DeepEqual(got, wantCalls) {
		t.Fatalf("runner calls = %#v, want %#v", got, wantCalls)
	}
	if got, want := out.String(), "systemctl daemon-reload\nsystemctl start api\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunStopAfterUsesAutoStopper(t *testing.T) {
	autoStopper := &fakeAutoStopper{}

	err := Run(context.Background(), []string{"api"}, Options{StopAfter: time.Minute}, Dependencies{
		Store:       fakeStore{services: []statuscmd.Service{{Name: "api", Enabled: true}}},
		Runner:      &fakeRunner{},
		AutoStopper: autoStopper,
		Out:         io.Discard,
	})
	if err != nil {
		t.Fatalf("run start --stop-after: %v", err)
	}
	if got, want := autoStopper.services, []string{"api"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("auto-stop services = %#v, want %#v", got, want)
	}
	if got, want := autoStopper.duration, time.Minute; got != want {
		t.Fatalf("auto-stop duration = %v, want %v", got, want)
	}
}

func TestRunTraceExecsLogForSingleService(t *testing.T) {
	traceRunner := &fakeTraceRunner{}

	err := Run(context.Background(), []string{"api"}, Options{Trace: true}, Dependencies{
		Store:       fakeStore{services: []statuscmd.Service{{Name: "api", Enabled: true}}},
		Runner:      &fakeRunner{},
		TraceRunner: traceRunner,
		Executable:  func() (string, error) { return "/usr/bin/otter", nil },
		Environ:     func() []string { return []string{"PATH=/bin", "AS=old"} },
		Out:         io.Discard,
	})
	if err != nil {
		t.Fatalf("run start --trace: %v", err)
	}
	if got, want := traceRunner.file, "/usr/bin/otter"; got != want {
		t.Fatalf("trace file = %q, want %q", got, want)
	}
	if got, want := strings.Join(traceRunner.args, " "), "otter service log -f api"; got != want {
		t.Fatalf("trace args = %q, want %q", got, want)
	}
	if got, want := strings.Join(traceRunner.env, " "), "PATH=/bin"; got != want {
		t.Fatalf("trace env = %q, want %q", got, want)
	}
}

func TestRunTracePrintsCommandsForMultipleServices(t *testing.T) {
	var out bytes.Buffer

	err := Run(context.Background(), []string{"all"}, Options{Trace: true}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{
			{Name: "api", Enabled: true},
			{Name: "worker", Enabled: true},
		}},
		Runner: &fakeRunner{},
		Out:    &out,
	})
	if err != nil {
		t.Fatalf("run start --trace all: %v", err)
	}
	if !strings.Contains(out.String(), "\x1b[33mTrace can only be used if start exactly one service, you can run the following command manually if need:\x1b[0m\n") ||
		!strings.Contains(out.String(), "> otter service log -f api\n") ||
		!strings.Contains(out.String(), "> otter service log -f worker\n") {
		t.Fatalf("trace output = %q", out.String())
	}
}

func TestRunTraceCanDisableColor(t *testing.T) {
	var out bytes.Buffer

	err := Run(context.Background(), []string{"all"}, Options{Trace: true, NoColor: true}, Dependencies{
		Store: fakeStore{services: []statuscmd.Service{
			{Name: "api", Enabled: true},
			{Name: "worker", Enabled: true},
		}},
		Runner: &fakeRunner{},
		Out:    &out,
	})
	if err != nil {
		t.Fatalf("run start --trace all without color: %v", err)
	}
	if strings.Contains(out.String(), "\x1b[33m") ||
		!strings.Contains(out.String(), "Trace can only be used if start exactly one service") {
		t.Fatalf("trace output = %q", out.String())
	}
}

func TestRunReturnsRunnerError(t *testing.T) {
	err := Run(context.Background(), []string{"api"}, Options{}, Dependencies{
		Store:  fakeStore{services: []statuscmd.Service{{Name: "api", Enabled: true}}},
		Runner: &fakeRunner{err: errors.New("boom")},
		Out:    io.Discard,
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected runner error, got %v", err)
	}
}

type fakeStore struct {
	services []statuscmd.Service
}

func (f fakeStore) List(ctx context.Context) ([]statuscmd.Service, error) {
	return f.services, nil
}

type fakeRunner struct {
	calls [][]string
	err   error
}

func (f *fakeRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	return f.err
}

var _ Runner = (*fakeRunner)(nil)

type fakeAutoStopper struct {
	services []string
	duration time.Duration
}

func (f *fakeAutoStopper) StopAfter(ctx context.Context, services []string, duration time.Duration) error {
	f.services = append([]string(nil), services...)
	f.duration = duration
	return nil
}

var _ AutoStopper = (*fakeAutoStopper)(nil)

type fakeTraceRunner struct {
	file string
	args []string
	env  []string
}

func (f *fakeTraceRunner) Exec(ctx context.Context, file string, args []string, env []string) error {
	f.file = file
	f.args = append([]string(nil), args...)
	f.env = append([]string(nil), env...)
	return nil
}

var _ TraceRunner = (*fakeTraceRunner)(nil)
