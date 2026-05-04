package command

const (
	OtterService = "otter service"
	OtterCore    = "otter-core"
	OtterRun     = "otter run"

	OtterCoreServiceName = "otter-core"
	OtterCoreUnitName    = OtterCoreServiceName + ".service"

	OtterAuditBypassEnv = "OTTER_AUDIT_BYPASS"

	ListenLocalFile = "/var/run/otter-core.socket"
	ListenTCPPort   = "3456"
	ListenTCPAddr   = "0.0.0.0:" + ListenTCPPort
	DialTCPAddr     = "127.0.0.1:" + ListenTCPPort
	DialLocalFile   = "unix://" + ListenLocalFile

	OtterEnvFilePath           = "/etc/otter/systemd.env"
	SystemdDropInFilePath      = "/etc/otter/systemd.conf"
	AuditDropInFilePath        = "/etc/otter/audit.conf"
	RestartAlwaysDropInPath    = "/etc/otter/restart-always.conf"
	RsyslogConfPath            = "/etc/otter/rsyslog.conf"
	OtterCoreAuditFilePath     = "/etc/otter/otter-core-audit.log"
	SystemServiceBaseDirectory = "/usr/lib/systemd/system/"
)
