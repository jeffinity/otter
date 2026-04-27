package otterfs

import (
	"strings"
	"testing"
)

func TestDefaultProviderUsesOtterPaths(t *testing.T) {
	fs := Default()

	paths := []string{
		fs.ConfigFilePath(),
		fs.MachineIDFilePath(),
		fs.DatabaseFilePath(),
		fs.ClusterCurrentServerFilePath(),
		fs.ClusterServersFilePath(),
		fs.RolesDirectoryPath(),
		fs.DataPath(),
		fs.ClassicServicePath(),
		fs.PackageServicePath(),
		fs.SystemScriptsPath(),
		fs.PackageInstallPath(),
		fs.PackageInstallPathFor("pkg"),
		fs.PackageServiceBasePath("pkg"),
		fs.PackageServicePathFor("pkg", "api"),
		fs.ServicesShortcutPath(),
		fs.ServicesInstallPath("pkg"),
		fs.ServicesInstallPathFor("pkg", "api"),
	}

	for _, p := range paths {
		if strings.Contains(p, "ambot") {
			t.Fatalf("path %q should not contain legacy name", p)
		}
	}
}

func TestSystemdServicePathsNormalizeSuffix(t *testing.T) {
	fs := Default()

	if got, want := fs.SystemdServicePathFor("api"), "/usr/lib/systemd/system/api.service"; got != want {
		t.Fatalf("SystemdServicePathFor without suffix = %q, want %q", got, want)
	}
	if got, want := fs.SystemdServicePathFor("api.service"), "/usr/lib/systemd/system/api.service"; got != want {
		t.Fatalf("SystemdServicePathFor with suffix = %q, want %q", got, want)
	}
	if got, want := fs.SystemdServiceDropInPathFor("api"), "/usr/lib/systemd/system/api.service.d"; got != want {
		t.Fatalf("SystemdServiceDropInPathFor = %q, want %q", got, want)
	}
}
