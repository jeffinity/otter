package showproperty

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

var defaultProperties = []string{
	"LoadState",
	"ActiveState",
	"SubState",
	"ExecMainPID",
	"ExecMainStartTimestamp",
	"ExecMainExitTimestamp",
	"ActiveEnterTimestamp",
	"ActiveExitTimestamp",
	"InactiveEnterTimestamp",
	"InactiveExitTimestamp",
	"StateChangeTimestamp",
	"UnitFileState",
	"NeedDaemonReload",
}

var timeProperties = map[string]struct{}{
	"ExecMainExitTimestamp":  {},
	"InactiveEnterTimestamp": {},
	"ActiveExitTimestamp":    {},
	"StateChangeTimestamp":   {},
	"ConditionTimestamp":     {},
	"AssertTimestamp":        {},
	"WatchdogTimestamp":      {},
	"InactiveExitTimestamp":  {},
	"ActiveEnterTimestamp":   {},
	"ExecMainStartTimestamp": {},
}

type Options struct {
	All     bool
	NoColor bool
}

type Dependencies struct {
	Getter Getter
	Runner Runner
	Out    io.Writer
	Now    func() time.Time
}

type Getter interface {
	Get(ctx context.Context, serviceName string) (Properties, error)
}

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type Properties map[string]string

func Run(ctx context.Context, serviceName string, opts Options, deps Dependencies) error {
	getter := deps.Getter
	if getter == nil {
		getter = NewSystemctlGetter(deps.Runner)
	}
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	now := time.Now
	if deps.Now != nil {
		now = deps.Now
	}

	trimmedName := strings.TrimSuffix(serviceName, ".service")
	properties, err := getter.Get(ctx, trimmedName)
	if err != nil {
		return err
	}
	for _, key := range propertyKeys(properties, opts) {
		if _, err := fmt.Fprintf(out, "%s: %s\n", colorKey(key, opts), renderValue(key, properties, opts, now)); err != nil {
			return err
		}
	}
	return nil
}

func propertyKeys(properties Properties, opts Options) []string {
	keys := append([]string(nil), defaultProperties...)
	if !opts.All {
		return keys
	}

	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		seen[key] = struct{}{}
	}
	extra := make([]string, 0, len(properties))
	for key := range properties {
		if _, ok := seen[key]; !ok {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	return append(keys, extra...)
}

func renderValue(key string, properties Properties, opts Options, now func() time.Time) string {
	value, ok := properties[key]
	if !ok {
		return colorize("<no value>", ansiYellow, opts.NoColor)
	}
	if _, ok := timeProperties[key]; ok {
		t, ok := parseTime(value)
		if !ok {
			return value
		}
		if t.IsZero() {
			return "-"
		}
		return fmt.Sprintf("%s (%s)", t.Format(time.RFC3339), relativeTime(t, now()))
	}
	if key == "NeedDaemonReload" && value == "true" {
		return colorize(value, ansiRed, opts.NoColor)
	}
	return value
}

func colorKey(key string, opts Options) string {
	return colorize(key, ansiBlue, opts.NoColor)
}

type ansiColor string

const (
	ansiRed    ansiColor = "\x1b[31m"
	ansiBlue   ansiColor = "\x1b[34m"
	ansiYellow ansiColor = "\x1b[33m"
	ansiReset  ansiColor = "\x1b[0m"
)

func colorize(text string, color ansiColor, noColor bool) string {
	if noColor {
		return text
	}
	return string(color) + text + string(ansiReset)
}

func parseTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "n/a" || value == "0" {
		return time.Time{}, true
	}
	if micros, err := strconv.ParseInt(value, 10, 64); err == nil {
		if micros == 0 {
			return time.Time{}, true
		}
		return time.UnixMicro(micros), true
	}
	layouts := []string{
		time.RFC3339,
		"Mon 2006-01-02 15:04:05 MST",
		"Mon 2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			if t.Unix() <= 0 {
				return time.Time{}, true
			}
			return t, true
		}
	}
	return time.Time{}, false
}

func relativeTime(t time.Time, now time.Time) string {
	duration := now.Sub(t)
	if duration < 0 {
		duration = 0
	}
	duration = duration.Round(time.Second)
	switch {
	case duration >= 24*time.Hour:
		return fmt.Sprintf("%dd ago", int(duration/(24*time.Hour)))
	case duration >= time.Hour:
		return fmt.Sprintf("%dh ago", int(duration/time.Hour))
	case duration >= time.Minute:
		return fmt.Sprintf("%dm ago", int(duration/time.Minute))
	default:
		return fmt.Sprintf("%ds ago", int(math.Max(0, duration.Seconds())))
	}
}

type SystemctlGetter struct {
	runner Runner
}

func NewSystemctlGetter(runner Runner) *SystemctlGetter {
	if runner == nil {
		runner = execRunner{}
	}
	return &SystemctlGetter{runner: runner}
}

func (g *SystemctlGetter) Get(ctx context.Context, serviceName string) (Properties, error) {
	unitName := strings.TrimSuffix(serviceName, ".service") + ".service"
	out, err := g.runner.Run(ctx, "systemctl", "show", unitName, "--no-pager")
	if err != nil {
		return nil, fmt.Errorf("systemctl show %s: %w", unitName, err)
	}
	return parseProperties(string(out)), nil
}

func parseProperties(out string) Properties {
	properties := Properties{}
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		properties[key] = value
	}
	return properties
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}
