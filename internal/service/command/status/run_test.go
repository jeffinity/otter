package status

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jeffinity/otter/internal/otterfs"
)

func TestRunNormalizesServiceSuffixAndPatterns(t *testing.T) {
	out, err := runStatus(t, []string{"api.service"}, Options{}, sampleServices())
	if err != nil {
		t.Fatalf("run status: %v", err)
	}
	if !strings.Contains(out, "api") || strings.Contains(out, "worker") {
		t.Fatalf("expected only api, got %q", out)
	}

	_, err = runStatus(t, []string{"all", "api"}, Options{}, sampleServices())
	if err == nil || err.Error() != "if use `all` or `*`, it must be the only one service pattern" {
		t.Fatalf("expected all conflict error, got %v", err)
	}

	_, err = runStatus(t, []string{"missing*"}, Options{}, sampleServices())
	if err == nil || err.Error() != "service pattern missing* cannot be match by any service" {
		t.Fatalf("expected unmatched pattern error, got %v", err)
	}
}

func TestRunFiltersServices(t *testing.T) {
	out, err := runStatus(t, nil, Options{}, sampleServices())
	if err != nil {
		t.Fatalf("run status: %v", err)
	}
	assertContains(t, out, "api", "timer")
	assertNotContains(t, out, "worker")

	out, err = runStatus(t, nil, Options{IncludeDisabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run status with disabled: %v", err)
	}
	assertContains(t, out, "api", "timer", "worker")

	out, err = runStatus(t, nil, Options{ExcludeEnabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run status without enabled: %v", err)
	}
	assertContains(t, out, "worker", "timer")
	assertNotContains(t, out, "api")

	out, err = runStatus(t, []string{"worker"}, Options{}, sampleServices())
	if err != nil {
		t.Fatalf("run explicit disabled status: %v", err)
	}
	assertContains(t, out, "worker")
}

func TestRunFiltersSources(t *testing.T) {
	out, err := runStatus(t, nil, Options{OnlyPackage: true, IncludeDisabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run package status: %v", err)
	}
	assertContains(t, out, "pkg")
	assertNotContains(t, out, "api", "worker", "timer")

	out, err = runStatus(t, nil, Options{OnlyClassic: true, IncludeDisabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run classic status: %v", err)
	}
	assertContains(t, out, "api", "worker", "timer")
	assertNotContains(t, out, "pkg")

	_, err = runStatus(t, nil, Options{OnlyPackage: true, OnlyClassic: true}, sampleServices())
	if err == nil || err.Error() != "--only-package and --only-classic cannot be apply in the meantime" {
		t.Fatalf("expected source conflict error, got %v", err)
	}

	_, err = runStatus(t, nil, Options{SortAsc: true, SortDesc: true}, sampleServices())
	if err == nil || err.Error() != "--asc and --desc cannot be apply in the meantime" {
		t.Fatalf("expected sort conflict error, got %v", err)
	}
}

func TestRunSortsByNameAndTime(t *testing.T) {
	out, err := runStatus(t, nil, Options{IncludeDisabled: true}, sampleServices())
	if err != nil {
		t.Fatalf("run status: %v", err)
	}
	assertOrder(t, out, "api", "pkg", "timer", "worker")

	out, err = runStatus(t, nil, Options{IncludeDisabled: true, SortAsc: true}, sampleServices())
	if err != nil {
		t.Fatalf("run status asc: %v", err)
	}
	assertOrder(t, out, "api", "pkg", "timer", "worker")

	out, err = runStatus(t, nil, Options{IncludeDisabled: true, SortDesc: true}, sampleServices())
	if err != nil {
		t.Fatalf("run status desc: %v", err)
	}
	assertOrder(t, out, "worker", "timer", "pkg", "api")
}

func TestRunTimeInfo(t *testing.T) {
	out, err := runStatus(t, []string{"api"}, Options{IncludeTimeInfo: true}, sampleServices())
	if err != nil {
		t.Fatalf("run status with time info: %v", err)
	}
	assertContains(t, out, "2026-04-27 11:59:50", "10s ago")

	out, err = runStatus(t, []string{"timer"}, Options{IncludeTimeInfo: true}, sampleServices())
	if err != nil {
		t.Fatalf("run status with missing mono: %v", err)
	}
	assertContains(t, out, "[无法获取 mono 时间]")

	out, err = runStatus(t, nil, Options{IncludeDisabled: true, Since: 30 * time.Second}, sampleServices())
	if err != nil {
		t.Fatalf("run status with since: %v", err)
	}
	assertContains(t, out, "api", "pkg", "timer")
	assertNotContains(t, out, "worker")

	out, err = runStatus(t, nil, Options{IncludeDisabled: true, Since: 30 * time.Second, NoMono: true}, sampleServices())
	if err != nil {
		t.Fatalf("run status with wall since: %v", err)
	}
	assertContains(t, out, "api", "pkg", "timer")
	assertNotContains(t, out, "worker")
}

func TestSystemdStoreUsesStructuredRunner(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string]string{
			"systemctl\x00list-unit-files\x00--type=service\x00--no-legend\x00--no-pager":     "api.service enabled enabled\npkg.service disabled enabled\napport-coredump-hook@.service static -\n",
			"systemctl\x00list-units\x00--type=service\x00--all\x00--no-legend\x00--no-pager": "api.service loaded active running API\n● pkg.service loaded inactive dead Package\napport-coredump-hook@.service loaded inactive dead Hook\n",
		},
	}
	showKey := "systemctl\x00show\x00--no-pager\x00--property=Id\x00--property=FragmentPath\x00--property=UnitFileState\x00--property=ActiveState\x00--property=SubState\x00--property=MainPID\x00--property=ActiveEnterTimestamp\x00--property=ActiveEnterTimestampMonotonic\x00--property=InactiveEnterTimestamp\x00--property=InactiveEnterTimestampMonotonic\x00api.service\x00pkg.service"
	runner.outputs[showKey] = strings.Join([]string{
		"Id=api.service",
		"FragmentPath=/etc/otter/services/api.service",
		"UnitFileState=enabled",
		"ActiveState=active",
		"SubState=running",
		"MainPID=42",
		"ActiveEnterTimestamp=2026-04-27T11:59:50Z",
		"ActiveEnterTimestampMonotonic=90000000",
		"",
		"Id=pkg.service",
		"FragmentPath=/tmp/packages/p1/pkg.service",
		"UnitFileState=disabled",
		"ActiveState=inactive",
		"SubState=dead",
		"MainPID=0",
		"InactiveEnterTimestamp=2026-04-27T11:59:40Z",
		"InactiveEnterTimestampMonotonic=80000000",
		"",
	}, "\n")

	store := NewSystemdStore(runner, otterfs.New(otterfs.Config{
		SystemdServicePath: "/usr/lib/systemd/system",
		PackageServicePath: "/tmp/packages",
	}))
	services, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list services: %v", err)
	}
	if got, want := len(services), 2; got != want {
		t.Fatalf("len services = %d, want %d", got, want)
	}
	if services[0].Name != "api" || !services[0].Enabled || !services[0].Running || services[0].MainPID != 42 {
		t.Fatalf("unexpected api service: %+v", services[0])
	}
	if services[1].Name != "pkg" || services[1].Source != SourcePackage {
		t.Fatalf("unexpected package service: %+v", services[1])
	}

	if got, want := runner.calls, [][]string{
		{"systemctl", "list-unit-files", "--type=service", "--no-legend", "--no-pager"},
		{"systemctl", "list-units", "--type=service", "--all", "--no-legend", "--no-pager"},
		{"systemctl", "show", "--no-pager", "--property=Id", "--property=FragmentPath", "--property=UnitFileState", "--property=ActiveState", "--property=SubState", "--property=MainPID", "--property=ActiveEnterTimestamp", "--property=ActiveEnterTimestampMonotonic", "--property=InactiveEnterTimestamp", "--property=InactiveEnterTimestampMonotonic", "api.service", "pkg.service"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
}

func TestManagedSystemdStoreOnlyShowsManagedServices(t *testing.T) {
	root := t.TempDir()
	classic := filepath.Join(root, "classic")
	pkg := filepath.Join(root, "packages")
	writeTestFile(t, filepath.Join(classic, "api.service"), "[Unit]\n")
	writeTestFile(t, filepath.Join(classic, "apport-coredump-hook@.service"), "[Unit]\n")
	writeTestFile(t, filepath.Join(pkg, "p1", "pkg", "pkg.service"), "[Unit]\n")

	runner := &fakeRunner{outputs: map[string]string{}}
	showKey := "systemctl\x00show\x00--no-pager\x00--property=Id\x00--property=FragmentPath\x00--property=UnitFileState\x00--property=ActiveState\x00--property=SubState\x00--property=MainPID\x00--property=ActiveEnterTimestamp\x00--property=ActiveEnterTimestampMonotonic\x00--property=InactiveEnterTimestamp\x00--property=InactiveEnterTimestampMonotonic\x00api.service\x00pkg.service"
	runner.outputs[showKey] = strings.Join([]string{
		"Id=api.service",
		"FragmentPath=" + filepath.Join(classic, "api.service"),
		"UnitFileState=enabled",
		"ActiveState=active",
		"SubState=running",
		"",
		"Id=pkg.service",
		"FragmentPath=" + filepath.Join(pkg, "p1", "pkg", "pkg.service"),
		"UnitFileState=disabled",
		"ActiveState=inactive",
		"SubState=dead",
		"",
	}, "\n")

	store := NewManagedSystemdStore(runner, otterfs.New(otterfs.Config{
		ClassicServicePath: classic,
		PackageServicePath: pkg,
	}))
	services, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list managed services: %v", err)
	}
	if got, want := len(services), 2; got != want {
		t.Fatalf("len services = %d, want %d", got, want)
	}
	if services[0].Name != "api" || services[0].Source != SourceSystemd {
		t.Fatalf("unexpected api service: %+v", services[0])
	}
	if services[1].Name != "pkg" || services[1].Source != SourcePackage {
		t.Fatalf("unexpected package service: %+v", services[1])
	}
	if got, want := runner.calls, [][]string{
		{"systemctl", "show", "--no-pager", "--property=Id", "--property=FragmentPath", "--property=UnitFileState", "--property=ActiveState", "--property=SubState", "--property=MainPID", "--property=ActiveEnterTimestamp", "--property=ActiveEnterTimestampMonotonic", "--property=InactiveEnterTimestamp", "--property=InactiveEnterTimestampMonotonic", "api.service", "pkg.service"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
}

func runStatus(t *testing.T, args []string, opts Options, services []Service) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(context.Background(), args, opts, Dependencies{
		Store: fakeStore{services: services},
		Out:   &out,
		Now: func() time.Time {
			return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
		},
		MonoNow: func() int64 {
			return int64(100 * time.Second)
		},
	})
	return out.String(), err
}

func writeTestFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sampleServices() []Service {
	base := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	return []Service{
		{
			Name:             "worker",
			UnitName:         "worker.service",
			Source:           SourceSystemd,
			Enabled:          false,
			ActiveState:      "inactive",
			SubState:         "dead",
			InactiveTime:     base.Add(-90 * time.Second),
			InactiveTimeMono: int64(10 * time.Second),
		},
		{
			Name:           "api",
			UnitName:       "api.service",
			Source:         SourceSystemd,
			Enabled:        true,
			Running:        true,
			ActiveState:    "active",
			SubState:       "running",
			ActiveTime:     base.Add(-10 * time.Second),
			ActiveTimeMono: int64(90 * time.Second),
		},
		{
			Name:           "pkg",
			UnitName:       "pkg.service",
			Source:         SourcePackage,
			Enabled:        true,
			Running:        true,
			ActiveState:    "active",
			SubState:       "running",
			ActiveTime:     base.Add(-20 * time.Second),
			ActiveTimeMono: int64(80 * time.Second),
		},
		{
			Name:        "timer",
			UnitName:    "timer.service",
			Source:      SourceSystemd,
			Enabled:     false,
			Running:     true,
			ActiveState: "active",
			SubState:    "running",
			ActiveTime:  base.Add(-25 * time.Second),
		},
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

func assertNotContains(t *testing.T, text string, values ...string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(text, value) {
			t.Fatalf("expected %q not to contain %q", text, value)
		}
	}
}

func assertOrder(t *testing.T, text string, values ...string) {
	t.Helper()
	last := -1
	for _, value := range values {
		index := strings.Index(text, value)
		if index == -1 {
			t.Fatalf("expected %q to contain %q", text, value)
		}
		if index < last {
			t.Fatalf("expected %q after previous values in %q", value, text)
		}
		last = index
	}
}

type fakeStore struct {
	services []Service
}

func (f fakeStore) List(ctx context.Context) ([]Service, error) {
	return f.services, nil
}

type fakeRunner struct {
	outputs map[string]string
	calls   [][]string
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)
	key := strings.Join(call, "\x00")
	out, ok := f.outputs[key]
	if !ok {
		return nil, errors.New("unexpected command: " + strings.Join(call, " "))
	}
	return []byte(out), nil
}
