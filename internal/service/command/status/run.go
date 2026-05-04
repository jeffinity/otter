package status

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	if opts.SortAsc && opts.SortDesc {
		return fmt.Errorf("--asc and --desc cannot be apply in the meantime")
	}

	now := time.Now
	if deps.Now != nil {
		now = deps.Now
	}
	monoNow := readMonoNow
	if deps.MonoNow != nil {
		monoNow = deps.MonoNow
	}

	services, err := Select(ctx, args, opts, deps)
	if err != nil {
		return err
	}
	sortServices(services, opts, now(), monoNow())
	return render(deps.Out, services, opts, now(), monoNow())
}

func Select(ctx context.Context, args []string, opts Options, deps Dependencies) ([]Service, error) {
	if deps.Store == nil {
		return nil, errors.New("service store is required")
	}
	if opts.OnlyPackage && opts.OnlyClassic {
		return nil, fmt.Errorf("--only-package and --only-classic cannot be apply in the meantime")
	}

	services, err := deps.Store.List(ctx)
	if err != nil {
		return nil, err
	}
	patterns, err := normalizePatterns(args)
	if err != nil {
		return nil, err
	}
	services = filterSource(services, opts)
	if err := checkPatterns(services, patterns); err != nil {
		return nil, err
	}
	services = filterServices(services, patterns, opts)
	if len(services) == 0 {
		return nil, fmt.Errorf("no services after filter")
	}
	sortByName(services)
	return services, nil
}

func normalizePatterns(args []string) ([]string, error) {
	patterns := make([]string, 0, len(args))
	includeAll := false
	for _, arg := range args {
		pattern := strings.TrimSuffix(strings.TrimSpace(arg), ".service")
		if pattern == "" {
			continue
		}
		if pattern == "all" || pattern == "*" {
			includeAll = true
		}
		patterns = append(patterns, pattern)
	}
	if includeAll && len(patterns) != 1 {
		return nil, fmt.Errorf("if use `all` or `*`, it must be the only one service pattern")
	}
	if includeAll {
		return nil, nil
	}
	return patterns, nil
}

func filterSource(services []Service, opts Options) []Service {
	if !opts.OnlyPackage && !opts.OnlyClassic {
		return services
	}
	filtered := make([]Service, 0, len(services))
	for _, service := range services {
		switch {
		case opts.OnlyPackage && service.Source == SourcePackage:
			filtered = append(filtered, service)
		case opts.OnlyClassic && service.Source != SourcePackage:
			filtered = append(filtered, service)
		}
	}
	return filtered
}

func checkPatterns(services []Service, patterns []string) error {
	if len(patterns) == 0 {
		return nil
	}
	for _, pattern := range patterns {
		matched := false
		for _, service := range services {
			if matchPattern(pattern, service.Name) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("service pattern %s cannot be match by any service", pattern)
		}
	}
	return nil
}

func filterServices(services []Service, patterns []string, opts Options) []Service {
	filtered := make([]Service, 0, len(services))
	explicit := len(patterns) != 0
	for _, service := range services {
		if explicit {
			if matchAny(patterns, service.Name) {
				filtered = append(filtered, service)
			}
			continue
		}

		switch {
		case !opts.ExcludeEnabled && !opts.IncludeDisabled:
			if service.Enabled || service.Running {
				filtered = append(filtered, service)
			}
		case !opts.ExcludeEnabled && opts.IncludeDisabled:
			filtered = append(filtered, service)
		default:
			if !service.Enabled {
				filtered = append(filtered, service)
			}
		}
	}
	return filtered
}

func matchAny(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if matchPattern(pattern, name) {
			return true
		}
	}
	return false
}

func matchPattern(pattern, name string) bool {
	ok, err := filepath.Match(pattern, name)
	if err != nil {
		return pattern == name
	}
	return ok
}

