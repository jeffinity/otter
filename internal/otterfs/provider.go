package otterfs

import (
	"path"
	"path/filepath"
	"strings"
)

type Config struct {
	ConfigFilePath               string
	MachineIDFilePath            string
	DatabaseFilePath             string
	ClusterCurrentServerFilePath string
	ClusterServersFilePath       string
	RolesDirectoryPath           string
	DataPath                     string
	SystemdServicePath           string
	ClassicServicePath           string
	PackageServicePath           string
	SystemScriptsPath            string
}

type Provider struct {
	config Config
}

func Default() Provider {
	return New(Config{
		ConfigFilePath:               "/etc/otter/.config",
		MachineIDFilePath:            "/etc/otter/machine-id",
		DatabaseFilePath:             "/etc/otter/otter-service.db",
		RolesDirectoryPath:           "/etc/otter/roles",
		DataPath:                     "/data/.otter/otter-packages",
		SystemdServicePath:           "/usr/lib/systemd/system",
		ClassicServicePath:           "/etc/otter/services",
		PackageServicePath:           "/etc/otter/services/.do-not-edit",
		SystemScriptsPath:            "/etc/otter/scripts",
		ClusterServersFilePath:       "/etc/otter/targets",
		ClusterCurrentServerFilePath: "/etc/otter/target",
	})
}

func New(config Config) Provider {
	return Provider{config: config}
}

func (p Provider) Config() Config {
	return p.config
}

func (p Provider) ConfigFilePath() string {
	return p.config.ConfigFilePath
}

func (p Provider) MachineIDFilePath() string {
	return p.config.MachineIDFilePath
}

func (p Provider) DatabaseFilePath() string {
	return p.config.DatabaseFilePath
}

func (p Provider) DatabaseFolderPath() string {
	return filepath.Dir(p.config.DatabaseFilePath)
}

func (p Provider) ClusterCurrentServerFilePath() string {
	return p.config.ClusterCurrentServerFilePath
}

func (p Provider) ClusterServersFilePath() string {
	return p.config.ClusterServersFilePath
}

func (p Provider) RolesDirectoryPath() string {
	return p.config.RolesDirectoryPath
}

func (p Provider) DataPath() string {
	return p.config.DataPath
}

func (p Provider) SystemdServicePath() string {
	return p.config.SystemdServicePath
}

func (p Provider) ClassicServicePath() string {
	return p.config.ClassicServicePath
}

func (p Provider) PackageServicePath() string {
	return p.config.PackageServicePath
}

func (p Provider) SystemScriptsPath() string {
	return p.config.SystemScriptsPath
}

func (p Provider) SystemdServicePathFor(serviceName string) string {
	serviceName = strings.TrimSuffix(serviceName, ".service") + ".service"
	return path.Join(p.config.SystemdServicePath, serviceName)
}

func (p Provider) SystemdServiceDropInPathFor(serviceName string) string {
	serviceDropInName := strings.TrimSuffix(serviceName, ".service") + ".service.d"
	return path.Join(p.config.SystemdServicePath, serviceDropInName)
}

func (p Provider) PackageInstallPath() string {
	return path.Join(p.config.DataPath, "packages")
}

func (p Provider) PackageInstallPathFor(pkgID string) string {
	return path.Join(p.config.DataPath, "packages", pkgID)
}

func (p Provider) PackageServiceBasePath(pkgID string) string {
	return path.Join(p.config.PackageServicePath, pkgID)
}

func (p Provider) PackageServicePathFor(pkgID string, serviceName string) string {
	return path.Join(p.config.PackageServicePath, pkgID, serviceName)
}

func (p Provider) ServicesShortcutPath() string {
	return path.Join(p.config.DataPath, "services")
}

func (p Provider) ServicesInstallPath(pkgID string) string {
	return path.Join(p.config.DataPath, "packages", pkgID, "services")
}

func (p Provider) ServicesInstallPathFor(pkgID string, serviceName string) string {
	return path.Join(p.config.DataPath, "packages", pkgID, "services", serviceName)
}
