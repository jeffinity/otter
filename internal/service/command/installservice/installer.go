package installservice

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
)

type FSInstaller struct {
	FS otterfs.Provider
}

func (i FSInstaller) Install(ctx context.Context, name string, data []byte) error {
	_ = ctx
	if err := validUnit(data); err != nil {
		return err
	}

	classicPath := filepath.Join(i.FS.ClassicServicePath(), name+".service")
	systemdPath := i.FS.SystemdServicePathFor(name)
	dropInDir := i.FS.SystemdServiceDropInPathFor(name)

	if err := os.Remove(systemdPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(classicPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.RemoveAll(dropInDir); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(classicPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(systemdPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(dropInDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(classicPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Symlink(classicPath, systemdPath); err != nil {
		return fmt.Errorf("link-service-file: %w", err)
	}
	return nil
}

func validUnit(data []byte) error {
	text := string(bytes.TrimSpace(data))
	if text == "" {
		return fmt.Errorf("service file is empty")
	}
	if !hasSection(text, "Unit") || !hasSection(text, "Service") {
		return fmt.Errorf("invalid service file")
	}
	return nil
}

func hasSection(text string, name string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "["+name+"]") {
			return true
		}
	}
	return false
}
