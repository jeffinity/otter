package edit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
	regeneratecmd "github.com/jeffinity/otter/internal/service/command/regenerate"
	"github.com/jeffinity/otter/internal/service/command/servicefile"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
)

type Dependencies struct {
	Finder       servicefile.Finder
	PackageFind  regeneratecmd.Finder
	Regenerator  regeneratecmd.Generator
	Prompter     Prompter
	Runner       Runner
	SystemRunner startcmd.Runner
	LookPath     func(string) (string, error)
	Getenv       func(string) string
	TempFile     func(pattern string) (*os.File, error)
	FS           otterfs.Provider
	Out          io.Writer
	ErrOut       io.Writer
	In           io.Reader
}

type Runner interface {
	Run(ctx context.Context, command string, args []string, in io.Reader, out io.Writer, errOut io.Writer) error
}

type Prompter interface {
	Choose(options []string) (string, error)
	Confirm(prompt string) (bool, error)
	Input(prompt string) (string, error)
}

func Run(ctx context.Context, serviceName string, deps Dependencies) error {
	name := servicefile.NormalizeName(serviceName)
	found, err := finder(deps).Find(ctx, name)
	if err != nil {
		return err
	}
	changed := []string{name}
	switch found.Source {
	case servicefile.SourceClassic:
		if err := editClassic(ctx, found, deps); err != nil {
			return err
		}
	case servicefile.SourcePackage:
		if err := editPackage(ctx, name, deps); err != nil {
			return err
		}
	default:
		return fmt.Errorf("service %s is not found", name)
	}
	return restartChanged(ctx, changed, deps)
}

func editClassic(ctx context.Context, service servicefile.File, deps Dependencies) error {
	_, editorPath, err := findEditor(deps)
	if err != nil {
		return err
	}
	if err := runner(deps).Run(ctx, editorPath, []string{service.Path}, deps.In, out(deps), errOut(deps)); err != nil {
		return err
	}
	return runSystemctl(ctx, []string{"daemon-reload"}, deps)
}

func editPackage(ctx context.Context, name string, deps Dependencies) error {
	editor, editorPath, err := findEditor(deps)
	if err != nil {
		return err
	}
	service, err := packageFinder(deps).FindPackage(ctx, name)
	if err != nil {
		return err
	}
	serviceTmp, envTmp, err := duplicate(service, deps)
	if err != nil {
		return err
	}
	defer os.Remove(serviceTmp)
	defer os.Remove(envTmp)

	for {
		choice, err := prompter(deps).Choose([]string{"1", "2", "3", "m", "y", "x"})
		if err != nil {
			return err
		}
		done, err := handleChoice(ctx, choice, editor, editorPath, serviceTmp, envTmp, deps)
		if err != nil {
			return err
		}
		if done {
			return savePackage(ctx, service, serviceTmp, envTmp, deps)
		}
		if choice == "x" {
			return nil
		}
	}
}

func handleChoice(
	ctx context.Context,
	choice string,
	editor string,
	editorPath string,
	serviceTmp string,
	envTmp string,
	deps Dependencies,
) (bool, error) {
	switch choice {
	case "y":
		return true, nil
	case "m":
		return false, printManualPaths(serviceTmp, envTmp, deps)
	case "1":
		return false, runner(deps).Run(ctx, editorPath, []string{serviceTmp}, deps.In, out(deps), errOut(deps))
	case "2":
		return false, runner(deps).Run(ctx, editorPath, []string{envTmp}, deps.In, out(deps), errOut(deps))
	case "3":
		return false, runner(deps).Run(ctx, editorPath, editBothArgs(editor, serviceTmp, envTmp), deps.In, out(deps), errOut(deps))
	default:
		return false, nil
	}
}

func printManualPaths(serviceTmp string, envTmp string, deps Dependencies) error {
	_, err := fmt.Fprintf(
		out(deps),
		"================================================\n| If you want to edit service file, please edit\n|  > %s\n| If you want to edit env file, please edit\n|  > %s\n================================================\n",
		serviceTmp,
		envTmp,
	)
	return err
}

func editBothArgs(editor string, serviceTmp string, envTmp string) []string {
	if editor == "vi" || editor == "vim" {
		return []string{"-O", serviceTmp, envTmp}
	}
	return []string{serviceTmp, envTmp}
}

func duplicate(service regeneratecmd.PackageService, deps Dependencies) (string, string, error) {
	serviceTmp, err := tempFile(deps)(service.Name + ".service")
	if err != nil {
		return "", "", err
	}
	defer serviceTmp.Close()
	envTmp, err := tempFile(deps)(service.Name + ".env")
	if err != nil {
		return "", "", err
	}
	defer envTmp.Close()
	if err := copyFile(filepath.Join(service.SourceDir, "SERVICE"), serviceTmp.Name()); err != nil {
		return "", "", err
	}
	if err := copyFile(filepath.Join(service.SourceDir, "ENV"), envTmp.Name()); err != nil {
		return "", "", err
	}
	return serviceTmp.Name(), envTmp.Name(), nil
}

