package view

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
	"github.com/jeffinity/otter/internal/service/command/servicefile"
)

type Options struct {
	NoColor bool
}

type Dependencies struct {
	Finder servicefile.Finder
	FS     otterfs.Provider
	Out    io.Writer
}

func Run(ctx context.Context, serviceName string, opts Options, deps Dependencies) error {
	finder := deps.Finder
	if finder == nil {
		finder = servicefile.FSFinder{FS: deps.FS}
	}
	file, err := finder.Find(ctx, serviceName)
	if err != nil {
		return err
	}

	data, err := servicefile.Read(file.Path)
	if err != nil {
		return err
	}

	printer := newPrinter(deps.Out)
	if file.Source == servicefile.SourcePackage {
		if err := printPackage(printer, data); err != nil {
			return err
		}
	} else if err := printer.Println(data); err != nil {
		return err
	}

	_ = showCompose(printer, data)
	return nil
}

type printer struct {
	out io.Writer
}

func newPrinter(out io.Writer) *printer {
	if out == nil {
		out = os.Stdout
	}
	return &printer{out: out}
}

func (p *printer) Println(a ...any) error {
	_, err := fmt.Fprintln(p.out, a...)
	return err
}

func printPackage(printer *printer, data string) error {
	lines := strings.Split(data, "\n")
	lines = trimPrefixEmpty(trimHeader(lines))
	for _, line := range lines {
		if err := printer.Println(line); err != nil {
			return err
		}
	}
	return nil
}

func showCompose(printer *printer, data string) error {
	baseDir, exists, err := servicefile.Option(data, "DockerComposeBaseDir")
	if err != nil || !exists || baseDir == "" {
		return err
	}

	composePath := filepath.Join(baseDir, "docker-compose.yml")
	composeData, err := os.ReadFile(composePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(composeData), "\n")
	lines = trimSuffixEmpty(trimPrefixEmpty(lines))

	if err := printer.Println(""); err != nil {
		return err
	}
	if err := printer.Println("#=========================="); err != nil {
		return err
	}
	if err := printer.Println("# Docker Compose: " + composePath); err != nil {
		return err
	}
	if err := printer.Println("#=========================="); err != nil {
		return err
	}
	for _, line := range lines {
		if err := printer.Println("# " + line); err != nil {
			return err
		}
	}
	return nil
}

func trimHeader(lines []string) []string {
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		return lines[i:]
	}
	return lines
}

func trimPrefixEmpty(lines []string) []string {
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		return lines[i:]
	}
	return nil
}

func trimSuffixEmpty(lines []string) []string {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		return lines[:i+1]
	}
	return nil
}
