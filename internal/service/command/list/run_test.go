package list

import (
	"bytes"
	"context"
	"strings"
	"testing"

	statuscmd "github.com/jeffinity/otter/internal/service/command/status"
)

func TestRunFiltersServices(t *testing.T) {
	out, err := runList(t, nil, Options{}, sampleServices())
	if err != nil {
		t.Fatalf("run list: %v", err)
	}
	assertLines(t, out, "api", "pkg", "timer")

	out, err = runList(t, nil, Options{IncludeDisabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run list with disabled: %v", err)
	}
	assertLines(t, out, "api", "pkg", "timer", "worker")

	out, err = runList(t, nil, Options{ExcludeEnabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run list without enabled: %v", err)
	}
	assertLines(t, out, "timer", "worker")

	out, err = runList(t, []string{"worker"}, Options{}, sampleServices())
	if err != nil {
		t.Fatalf("run explicit disabled list: %v", err)
	}
	assertLines(t, out, "worker")
}

func TestRunMatchesPatterns(t *testing.T) {
	out, err := runList(t, []string{"api.service"}, Options{}, sampleServices())
	if err != nil {
		t.Fatalf("run list by suffix: %v", err)
	}
	assertLines(t, out, "api")

	out, err = runList(t, []string{"*er"}, Options{}, sampleServices())
	if err != nil {
		t.Fatalf("run list by glob: %v", err)
	}
	assertLines(t, out, "timer", "worker")

	_, err = runList(t, []string{"all", "api"}, Options{}, sampleServices())
	if err == nil || err.Error() != "if use `all` or `*`, it must be the only one service pattern" {
		t.Fatalf("expected all conflict error, got %v", err)
	}

	_, err = runList(t, []string{"missing*"}, Options{}, sampleServices())
	if err == nil || err.Error() != "service pattern missing* cannot be match by any service" {
		t.Fatalf("expected unmatched pattern error, got %v", err)
	}
}

func TestRunFiltersSources(t *testing.T) {
	out, err := runList(t, nil, Options{OnlyPackage: true, IncludeDisabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run package list: %v", err)
	}
	assertLines(t, out, "pkg")

	out, err = runList(t, nil, Options{OnlyClassic: true, IncludeDisabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run classic list: %v", err)
	}
	assertLines(t, out, "api", "timer", "worker")

	_, err = runList(t, nil, Options{OnlyPackage: true, OnlyClassic: true}, sampleServices())
	if err == nil || err.Error() != "--only-package and --only-classic cannot be apply in the meantime" {
		t.Fatalf("expected source conflict error, got %v", err)
	}
}

func TestRunOneLine(t *testing.T) {
	out, err := runList(t, nil, Options{IncludeDisabled: true, OneLine: true}, sampleServices())
	if err != nil {
		t.Fatalf("run one-line list: %v", err)
	}
	if out != "api pkg timer worker\n" {
		t.Fatalf("one-line output = %q", out)
	}
}

func runList(t *testing.T, args []string, opts Options, services []statuscmd.Service) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(context.Background(), args, opts, Dependencies{
		Store: fakeStore{services: services},
		Out:   &out,
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

func assertLines(t *testing.T, text string, values ...string) {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	if len(lines) != len(values) {
		t.Fatalf("lines = %#v, want %#v", lines, values)
	}
	for index, value := range values {
		if lines[index] != value {
			t.Fatalf("lines = %#v, want %#v", lines, values)
		}
	}
}

type fakeStore struct {
	services []statuscmd.Service
}

func (f fakeStore) List(ctx context.Context) ([]statuscmd.Service, error) {
	return f.services, nil
}
