package regenerate

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
)

type Options struct {
	Restart    bool
	NotRestart bool
}

type Dependencies struct {
	Finder    Finder
	Generator Generator
	Prompter  Prompter
	FS        otterfs.Provider
	Runner    startcmd.Runner
	Out       io.Writer
	ErrOut    io.Writer
	In        io.Reader
}

type PackageService struct {
	Name       string
	PackageID  string
	SourceDir  string
	InstallDir string
	Children   []string
}

type Finder interface {
	FindPackage(ctx context.Context, serviceName string) (PackageService, error)
}

type Generator interface {
	Generate(ctx context.Context, service PackageService) error
}

type Prompter interface {
	Confirm(prompt string) (bool, error)
}

func Run(ctx context.Context, serviceName string, opts Options, deps Dependencies) error {
	name := strings.TrimSuffix(serviceName, ".service")
	service, err := finder(deps).FindPackage(ctx, name)
	if err != nil {
		return fmt.Errorf("service may not exist: %w", err)
	}
	if err := generator(deps).Generate(ctx, service); err != nil {
		return fmt.Errorf("regen failed: %w", err)
	}
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	if _, err := fmt.Fprintf(out, "Re-generate service %s success\n", name); err != nil {
		return err
	}
	restart, err := shouldRestart(opts, deps)
	if err != nil {
		return err
	}
	if restart {
		if _, err := fmt.Fprintf(out, "Will restart service %s\n", name); err != nil {
			return err
		}
		if err := run(ctx, []string{"restart", name}, deps); err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, "Success")
		return err
	}
	if _, err := fmt.Fprintf(out, "Won't restart service %s\n", name); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "Success")
	return err
}

func shouldRestart(opts Options, deps Dependencies) (bool, error) {
	if opts.Restart {
		return true, nil
	}
	if opts.NotRestart {
		return false, nil
	}
	return prompter(deps).Confirm("Do you want to restart the service? [y/n]: ")
}

func run(ctx context.Context, args []string, deps Dependencies) error {
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := deps.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}
	in := deps.In
	if in == nil {
		in = os.Stdin
	}
	if _, err := fmt.Fprintln(out, "systemctl "+strings.Join(args, " ")); err != nil {
		return err
	}
	return runner(deps).Run(ctx, args, in, out, errOut)
}

func finder(deps Dependencies) Finder {
	if deps.Finder != nil {
		return deps.Finder
	}
	fs := deps.FS
	if fs.Config().PackageServicePath == "" {
		fs = otterfs.Default()
	}
	return FSFinder{FS: fs}
}

func generator(deps Dependencies) Generator {
	if deps.Generator != nil {
		return deps.Generator
	}
	return CopyGenerator{}
}

func runner(deps Dependencies) startcmd.Runner {
	if deps.Runner != nil {
		return deps.Runner
	}
	return systemctlRunner{}
}

func prompter(deps Dependencies) Prompter {
	if deps.Prompter != nil {
		return deps.Prompter
	}
	return stdPrompter{In: deps.In, Out: deps.Out}
}

type FSFinder struct {
	FS otterfs.Provider
}

func (f FSFinder) FindPackage(ctx context.Context, serviceName string) (PackageService, error) {
	_ = ctx
	matches, err := filepath.Glob(filepath.Join(f.FS.PackageServicePath(), "*", serviceName, "*.service"))
	if err != nil {
		return PackageService{}, err
	}
	if len(matches) == 0 {
		matches, err = filepath.Glob(filepath.Join(f.FS.PackageServicePath(), "*", "*", serviceName+".service"))
		if err != nil {
			return PackageService{}, err
		}
	}
	if len(matches) == 0 {
		return PackageService{}, fmt.Errorf("service folder is not exist")
	}
	installDir := filepath.Dir(matches[0])
	serviceDir := filepath.Base(installDir)
	pkgDir := filepath.Dir(installDir)
	pkgID := filepath.Base(pkgDir)
	sourceDir := f.FS.ServicesInstallPathFor(pkgID, serviceDir)
	children, _ := children(installDir)
	return PackageService{
		Name:       serviceName,
		PackageID:  pkgID,
		SourceDir:  sourceDir,
		InstallDir: installDir,
		Children:   children,
	}, nil
}

func children(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.service"))
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		name := strings.TrimSuffix(filepath.Base(match), ".service")
		if !strings.HasPrefix(name, ".") {
			result = append(result, name)
		}
	}
	return result, nil
}

type CopyGenerator struct{}

func (CopyGenerator) Generate(ctx context.Context, service PackageService) error {
	_ = ctx
	serviceData, err := os.ReadFile(filepath.Join(service.SourceDir, "SERVICE"))
	if err != nil {
		return fmt.Errorf("cannot read service file: %w", err)
	}
	envData, err := os.ReadFile(filepath.Join(service.SourceDir, "ENV"))
	if err != nil {
		return fmt.Errorf("cannot read env file: %w", err)
	}
	if err := os.MkdirAll(service.InstallDir, 0o755); err != nil {
		return err
	}
	name := service.Name
	if name == "" {
		name = filepath.Base(service.InstallDir)
	}
	if err := os.WriteFile(filepath.Join(service.InstallDir, name+".service"), withHeader(name, serviceData), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(service.InstallDir, name+".env"), withHeader(name, envData), 0o644)
}

func withHeader(name string, data []byte) []byte {
	header := fmt.Sprintf(
		"##############################################################\n# DO NOT EDIT! Use \"otter service edit %s\" instead.\n##############################################################\n\n",
		name,
	)
	return append([]byte(header), data...)
}

type systemctlRunner struct{}

func (systemctlRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

type stdPrompter struct {
	In  io.Reader
	Out io.Writer
}

func (p stdPrompter) Confirm(prompt string) (bool, error) {
	out := p.Out
	if out == nil {
		out = os.Stdout
	}
	in := p.In
	if in == nil {
		in = os.Stdin
	}
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return false, err
	}
	var answer string
	if _, err := fmt.Fscanln(in, &answer); err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}
