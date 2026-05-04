package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tj/go-naturaldate"

	"github.com/jeffinity/otter/internal/otterfs"
	"github.com/jeffinity/otter/internal/service/command/servicefile"
)

const journalctlTimeFormat = "2006-01-02 15:04:05"

type Options struct {
	Follow          bool
	Lines           int
	Since           string
	Until           string
	Output          string
	PagerEnd        bool
	Reverse         bool
	ForceJournalctl bool
	NoColor         bool
}

type Dependencies struct {
	Finder   servicefile.Finder
	Runner   Runner
	LookPath func(file string) (string, error)
	FS       otterfs.Provider
	Out      io.Writer
	ErrOut   io.Writer
	In       io.Reader
	Now      func() time.Time
}

type Runner interface {
	Run(ctx context.Context, command string, args []string, in io.Reader, out io.Writer, errOut io.Writer) error
}

func Run(ctx context.Context, serviceName string, opts Options, deps Dependencies) error {
	logFile, err := findLogFile(ctx, serviceName, deps)
	if err != nil {
		return err
	}
	command, args, err := buildCommand(serviceName, logFile, opts, deps)
	if err != nil {
		return err
	}

	in, out, errOut := streams(deps)
	if _, err := fmt.Fprintln(out, Join(args)); err != nil {
		return err
	}

	return runner(deps).Run(ctx, command, args, in, out, errOut)
}

func findLogFile(ctx context.Context, serviceName string, deps Dependencies) (string, error) {
	finder := deps.Finder
	if finder == nil {
		finder = servicefile.FSFinder{FS: deps.FS}
	}
	file, err := finder.Find(ctx, serviceName)
	if err != nil {
		return "", err
	}
	data, err := servicefile.Read(file.Path)
	if err != nil {
		return "", err
	}
	return customLogFile(data)
}

func buildCommand(serviceName string, logFile string, opts Options, deps Dependencies) (string, []string, error) {
	lookPath := deps.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if opts.ForceJournalctl || logFile == "" {
		return useJournalctl(servicefile.NormalizeName(serviceName), opts, lookPath, deps.Now, deps.ErrOut)
	}
	return useLessOrTail(logFile, opts, lookPath)
}

func streams(deps Dependencies) (io.Reader, io.Writer, io.Writer) {
	in := deps.In
	if in == nil {
		in = os.Stdin
	}
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := deps.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}
	return in, out, errOut
}

func runner(deps Dependencies) Runner {
	if deps.Runner != nil {
		return deps.Runner
	}
	return execRunner{}
}

func customLogFile(data string) (string, error) {
	value, exists, err := servicefile.Option(data, "LogFile")
	if err != nil || !exists {
		return "", err
	}
	return value, nil
}

func useLessOrTail(logFile string, opts Options, lookPath func(string) (string, error)) (string, []string, error) {
	tail, err := lookPath("tail")
	if err != nil {
		return "", nil, fmt.Errorf("cannot found tail in PATH: %w", err)
	}
	command := tail
	args := []string{"tail"}

	lines := opts.Lines
	if lines == 0 {
		lines = 80
	}
	if lines > 0 {
		args = append(args, fmt.Sprintf("--lines=%d", lines))
	} else if lines < 0 {
		args = append(args, fmt.Sprintf("--lines=+%d", lines))
	}

	if opts.Follow {
		args = append(args, "-F", logFile)
		return command, args, nil
	}

	less, _ := lookPath("less")
	bash, _ := lookPath("bash")
	if bash != "" && less != "" {
		command = bash
		shellCmd := Join(args) + " " + Quote(logFile) + " 2>&1 | '" + less + "'"
		if opts.PagerEnd {
			shellCmd += " +G"
		}
		args = []string{"bash", "-c", shellCmd}
	}
	return command, args, nil
}

func useJournalctl(
	serviceName string,
	opts Options,
	lookPath func(string) (string, error),
	now func() time.Time,
	errOut io.Writer,
) (string, []string, error) {
	journalctl, err := lookPath("journalctl")
	if err != nil {
		return "", nil, fmt.Errorf("cannot found journalctl in PATH: %w", err)
	}
	args := journalctlArgs(serviceName, opts)
	args, timeErr := appendJournalTimes(args, opts, now, errOut)
	if timeErr != nil {
		return "", nil, timeErr
	}

	return journalctl, args, nil
}

func journalctlArgs(serviceName string, opts Options) []string {
	output := opts.Output
	if output == "" {
		output = "cat"
	}
	args := []string{"journalctl", "--unit=" + serviceName, "--output=" + output}
	args = appendJournalLines(args, opts)
	return appendJournalFlags(args, opts)
}

func appendJournalLines(args []string, opts Options) []string {
	lines := opts.Lines
	if lines == 0 && opts.Since == "" && opts.Until == "" {
		lines = 80
	}
	if lines > 0 {
		return append(args, fmt.Sprintf("--lines=%d", lines))
	}
	if lines < 0 {
		return append(args, "--lines=all")
	}
	return args
}

func appendJournalFlags(args []string, opts Options) []string {
	if opts.Follow {
		args = append(args, "--follow")
	}
	if opts.PagerEnd {
		args = append(args, "--pager-end")
	}
	if opts.Reverse {
		args = append(args, "--reverse")
	}
	return args
}

