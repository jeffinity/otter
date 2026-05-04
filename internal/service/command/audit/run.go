package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const BypassEnv = "OTTER_AUDIT_BYPASS"

type Options struct {
	ServiceName string
	ActionName  string
}

type Dependencies struct {
	Writer  Writer
	Environ func() []string
	Out     io.Writer
	NoColor bool
	Now     func() time.Time
	Path    string
}

type Writer interface {
	Write(ctx context.Context, record Record) error
}

type Record struct {
	From        string            `json:"from"`
	Actions     []string          `json:"actions"`
	Services    []string          `json:"services"`
	Environment map[string]string `json:"environment"`
	Time        time.Time         `json:"time"`
}

func Run(ctx context.Context, opts Options, deps Dependencies) error {
	if opts.ServiceName == "" || opts.ActionName == "" {
		_, err := fmt.Fprintln(out(deps), colorize("Maybe you want to use `otter-audit`?", ansiRed, deps.NoColor))
		return err
	}
	mode := auditMode(deps)
	if mode == "-1" {
		return nil
	}
	record := Record{
		From:        "notify",
		Actions:     []string{opts.ActionName},
		Services:    []string{opts.ServiceName},
		Environment: environment(deps),
		Time:        now(deps),
	}
	if err := writer(deps).Write(ctx, record); err != nil && mode != "1" {
		return fmt.Errorf("Audit Fail: %w", err)
	}
	return nil
}

func out(deps Dependencies) io.Writer {
	if deps.Out != nil {
		return deps.Out
	}
	return os.Stdout
}

type ansiColor string

const (
	ansiRed   ansiColor = "\x1b[31m"
	ansiReset ansiColor = "\x1b[0m"
)

func colorize(text string, color ansiColor, noColor bool) string {
	if noColor {
		return text
	}
	return string(color) + text + string(ansiReset)
}

func writer(deps Dependencies) Writer {
	if deps.Writer != nil {
		return deps.Writer
	}
	return FileWriter{Path: deps.Path}
}

func now(deps Dependencies) time.Time {
	if deps.Now != nil {
		return deps.Now()
	}
	return time.Now()
}

func auditMode(deps Dependencies) string {
	return environment(deps)[BypassEnv]
}

func environment(deps Dependencies) map[string]string {
	environ := deps.Environ
	if environ == nil {
		environ = os.Environ
	}
	result := map[string]string{}
	for _, item := range environ() {
		for i, r := range item {
			if r == '=' {
				result[item[:i]] = item[i+1:]
				break
			}
		}
	}
	return result
}

type FileWriter struct {
	Path string
}

func (w FileWriter) Write(ctx context.Context, record Record) error {
	_ = ctx
	path := w.Path
	if path == "" {
		path = "/etc/otter/otter-core-audit.log"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(record)
}