func savePackage(ctx context.Context, service regeneratecmd.PackageService, serviceTmp string, envTmp string, deps Dependencies) error {
	servicePath := filepath.Join(service.SourceDir, "SERVICE")
	envPath := filepath.Join(service.SourceDir, "ENV")
	serviceChanged, err := changed(servicePath, serviceTmp)
	if err != nil {
		return err
	}
	envChanged, err := changed(envPath, envTmp)
	if err != nil {
		return err
	}
	if !serviceChanged && !envChanged {
		if _, err := fmt.Fprintln(out(deps), "Service and Environment file not change"); err != nil {
			return err
		}
		return nil
	}
	if _, err := prompter(deps).Input("Commit message: "); err != nil {
		return err
	}
	if err := saveOneFile(serviceChanged, serviceTmp, servicePath, "Service file", deps); err != nil {
		return err
	}
	if err := saveOneFile(envChanged, envTmp, envPath, "Environment file", deps); err != nil {
		return err
	}
	if err := generator(deps).Generate(ctx, service); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out(deps), "Re-generate service %s success\n", service.Name)
	return err
}

func saveOneFile(changed bool, from string, to string, label string, deps Dependencies) error {
	if !changed {
		_, err := fmt.Fprintf(out(deps), "%s not change\n", label)
		return err
	}
	if err := copyFile(from, to); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out(deps), "%s saved successfully\n", label)
	return err
}

func restartChanged(ctx context.Context, services []string, deps Dependencies) error {
	ok, err := prompter(deps).Confirm("Do you want to restart the service? [y/n]: ")
	if err != nil {
		return err
	}
	if !ok {
		_, err := fmt.Fprintf(out(deps), "Won't restart service %v\n", services)
		return err
	}
	if _, err := fmt.Fprintf(out(deps), "Will restart service %v\n", services); err != nil {
		return err
	}
	args := append([]string{"restart"}, services...)
	if err := runSystemctl(ctx, args, deps); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out(deps), "Success")
	return err
}

func findEditor(deps Dependencies) (string, string, error) {
	getenv := deps.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	look := deps.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if editor := getenv("EDITOR"); editor != "" {
		base := filepath.Base(editor)
		if base != "vi" && base != "vim" && base != "nano" {
			return "", "", fmt.Errorf("unknown editor %s", editor)
		}
		path, err := look(editor)
		return base, path, err
	}
	for _, editor := range []string{"vim", "vi", "nano"} {
		if path, err := look(editor); err == nil {
			return editor, path, nil
		}
	}
	return "", "", fmt.Errorf("no available editors can be used in [vim vi nano]")
}

func runSystemctl(ctx context.Context, args []string, deps Dependencies) error {
	if _, err := fmt.Fprintln(out(deps), "systemctl "+strings.Join(args, " ")); err != nil {
		return err
	}
	return systemRunner(deps).Run(ctx, args, deps.In, out(deps), errOut(deps))
}

func finder(deps Dependencies) servicefile.Finder {
	if deps.Finder != nil {
		return deps.Finder
	}
	return servicefile.FSFinder{FS: fs(deps)}
}

func packageFinder(deps Dependencies) regeneratecmd.Finder {
	if deps.PackageFind != nil {
		return deps.PackageFind
	}
	return regeneratecmd.FSFinder{FS: fs(deps)}
}

func generator(deps Dependencies) regeneratecmd.Generator {
	if deps.Regenerator != nil {
		return deps.Regenerator
	}
	return regeneratecmd.CopyGenerator{}
}

func prompter(deps Dependencies) Prompter {
	if deps.Prompter != nil {
		return deps.Prompter
	}
	return stdPrompter{In: deps.In, Out: deps.Out}
}

func runner(deps Dependencies) Runner {
	if deps.Runner != nil {
		return deps.Runner
	}
	return execRunner{}
}

func systemRunner(deps Dependencies) startcmd.Runner {
	if deps.SystemRunner != nil {
		return deps.SystemRunner
	}
	return systemctlRunner{}
}

func tempFile(deps Dependencies) func(string) (*os.File, error) {
	if deps.TempFile != nil {
		return deps.TempFile
	}
	return func(pattern string) (*os.File, error) { return os.CreateTemp("", pattern) }
}

func fs(deps Dependencies) otterfs.Provider {
	if deps.FS.Config().ClassicServicePath == "" {
		return otterfs.Default()
	}
	return deps.FS
}

func out(deps Dependencies) io.Writer {
	if deps.Out != nil {
		return deps.Out
	}
	return os.Stdout
}

func errOut(deps Dependencies) io.Writer {
	if deps.ErrOut != nil {
		return deps.ErrOut
	}
	return os.Stderr
}

func copyFile(from string, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}

func changed(a string, b string) (bool, error) {
	dataA, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	dataB, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(dataA, dataB), nil
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, command string, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
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

func (p stdPrompter) Choose(options []string) (string, error) {
	var value string
	if _, err := fmt.Fscanln(input(p.In), &value); err != nil {
		return "", err
	}
	for _, option := range options {
		if value == option {
			return value, nil
		}
	}
	return "", fmt.Errorf("invalid choice %s", value)
}

func (p stdPrompter) Confirm(prompt string) (bool, error) {
	if _, err := fmt.Fprint(output(p.Out), prompt); err != nil {
		return false, err
	}
	var answer string
	if _, err := fmt.Fscanln(input(p.In), &answer); err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

func (p stdPrompter) Input(prompt string) (string, error) {
	if _, err := fmt.Fprint(output(p.Out), prompt); err != nil {
		return "", err
	}
	var answer string
	_, err := fmt.Fscanln(input(p.In), &answer)
	return answer, err
}

func input(in io.Reader) io.Reader {
	if in != nil {
		return in
	}
	return os.Stdin
}

func output(out io.Writer) io.Writer {
	if out != nil {
		return out
	}
	return os.Stdout
}
