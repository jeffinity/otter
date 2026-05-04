package showproperty

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRunShowsDefaultProperties(t *testing.T) {
	out, err := runShow(t, "api", Options{}, Properties{
		"LoadState":              "loaded",
		"ActiveState":            "active",
		"SubState":               "running",
		"ExecMainPID":            "42",
		"ExecMainStartTimestamp": "2026-04-27T11:59:50Z",
		"ExecMainExitTimestamp":  "n/a",
		"ActiveEnterTimestamp":   "2026-04-27T11:59:40Z",
		"InactiveEnterTimestamp": "0",
		"UnitFileState":          "enabled",
		"NeedDaemonReload":       "true",
		"UnshownProperty":        "hidden",
	})
	if err != nil {
		t.Fatalf("run show-property: %v", err)
	}
	assertContains(t, out,
		"LoadState: loaded\n",
		"ActiveState: active\n",
		"ExecMainStartTimestamp: 2026-04-27T11:59:50Z (10s ago)\n",
		"ExecMainExitTimestamp: -\n",
		"ActiveEnterTimestamp: 2026-04-27T11:59:40Z (20s ago)\n",
		"ActiveExitTimestamp: <no value>\n",
		"InactiveEnterTimestamp: -\n",
		"NeedDaemonReload: true\n",
	)
	if strings.Contains(out, "UnshownProperty") {
		t.Fatalf("default output should not include extra property: %q", out)
	}
}

func TestRunTrimsServiceSuffix(t *testing.T) {
	getter := &fakeGetter{properties: Properties{"LoadState": "loaded"}}
	var out bytes.Buffer
	err := Run(context.Background(), "api.service", Options{}, Dependencies{Getter: getter, Out: &out})
	if err != nil {
		t.Fatalf("run show-property: %v", err)
	}
	if getter.serviceName != "api" {
		t.Fatalf("service name = %q, want %q", getter.serviceName, "api")
	}
}

func TestRunShowsAllProperties(t *testing.T) {
	out, err := runShow(t, "api", Options{All: true}, Properties{
		"LoadState": "loaded",
		"Zeta":      "z",
		"Alpha":     "a",
	})
	if err != nil {
		t.Fatalf("run show-property --all: %v", err)
	}
	loadIndex := strings.Index(out, "LoadState: loaded\n")
	alphaIndex := strings.Index(out, "Alpha: a\n")
	zetaIndex := strings.Index(out, "Zeta: z\n")
	if loadIndex == -1 || alphaIndex == -1 || zetaIndex == -1 {
		t.Fatalf("missing expected properties: %q", out)
	}
	if !(loadIndex < alphaIndex && alphaIndex < zetaIndex) {
		t.Fatalf("extra properties should be appended in sorted order: %q", out)
	}
}

func TestRunReturnsGetterError(t *testing.T) {
	wantErr := errors.New("get failed")
	var out bytes.Buffer
	err := Run(context.Background(), "api", Options{}, Dependencies{
		Getter: &fakeGetter{err: wantErr},
		Out:    &out,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected getter error, got %v", err)
	}
}

func TestSystemctlGetterUsesRunner(t *testing.T) {
	runner := &fakeRunner{out: "LoadState=loaded\nEmpty=\nIgnoredLine\n"}
	getter := NewSystemctlGetter(runner)
	properties, err := getter.Get(context.Background(), "api.service")
	if err != nil {
		t.Fatalf("get properties: %v", err)
	}
	if got, want := runner.calls, [][]string{{"systemctl", "show", "api.service", "--no-pager"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
	if properties["LoadState"] != "loaded" || properties["Empty"] != "" {
		t.Fatalf("unexpected properties: %#v", properties)
	}
	if _, ok := properties["IgnoredLine"]; ok {
		t.Fatalf("ignored line should not be parsed: %#v", properties)
	}
}

func runShow(t *testing.T, serviceName string, opts Options, properties Properties) (string, error) {
	t.Helper()
	opts.NoColor = true
	return runShowColor(t, serviceName, opts, properties)
}

func runShowColor(t *testing.T, serviceName string, opts Options, properties Properties) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(context.Background(), serviceName, opts, Dependencies{
		Getter: &fakeGetter{properties: properties},
		Out:    &out,
		Now: func() time.Time {
			return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
		},
	})
	return out.String(), err
}

func assertContains(t *testing.T, text string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(text, value) {
			t.Fatalf("expected %q to contain %q", text, value)
		}
	}
}

type fakeGetter struct {
	serviceName string
	properties  Properties
	err         error
}

func (f *fakeGetter) Get(ctx context.Context, serviceName string) (Properties, error) {
	f.serviceName = serviceName
	return f.properties, f.err
}

type fakeRunner struct {
	out   string
	calls [][]string
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	return []byte(f.out), nil
}
