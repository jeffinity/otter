package logx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	clog "github.com/charmbracelet/log"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  clog.Level
	}{
		{name: "debug", input: "debug", want: clog.DebugLevel},
		{name: "warn", input: "warn", want: clog.WarnLevel},
		{name: "warning", input: "warning", want: clog.WarnLevel},
		{name: "error", input: "error", want: clog.ErrorLevel},
		{name: "fatal", input: "fatal", want: clog.FatalLevel},
		{name: "empty default info", input: "", want: clog.InfoLevel},
		{name: "unknown default info", input: "trace", want: clog.InfoLevel},
		{name: "trim and case", input: "  DeBuG ", want: clog.DebugLevel},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseLevel(tc.input)
			if got != tc.want {
				t.Fatalf("parseLevel(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestDefaultStylesContainsLevels(t *testing.T) {
	t.Parallel()

	styles := defaultStyles()
	if styles == nil {
		t.Fatal("defaultStyles() returned nil")
	}
	levels := []clog.Level{
		clog.DebugLevel,
		clog.InfoLevel,
		clog.WarnLevel,
		clog.ErrorLevel,
		clog.FatalLevel,
	}
	for _, lv := range levels {
		if _, ok := styles.Levels[lv]; !ok {
			t.Fatalf("styles.Levels missing %v", lv)
		}
	}
}

func TestInitAndLogOutput(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Prefix:    "unit",
		Level:     "debug",
		Writer:    &buf,
		Timestamp: false,
		Caller:    false,
	})

	Debug("debug message")
	Infof("hello %s", "world")
	Warn("warn message")

	out := buf.String()
	if !strings.Contains(out, "unit") {
		t.Fatalf("expected prefix in output, got: %s", out)
	}
	if !strings.Contains(out, "debug message") {
		t.Fatalf("expected debug message in output, got: %s", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Fatalf("expected infof message in output, got: %s", out)
	}
	if !strings.Contains(out, "warn message") {
		t.Fatalf("expected warn message in output, got: %s", out)
	}
}

func TestSetPrefixAndErrorErr(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Prefix:    "old",
		Writer:    &buf,
		Timestamp: false,
		Caller:    false,
	})

	SetPrefix("new-prefix")
	ErrorErr(errors.New("boom"), "operation failed", "key", "value")

	out := buf.String()
	if !strings.Contains(out, "new-prefix") {
		t.Fatalf("expected updated prefix in output, got: %s", out)
	}
	if !strings.Contains(out, "operation failed") {
		t.Fatalf("expected error message in output, got: %s", out)
	}
	if !strings.Contains(out, "boom") {
		t.Fatalf("expected wrapped error in output, got: %s", out)
	}
	if !strings.Contains(out, "key") || !strings.Contains(out, "value") {
		t.Fatalf("expected keyvals in output, got: %s", out)
	}
}

func TestWithReturnsLogger(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Prefix:    "with",
		Writer:    &buf,
		Timestamp: false,
		Caller:    false,
	})

	l := With("k", "v")
	if l == nil {
		t.Fatal("With() returned nil logger")
	}
	l.Info("message")
	out := buf.String()
	if !strings.Contains(out, "message") {
		t.Fatalf("expected message in output, got: %s", out)
	}
	if !strings.Contains(out, "k") || !strings.Contains(out, "v") {
		t.Fatalf("expected keyvals in output, got: %s", out)
	}
}
