package logx

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	clog "github.com/charmbracelet/log"
)

// Config controls global logger behavior.
type Config struct {
	Prefix     string
	Level      string
	Writer     io.Writer
	Timestamp  bool
	Caller     bool
	TimeFormat string
}

// Init initializes and installs a colorful default logger.
func Init(cfg Config) *clog.Logger {
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stderr
	}

	logger := clog.NewWithOptions(writer, clog.Options{
		Prefix:          cfg.Prefix,
		Level:           parseLevel(cfg.Level),
		ReportTimestamp: cfg.Timestamp,
		ReportCaller:    cfg.Caller,
		TimeFormat:      cfg.TimeFormat,
	})
	logger.SetStyles(defaultStyles())
	clog.SetDefault(logger)
	return logger
}

// L returns current global logger.
func L() *clog.Logger {
	return clog.Default()
}

func With(keyvals ...any) *clog.Logger {
	return clog.With(keyvals...)
}

func SetPrefix(prefix string) {
	clog.SetPrefix(prefix)
}

func SetTimeFormat(format string) {
	clog.SetTimeFormat(format)
}

func Debug(msg any, keyvals ...any) { clog.Debug(msg, keyvals...) }
func Info(msg any, keyvals ...any)  { clog.Info(msg, keyvals...) }
func Warn(msg any, keyvals ...any)  { clog.Warn(msg, keyvals...) }
func Error(msg any, keyvals ...any) { clog.Error(msg, keyvals...) }
func Fatal(msg any, keyvals ...any) { clog.Fatal(msg, keyvals...) }

func Debugf(format string, args ...any) { clog.Debugf(format, args...) }
func Infof(format string, args ...any)  { clog.Infof(format, args...) }
func Warnf(format string, args ...any)  { clog.Warnf(format, args...) }
func Errorf(format string, args ...any) { clog.Errorf(format, args...) }
func Fatalf(format string, args ...any) { clog.Fatalf(format, args...) }

func ErrorErr(err error, msg string, keyvals ...any) {
	kvs := make([]any, 0, len(keyvals)+2)
	kvs = append(kvs, keyvals...)
	kvs = append(kvs, "error", err)
	clog.Error(msg, kvs...)
}

func parseLevel(level string) clog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return clog.DebugLevel
	case "warn", "warning":
		return clog.WarnLevel
	case "error":
		return clog.ErrorLevel
	case "fatal":
		return clog.FatalLevel
	default:
		return clog.InfoLevel
	}
}

func defaultStyles() *clog.Styles {
	styles := clog.DefaultStyles()
	styles.Prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styles.Key = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("245"))
	styles.Value = lipgloss.NewStyle().Foreground(lipgloss.Color("254"))
	styles.Separator = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("241"))

	styles.Levels[clog.DebugLevel] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	styles.Levels[clog.InfoLevel] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	styles.Levels[clog.WarnLevel] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	styles.Levels[clog.ErrorLevel] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styles.Levels[clog.FatalLevel] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("160"))
	return styles
}