func sortServices(services []Service, opts Options, now time.Time, nowMono int64) {
	sort.SliceStable(services, func(i, j int) bool {
		if opts.SortAsc || opts.SortDesc {
			left := serviceAge(services[i], opts, now, nowMono)
			right := serviceAge(services[j], opts, now, nowMono)
			if left != right {
				if opts.SortAsc {
					return left < right
				}
				return left > right
			}
		}
		return services[i].Name < services[j].Name
	})
}

func sortByName(services []Service) {
	sort.SliceStable(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
}

func render(out io.Writer, services []Service, opts Options, now time.Time, nowMono int64) error {
	if out == nil {
		out = os.Stdout
	}
	nameWidth := 0
	stateWidth := 0
	for _, service := range services {
		nameWidth = max(nameWidth, len(service.Name))
		stateWidth = max(stateWidth, len(stateText(service)))
	}

	for _, service := range services {
		if opts.Since != 0 && serviceAge(service, opts, now, nowMono) > opts.Since {
			continue
		}
		state := stateText(service)
		line := fmt.Sprintf(
			"%-*s    %s%s",
			nameWidth,
			service.Name,
			colorfulStateText(service, opts.NoColor),
			strings.Repeat(" ", stateWidth-len(state)),
		)
		if opts.IncludeTimeInfo {
			line += timeInfo(service, opts, now, nowMono)
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func stateText(service Service) string {
	state := serviceStateText(service)
	if !service.Enabled {
		state += ", disabled"
	}
	return state
}

func serviceStateText(service Service) string {
	state := service.ActiveState
	switch {
	case state == "active" && service.SubState != "":
		state = service.SubState
	case state != "" && service.SubState != "" && state != service.SubState:
		state += "/" + service.SubState
	case state == "":
		state = service.SubState
	}
	if state == "" {
		state = "unknown"
	}
	return state
}

func colorfulStateText(service Service, noColor bool) string {
	state := serviceStateText(service)
	if service.Running {
		state = colorize(state, ansiGreen, noColor)
	} else {
		state = colorize(state, ansiRed, noColor)
	}
	if !service.Enabled {
		state += ", " + colorize("disabled", ansiYellow, noColor)
	}
	return state
}

func timeInfo(service Service, opts Options, now time.Time, nowMono int64) string {
	showTime := service.showTime()
	if showTime.IsZero() {
		return "  (时间信息未知)"
	}

	age := serviceAge(service, opts, now, nowMono)
	info := fmt.Sprintf("  (%s, %s)", showTime.Format("2006-01-02 15:04:05"), relativeDuration(age))
	showMono := service.showTimeMono()
	if opts.NoMono {
		return info
	}
	if showMono == 0 {
		return info + colorize(" [无法获取 mono 时间]", ansiRed, opts.NoColor)
	}
	wallAge := now.Sub(showTime)
	if math.Abs(float64(wallAge-age)) > float64(10*time.Second) {
		return info + colorize(" [系统时间在任务启动后有修改]", ansiRed, opts.NoColor)
	}
	return info
}

type ansiColor string

const (
	ansiRed    ansiColor = "\x1b[31m"
	ansiGreen  ansiColor = "\x1b[32m"
	ansiYellow ansiColor = "\x1b[33m"
	ansiReset  ansiColor = "\x1b[0m"
)

func colorize(text string, color ansiColor, noColor bool) string {
	if noColor {
		return text
	}
	return string(color) + text + string(ansiReset)
}

func serviceAge(service Service, opts Options, now time.Time, nowMono int64) time.Duration {
	showTime := service.showTime()
	showMono := service.showTimeMono()
	if !opts.NoMono && showMono > 0 && nowMono > showMono {
		return time.Duration(nowMono - showMono)
	}
	if showTime.IsZero() {
		return time.Duration(math.MaxInt64)
	}
	age := now.Sub(showTime)
	if age < 0 {
		return 0
	}
	return age
}

func relativeDuration(duration time.Duration) string {
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
		return fmt.Sprintf("%ds ago", int(duration/time.Second))
	}
}

func readMonoNow() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	seconds, err := time.ParseDuration(fields[0] + "s")
	if err != nil {
		return 0
	}
	return int64(seconds)
}
