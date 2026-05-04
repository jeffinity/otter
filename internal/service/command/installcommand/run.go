package installcommand

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
	installservicecmd "github.com/jeffinity/otter/internal/service/command/installservice"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
)

type Options struct {
	Name             string
	WorkingDirectory string
	NoInstall        bool
	NoEnable         bool
	NoStart          bool
}

type Dependencies struct {
	Installer installservicecmd.Installer
	FS        otterfs.Provider
	Runner    startcmd.Runner
	LookPath  func(string) (string, error)
	Getwd     func() (string, error)
	Out       io.Writer
	ErrOut    io.Writer
	In        io.Reader
}

func Run(ctx context.Context, args []string, opts Options, deps Dependencies) error {
	if opts.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if len(args) == 0 {
		return fmt.Errorf("command is required")
	}

	cmdArgs, err := normalizeArgs(args, deps)
	if err != nil {
		return err
	}
	wd, err := normalizeWD(opts.WorkingDirectory, deps)
	if err != nil {
		return err
	}
	content := Generate(opts.Name, cmdArgs, wd)
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	if _, err := fmt.Fprintf(out, "#############\n# Service: %s.service\n#############\n%s\n", opts.Name, content); err != nil {
		return err
	}
	if opts.NoInstall {
		return nil
	}
	if _, err := fmt.Fprintln(out, "#############"); err != nil {
		return err
	}
	return installservicecmd.Install(ctx, opts.Name, []byte(content), installservicecmd.Options{
		Name:     opts.Name,
		NoEnable: opts.NoEnable,
		NoStart:  opts.NoStart,
	}, installservicecmd.Dependencies{
		Installer: deps.Installer,
		FS:        deps.FS,
		Runner:    deps.Runner,
		Out:       out,
		ErrOut:    deps.ErrOut,
		In:        deps.In,
	})
}

func Generate(name string, args []string, wd string) string {
	var b strings.Builder
	b.WriteString("[Unit]\nDescription=")
	b.WriteString(name)
	b.WriteString("\n\n[Service]\n")
	if wd != "" {
		b.WriteString("WorkingDirectory=")
		b.WriteString(wd)
		b.WriteByte('\n')
	}
	b.WriteString("ExecStart=")
	b.WriteString(shellJoin(args))
	b.WriteString("\n\n[Install]\nWantedBy=multi-user.target")
	return b.String()
}

func normalizeArgs(args []string, deps Dependencies) ([]string, error) {
	if len(args) == 1 && strings.Contains(args[0], " ") {
		parts, err := shellSplit(args[0])
		if err != nil {
			return nil, err
		}
		args = parts
	}
	args = append([]string(nil), args...)
	arg0 := args[0]
	var err error
	if filepath.IsAbs(arg0) {
		args[0], err = lookPath(deps)(arg0)
		if err != nil {
			return nil, fmt.Errorf("cannot find command %s: %w", arg0, err)
		}
		return args, nil
	}
	args[0], err = filepath.Abs(arg0)
	if err != nil {
		return nil, fmt.Errorf("invalid relative path %s: %w", arg0, err)
	}
	return args, nil
}

func normalizeWD(wd string, deps Dependencies) (string, error) {
	if wd == "-" {
		return getwd(deps)()
	}
	if wd == "" || filepath.IsAbs(wd) {
		return wd, nil
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		return "", fmt.Errorf("invalid relative path %s: %w", wd, err)
	}
	return abs, nil
}

func lookPath(deps Dependencies) func(string) (string, error) {
	if deps.LookPath != nil {
		return deps.LookPath
	}
	return exec.LookPath
}

func getwd(deps Dependencies) func() (string, error) {
	if deps.Getwd != nil {
		return deps.Getwd
	}
	return os.Getwd
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\n'\"\\$`") {
			quoted = append(quoted, "'"+strings.ReplaceAll(arg, "'", `'\''`)+"'")
		} else {
			quoted = append(quoted, arg)
		}
	}
	return strings.Join(quoted, " ")
}

func shellSplit(s string) ([]string, error) {
	var args []string
	var b strings.Builder
	var quote rune
	escaped := false
	for _, r := range s {
		args = splitRune(args, &b, r, &quote, &escaped)
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("invalid shell command")
	}
	if b.Len() != 0 {
		args = append(args, b.String())
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("command is required")
	}
	return args, nil
}

func splitRune(args []string, b *strings.Builder, r rune, quote *rune, escaped *bool) []string {
	if *escaped {
		b.WriteRune(r)
		*escaped = false
		return args
	}
	if r == '\\' {
		*escaped = true
		return args
	}
	if *quote != 0 {
		writeQuotedRune(b, r, quote)
		return args
	}
	if r == '\'' || r == '"' {
		*quote = r
		return args
	}
	if isShellSpace(r) {
		return flushArg(args, b)
	}
	b.WriteRune(r)
	return args
}

func writeQuotedRune(b *strings.Builder, r rune, quote *rune) {
	if r == *quote {
		*quote = 0
		return
	}
	b.WriteRune(r)
}

func isShellSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

func flushArg(args []string, b *strings.Builder) []string {
	if b.Len() == 0 {
		return args
	}
	args = append(args, b.String())
	b.Reset()
	return args
}
