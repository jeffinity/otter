package detail

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

func TestRunCallsSystemctlStatus(t *testing.T) {
	runner := &fakeRunner{}
	out, err := runDetail(t, []string{"api"}, Options{}, sampleServices(), runner)
	if err != nil {
		t.Fatalf("run detail: %v", err)
	}
	if out != "systemctl status api\n" {
		t.Fatalf("output = %q", out)
	}
	if got, want := runner.calls, [][]string{{"status", "api"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func TestRunSelectsServices(t *testing.T) {
	runner := &fakeRunner{}
	_, err := runDetail(t, []string{"api.service", "*er"}, Options{}, sampleServices(), runner)
	if err != nil {
		t.Fatalf("run detail with patterns: %v", err)
	}
	if got, want := runner.calls, [][]string{{"status", "api", "timer", "worker"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func TestRunFiltersServices(t *testing.T) {
	runner := &fakeRunner{}
	_, err := runDetail(t, []string{"all"}, Options{IncludeDisabled: true}, sampleServices(), runner)
	if err != nil {
		t.Fatalf("run detail with disabled: %v", err)
	}
	if got, want := runner.calls, [][]string{{"status", "api", "pkg", "timer", "worker"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}

	runner = &fakeRunner{}
	_, err = runDetail(t, []string{"all"}, Options{ExcludeEnabled: true}, sampleServices(), runner)
	if err != nil {
		t.Fatalf("run detail without enabled: %v", err)
	}
	if got, want := runner.calls, [][]string{{"status", "timer", "worker"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func TestRunNoPager(t *testing.T) {
	runner := &fakeRunner{}
	_, err := runDetail(t, []string{"api"}, Options{NoPager: true}, sampleServices(), runner)
	if err != nil {
		t.Fatalf("run detail without pager: %v", err)
	}
	if got, want := runner.calls, [][]string{{"status", "api", "--no-pager"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func TestRunReturnsSelectionError(t *testing.T) {
	runner := &fakeRunner{}
	_, err := runDetail(t, []string{"missing*"}, Options{}, sampleServices(), runner)
	if err == nil || err.Error() != "service pattern missing* cannot be match by any service" {
		t.Fatalf("expected unmatched pattern error, got %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner should not be called, got %#v", runner.calls)
	}
}

func TestRunReturnsRunnerError(t *testing.T) {
	wantErr := errors.New("systemctl failed")
	runner := &fakeRunner{err: wantErr}
	_, err := runDetail(t, []string{"api"}, Options{}, sampleServices(), runner)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected runner error, got %v", err)
	}
}

func runDetail(
	t *testing.T,
	args []string,
	opts Options,
	services []statuscmd.Service,
	runner Runner,
) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(context.Background(), args, opts, Dependencies{
		Store:  fakeStore{services: services},
		Runner: runner,
		Out:    &out,
	})
	return out.String(), err
}

func sampleServices() []statuscmd.Service {
	return []statuscmd.Service{
		{Name: "worker", UnitName: "worker.service", Source: statuscmd.SourceSystemd, Enabled: false},
		{Name: "api", UnitName: "api.service", Source: statuscmd.SourceSystemd, Enabled: true, Running: true},
		{Name: "pkg", UnitName: "pkg.service", Source: statuscmd.SourcePackage, Enabled: true, Running: true},
		{Name: "timer", UnitName: "timer.service", Source: statuscmd.SourceSystemd, Enabled: false, Running: true},
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
	if out != nil {
		_, _ = io.Copy(out, strings.NewReader(""))
	}
	return f.err
}
