package groupstart

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	grouplistcmd "github.com/jeffinity/otter/internal/service/command/grouplist"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

func TestRunStartsGroupServices(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeRunner{}

	err := Run(context.Background(), []string{"web", "worker"}, Options{}, Dependencies{
		GroupStore: fakeGroupStore{groups: map[string][]string{
			"web":    {"api", "job"},
			"worker": {"job", "worker"},
		}},
		ActionStore: fakeStore{services: []statuscmd.Service{
			{Name: "worker", Enabled: true},
			{Name: "api", Enabled: true},
			{Name: "job", Enabled: true},
		}},
		Runner: runner,
		Out:    &out,
	})
	if err != nil {
		t.Fatalf("run group-start: %v", err)
	}
	if got, want := strings.Join(runner.calls[0], " "), "start api job worker"; got != want {
		t.Fatalf("runner args = %q, want %q", got, want)
	}
	if got, want := out.String(), "systemctl start api job worker\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunPassesStopAfter(t *testing.T) {
	stopper := &fakeStopper{}

	err := Run(context.Background(), []string{"web"}, Options{StopAfter: 2 * time.Minute}, Dependencies{
		GroupStore:  fakeGroupStore{groups: map[string][]string{"web": {"api"}}},
		ActionStore: fakeStore{services: []statuscmd.Service{{Name: "api", Enabled: true}}},
		Runner:      &fakeRunner{},
		AutoStopper: stopper,
		Out:         io.Discard,
	})
	if err != nil {
		t.Fatalf("run group-start --stop-after: %v", err)
	}
	if got, want := strings.Join(stopper.services, " "), "api"; got != want {
		t.Fatalf("auto-stop services = %q, want %q", got, want)
	}
	if got, want := stopper.duration, 2*time.Minute; got != want {
		t.Fatalf("auto-stop duration = %s, want %s", got, want)
	}
}

func TestRunRequiresGroup(t *testing.T) {
	err := Run(context.Background(), nil, Options{}, Dependencies{})
	if err == nil || err.Error() != "at least one group should be provided" {
		t.Fatalf("expected group error, got %v", err)
	}
}

func TestRunReportsMissingGroup(t *testing.T) {
	err := Run(context.Background(), []string{"missing"}, Options{}, Dependencies{
		GroupStore: fakeGroupStore{groups: map[string][]string{"web": {"api"}}},
	})
	if err == nil || err.Error() != "cannot get services from group: group missing is not exist" {
		t.Fatalf("expected missing group error, got %v", err)
	}
}

func TestRunWrapsGroupStoreError(t *testing.T) {
	errBoom := errors.New("boom")
	err := Run(context.Background(), []string{"web"}, Options{}, Dependencies{
		GroupStore: fakeGroupStore{err: errBoom},
	})
	if !errors.Is(err, errBoom) || err.Error() != "cannot get services from group: boom" {
		t.Fatalf("expected wrapped store error, got %v", err)
	}
}

type fakeGroupStore struct {
	groups map[string][]string
	err    error
}

func (f fakeGroupStore) List(ctx context.Context) (map[string][]string, error) {
	return f.groups, f.err
}

var _ grouplistcmd.Store = fakeGroupStore{}

type fakeStore struct {
	services []statuscmd.Service
	err      error
}

func (f fakeStore) List(ctx context.Context) ([]statuscmd.Service, error) {
	return f.services, f.err
}

var _ statuscmd.Store = fakeStore{}

type fakeRunner struct {
	calls [][]string
}

func (f *fakeRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	return nil
}

var _ startcmd.Runner = (*fakeRunner)(nil)

type fakeStopper struct {
	services []string
	duration time.Duration
}

func (f *fakeStopper) StopAfter(ctx context.Context, services []string, duration time.Duration) error {
	f.services = append([]string(nil), services...)
	f.duration = duration
	return nil
}

var _ startcmd.AutoStopper = (*fakeStopper)(nil)
