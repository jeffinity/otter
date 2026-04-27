package serviceconfig

import (
	"strings"
	"testing"
)

func TestConfigUsesOtterNaming(t *testing.T) {
	values := []string{
		OtterService,
		OtterCore,
		OtterRun,
		OtterCoreServiceName,
		OtterCoreUnitName,
		OtterAuditBypassEnv,
		ListenLocalFile,
		ListenTCPAddr,
		DialTCPAddr,
		DialLocalFile,
		OtterEnvFilePath,
		SystemdDropInFilePath,
		AuditDropInFilePath,
		RestartAlwaysDropInPath,
		RsyslogConfPath,
		OtterCoreAuditFilePath,
	}

	for _, value := range values {
		if strings.Contains(value, "ambot") || strings.Contains(value, "AMBOT") {
			t.Fatalf("value %q should not contain legacy name", value)
		}
	}
}
