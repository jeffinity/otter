package installdockercompose

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jeffinity/otter/internal/otterfs"
	installservicecmd "github.com/jeffinity/otter/internal/service/command/installservice"
	startcmd "github.com/jeffinity/otter/internal/service/command/start"
)

type Options struct {
	Name     string
	BaseDir  string
	Force    bool
	NoEnable bool
	NoStart  bool
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
	source, name, err := composeSource(args, opts.Name, deps)
	if err != nil {
		return err
	}
	baseDir := opts.BaseDir
	if baseDir == "" {
		baseDir = filepath.Join("/opt", name)
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", baseDir, err)
	}
	target := filepath.Join(baseDir, "docker-compose.yml")
	if err := copyTo(source, target, opts.Force); err != nil {
		return fmt.Errorf("cannot copy %s to %s: %w", source, target, err)
	}
	docker, err := lookPath(deps)("docker")
	if err != nil {
		return fmt.Errorf("cannot find docker: %w", err)
	}
	content := Generate(name, baseDir, docker)
	return installservicecmd.Install(ctx, name, []byte(content), installservicecmd.Options{
		Name:     name,
		NoEnable: opts.NoEnable,
		NoStart:  opts.NoStart,
	}, installservicecmd.Dependencies{
		Installer: deps.Installer,
		FS:        deps.FS,
		Runner:    deps.Runner,
		Out:       deps.Out,
		ErrOut:    deps.ErrOut,
		In:        deps.In,
	})
}

func Generate(name string, baseDir string, docker string) string {
	return fmt.Sprintf(`[Unit]
Description=%s

[Service]
WorkingDirectory=%s
ExecStart=%s compose -p %%N up --remove-orphans
TimeoutSec=3min

[Install]
WantedBy=multi-user.target

[X-Otter]
DockerComposeBaseDir=%s
`, name, baseDir, docker, baseDir)
}

func composeSource(args []string, name string, deps Dependencies) (string, string, error) {
	var composeDir, composeFile string
	if len(args) == 1 {
		p, err := filepath.Abs(args[0])
		if err != nil {
			return "", "", fmt.Errorf("cannot get absolute path of %s: %w", args[0], err)
		}
		stat, err := os.Stat(p)
		if err != nil {
			return "", "", fmt.Errorf("cannot stat %s: %w", p, err)
		}
		if stat.IsDir() {
			composeDir = p
		} else {
			composeFile = p
			composeDir = filepath.Dir(p)
		}
	} else {
		wd, err := getwd(deps)()
		if err != nil {
			return "", "", fmt.Errorf("cannot get current working directory: %w", err)
		}
		composeDir = wd
		if _, err := os.Stat(filepath.Join(composeDir, "docker-compose.yml")); err != nil {
			return "", "", fmt.Errorf("docker-compose.yml file path is required")
		}
	}
	if composeFile == "" {
		composeFile = filepath.Join(composeDir, "docker-compose.yml")
	}
	if name == "" {
		name = filepath.Base(composeDir)
		if name == "/" || name == "." || name == "" {
			return "", "", fmt.Errorf("service name is required")
		}
	}
	return composeFile, name, nil
}

func copyTo(from string, to string, force bool) error {
	fromReal, err := filepath.EvalSymlinks(from)
	if err == nil {
		from = fromReal
	}
	if from == to {
		return nil
	}
	if _, err := os.Stat(to); err == nil && !force {
		return fmt.Errorf("file exists")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
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
