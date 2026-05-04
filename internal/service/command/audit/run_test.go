package audit

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRunWritesRecord(t *testing.T) {
	writer := &fakeWriter{}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	err := Run(context.Background(), Options{ServiceName: "api", ActionName: "start"}, Dependencies{
		Writer:  writer,
		Now:     func() time.Time { return now },
		Environ: func() []string { return []string{"A=B"} },
	})
	if err != nil {
		t.Fatalf("run audit: %v", err)
	}
	if writer.record.Services[0] != "api" || writer.record.Actions[0] != "start" {
		t.Fatalf("record = %+v", writer.record)
	}
	if writer.record.Environment["A"] != "B" {
		t.Fatalf("environment = %+v", writer.record.Environment)
	}
}

func TestRunBypassSkipsWrite(t *testing.T) {
	writer := &fakeWriter{}
	err := Run(context.Background(), Options{ServiceName: "api", ActionName: "start"}, Dependencies{
		Writer:  writer,
		Environ: func() []string { return []string{BypassEnv + "=-1"} },
	})
	if err != nil {
		t.Fatalf("run audit: %v", err)
	}
	if writer.called {
		t.Fatalf("writer should not be called")
	}
}

func TestRunAllowModeIgnoresWriteError(t *testing.T) {
	err := Run(context.Background(), Options{ServiceName: "api", ActionName: "start"}, Dependencies{
		Writer:  &fakeWriter{err: errors.New("boom")},
		Environ: func() []string { return []string{BypassEnv + "=1"} },
	})
	if err != nil {
		t.Fatalf("run audit should ignore error, got %v", err)
	}
}

func TestRunMissingFlagsPrintsRedHint(t *testing.T) {
	writer := &fakeWriter{}
	var out bytes.Buffer

	err := Run(context.Background(), Options{}, Dependencies{
		Writer: writer,
		Out:    &out,
	})
	if err != nil {
		t.Fatalf("run audit missing flags: %v", err)
	}
	if writer.called {
		t.Fatalf("writer should not be called")
	}
	if !strings.Contains(out.String(), "\x1b[31mMaybe you want to use `otter-audit`?\x1b[0m\n") {
		t.Fatalf("output = %q", out.String())
	}
}

type fakeWriter struct {
	record Record
	called bool
	err    error
}

func (f *fakeWriter) Write(ctx context.Context, record Record) error {
	f.called = true
	f.record = record
	return f.err
}
