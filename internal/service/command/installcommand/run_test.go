package installcommand

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestRunNoInstallPrintsService(t *testing.T) {
	var out bytes.Buffer
	err := Run(context.Background(), []string{"/bin/echo", "hello world"}, Options{
		Name:      "echo",
		NoInstall: true,
	}, Dependencies{
		LookPath: func(file string) (string, error) { return file, nil },
		Out:      &out,
	})
	if err != nil {
		t.Fatalf("run install-command: %v", err)
	}
	if !strings.Contains(out.String(), "# Service: echo.service") {
		t.Fatalf("missing service header: %q", out.String())
	}
	if !strings.Contains(out.String(), "ExecStart=/bin/echo 'hello world'") {
		t.Fatalf("missing exec start: %q", out.String())
	}
}

func TestRunInstallsGeneratedService(t *testing.T) {
	installer := &fakeInstaller{}
	var out bytes.Buffer
	err := Run(context.Background(), []string{"echo hello"}, Options{
		Name:             "echo",
		WorkingDirectory: "-",
		NoEnable:         true,
		NoStart:          true,
	}, Dependencies{
		Installer: installer,
		Getwd:     func() (string, error) { return "/srv/app", nil },
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("run install-command: %v", err)
	}
	if installer.name != "echo" {
		t.Fatalf("installer name = %q, want echo", installer.name)
	}
	data := string(installer.data)
	if !strings.Contains(data, "WorkingDirectory=/srv/app") || !strings.Contains(data, "ExecStart=") {
		t.Fatalf("generated data = %q", data)
	}
}

type fakeInstaller struct {
	name string
	data []byte
}

func (f *fakeInstaller) Install(ctx context.Context, name string, data []byte) error {
	f.name = name
	f.data = append([]byte(nil), data...)
	return nil
}

type fakeRunner struct{}

func (fakeRunner) Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	return nil
}