func appendJournalTimes(args []string, opts Options, now func() time.Time, errOut io.Writer) ([]string, error) {
	base := time.Now
	if now != nil {
		base = now
	}
	if opts.Since != "" {
		parsed, err := parseDateString(opts.Since, base(), true)
		if err != nil {
			return nil, fmt.Errorf("unknown start time: %w", err)
		}
		args = append(args, "--since="+parsed.Format(journalctlTimeFormat))
	}
	if opts.Until == "" {
		return args, nil
	}
	warnUntil(opts, errOut)
	parsed, err := parseDateString(opts.Until, base(), false)
	if err != nil {
		return nil, fmt.Errorf("unknown end time: %w", err)
	}
	return append(args, "--until="+parsed.Format(journalctlTimeFormat)), nil
}

func warnUntil(opts Options, errOut io.Writer) {
	if errOut != nil && opts.Follow {
		_, _ = fmt.Fprintln(errOut, "Use --until with --follow may produce empty output in most cases")
	}
	if errOut != nil && opts.Lines > 0 {
		_, _ = fmt.Fprintln(errOut, "Use --until with --lines may produce empty output in most cases")
	}
}

func parseDateString(value string, base time.Time, start bool) (time.Time, error) {
	if parsed, ok, err := parseRelativeDate(value, base); ok || err != nil {
		return parsed, err
	}

	china := time.FixedZone("Asia/Shanghai", 8*60*60)
	if parsed, ok := parseFixedDate(value, base, start, china); ok {
		return parsed, nil
	}

	t, err := naturaldate.Parse(value, base)
	if err != nil {
		return time.Time{}, fmt.Errorf("unknown date")
	}
	return t, nil
}

func parseRelativeDate(value string, base time.Time) (time.Time, bool, error) {
	if strings.HasPrefix(value, "+") {
		duration, err := parseDuration(value[1:])
		if err != nil {
			return time.Time{}, true, fmt.Errorf("invalid duration %s", value[1:])
		}
		return base.Add(duration), true, nil
	}
	if strings.HasPrefix(value, "-") {
		duration, err := parseDuration(value[1:])
		if err != nil {
			return time.Time{}, true, fmt.Errorf("invalid duration %s", value[1:])
		}
		return base.Add(-duration), true, nil
	}
	if duration, err := parseDuration(value); err == nil {
		return base.Add(-duration), true, nil
	}
	return time.Time{}, false, nil
}

func parseFixedDate(value string, base time.Time, start bool, china *time.Location) (time.Time, bool) {
	if strings.Contains(value, "T") {
		t, err := time.ParseInLocation("2006-01-02T15:04:05", value, china)
		return t, err == nil
	}
	if strings.Contains(value, " ") {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, china); err == nil {
			return t, true
		}
		if t, err := time.ParseInLocation("2006-01-02 15:04", value, china); err == nil {
			return endIfNeeded(t, time.Second, start), true
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", value, china); err == nil {
		return endIfNeeded(t, 24*time.Hour, start), true
	}
	if day, err := strconv.Atoi(value); err == nil && day >= 1 && day <= 31 {
		base = base.In(china)
		t := time.Date(base.Year(), base.Month(), day, 0, 0, 0, 0, china)
		return endIfNeeded(t, 24*time.Hour, start), true
	}
	return time.Time{}, false
}

func endIfNeeded(t time.Time, span time.Duration, start bool) time.Time {
	if start {
		return t
	}
	return t.Add(span - time.Nanosecond)
}

func parseDuration(value string) (time.Duration, error) {
	if d, err := time.ParseDuration(value); err == nil {
		return d, nil
	}
	if value == "" {
		return 0, fmt.Errorf("empty duration")
	}
	var total time.Duration
	for value != "" {
		duration, rest, err := parseDurationPart(value)
		if err != nil {
			return 0, err
		}
		total += duration
		value = rest
	}
	return total, nil
}

func parseDurationPart(value string) (time.Duration, string, error) {
	numberEnd := durationNumberEnd(value)
	if numberEnd == 0 {
		return 0, "", fmt.Errorf("invalid duration")
	}
	unitEnd := durationUnitEnd(value, numberEnd)
	scale, ok := durationUnits[value[numberEnd:unitEnd]]
	if !ok {
		return 0, "", fmt.Errorf("invalid duration unit")
	}
	amount, err := strconv.ParseFloat(value[:numberEnd], 64)
	if err != nil {
		return 0, "", err
	}
	return time.Duration(amount * float64(scale)), value[unitEnd:], nil
}

func durationNumberEnd(value string) int {
	i := 0
	for i < len(value) && ((value[i] >= '0' && value[i] <= '9') || value[i] == '.') {
		i++
	}
	return i
}

func durationUnitEnd(value string, start int) int {
	i := start
	for i < len(value) && (value[i] < '0' || value[i] > '9') && value[i] != '.' {
		i++
	}
	return i
}

var durationUnits = map[string]time.Duration{
	"ns": time.Nanosecond,
	"us": time.Microsecond,
	"µs": time.Microsecond,
	"μs": time.Microsecond,
	"ms": time.Millisecond,
	"s":  time.Second,
	"m":  time.Minute,
	"h":  time.Hour,
	"d":  24 * time.Hour,
	"w":  7 * 24 * time.Hour,
}

func Join(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, Quote(arg))
	}
	return strings.Join(quoted, " ")
}

func Quote(arg string) string {
	if arg == "" {
		return "''"
	}
	if isSafeArg(arg) {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
}

func isSafeArg(arg string) bool {
	for _, r := range arg {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case strings.ContainsRune("_@%+=:,./-", r):
		default:
			return false
		}
	}
	return true
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, command string, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	cmd := exec.CommandContext(ctx, command, args[1:]...)
	cmd.Args = args
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
