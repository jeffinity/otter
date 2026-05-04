package command

import (
	"testing"
)

func TestConfigUsesOtterNaming(t *testing.T) {
	for name, tc := range map[string]struct {
		got  string
		want string
	}{
		"OtterService":               {got: OtterService, want: "otter service"},
		"OtterCore":                  {got: OtterCore, want: "otter-core"},
		"OtterRun":                   {got: OtterRun, want: "otter run"},
		"OtterCoreServiceName":       {got: OtterCoreServiceName, want: "otter-core"},
		"OtterCoreUnitName":          {got: OtterCoreUnitName, want: "otter-core.service"},
		"OtterAuditBypassEnv":        {got: OtterAuditBypassEnv, want: "OTTER_AUDIT_BYPASS"},
		"ListenLocalFile":            {got: ListenLocalFile, want: "/var/run/otter-core.socket"},
		"ListenTCPAddr":              {got: ListenTCPAddr, want: "0.0.0.0:3456"},
		"DialTCPAddr":                {got: DialTCPAddr, want: "127.0.0.1:3456"},
		"DialLocalFile":              {got: DialLocalFile, want: "unix:///var/run/otter-core.socket"},
		"OtterEnvFilePath":           {got: OtterEnvFilePath, want: "/etc/otter/systemd.env"},
		"SystemdDropInFilePath":      {got: SystemdDropInFilePath, want: "/etc/otter/systemd.conf"},
		"AuditDropInFilePath":        {got: AuditDropInFilePath, want: "/etc/otter/audit.conf"},
		"RestartAlwaysDropInPath":    {got: RestartAlwaysDropInPath, want: "/etc/otter/restart-always.conf"},
		"RsyslogConfPath":            {got: RsyslogConfPath, want: "/etc/otter/rsyslog.conf"},
		"OtterCoreAuditFilePath":     {got: OtterCoreAuditFilePath, want: "/etc/otter/otter-core-audit.log"},
		"SystemServiceBaseDirectory": {got: SystemServiceBaseDirectory, want: "/usr/lib/systemd/system/"},
	} {
		if tc.got != tc.want {
			t.Fatalf("%s = %q, want %q", name, tc.got, tc.want)
		}
	}
}
