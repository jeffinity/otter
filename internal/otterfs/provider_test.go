package otterfs

import (
	"testing"
)

func TestDefaultProviderUsesOtterPaths(t *testing.T) {
	fs := Default()

	for name, tc := range map[string]struct {
		got  string
		want string
	}{
		"ConfigFilePath":               {got: fs.ConfigFilePath(), want: "/etc/otter/.config"},
		"MachineIDFilePath":            {got: fs.MachineIDFilePath(), want: "/etc/otter/machine-id"},
		"DatabaseFilePath":             {got: fs.DatabaseFilePath(), want: "/etc/otter/otter-service.db"},
		"ClusterCurrentServerFilePath": {got: fs.ClusterCurrentServerFilePath(), want: "/etc/otter/target"},
		"ClusterServersFilePath":       {got: fs.ClusterServersFilePath(), want: "/etc/otter/targets"},
		"RolesDirectoryPath":           {got: fs.RolesDirectoryPath(), want: "/etc/otter/roles"},
		"DataPath":                     {got: fs.DataPath(), want: "/data/.otter/otter-packages"},
		"ClassicServicePath":           {got: fs.ClassicServicePath(), want: "/etc/otter/services"},
		"PackageServicePath":           {got: fs.PackageServicePath(), want: "/etc/otter/services/.do-not-edit"},
		"SystemScriptsPath":            {got: fs.SystemScriptsPath(), want: "/etc/otter/scripts"},
		"PackageInstallPath":           {got: fs.PackageInstallPath(), want: "/data/.otter/otter-packages/packages"},
		"PackageInstallPathFor": {
			got:  fs.PackageInstallPathFor("pkg"),
			want: "/data/.otter/otter-packages/packages/pkg",
		},
		"PackageServiceBasePath": {
			got:  fs.PackageServiceBasePath("pkg"),
			want: "/etc/otter/services/.do-not-edit/pkg",
		},
		"PackageServicePathFor": {
			got:  fs.PackageServicePathFor("pkg", "api"),
			want: "/etc/otter/services/.do-not-edit/pkg/api",
		},
		"ServicesShortcutPath": {got: fs.ServicesShortcutPath(), want: "/data/.otter/otter-packages/services"},
		"ServicesInstallPath": {
			got:  fs.ServicesInstallPath("pkg"),
			want: "/data/.otter/otter-packages/packages/pkg/services",
		},
		"ServicesInstallPathFor": {
			got:  fs.ServicesInstallPathFor("pkg", "api"),
			want: "/data/.otter/otter-packages/packages/pkg/services/api",
		},
	} {
		if tc.got != tc.want {
			t.Fatalf("%s = %q, want %q", name, tc.got, tc.want)
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
