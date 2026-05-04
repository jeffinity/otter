package servicefile

import "testing"

func TestOptionReadsOtterSection(t *testing.T) {
	value, exists, err := Option("[Unit]\nDescription=API\n\n[X-Otter]\nLogFile=/var/log/api.log\n", "LogFile")
	if err != nil {
		t.Fatalf("option: %v", err)
	}
	if !exists {
		t.Fatalf("expected LogFile to exist")
	}
	if value != "/var/log/api.log" {
		t.Fatalf("LogFile = %q, want /var/log/api.log", value)
	}
}

func TestValuesReadsRepeatedOtterKeys(t *testing.T) {
	values, exists, err := Values("[X-Otter]\nGroup=web, edge\nGroup=worker\n", "Group")
	if err != nil {
		t.Fatalf("values: %v", err)
	}
	if !exists {
		t.Fatalf("expected metadata section to exist")
	}
	if got, want := len(values), 2; got != want {
		t.Fatalf("len(values) = %d, want %d", got, want)
	}
	if values[0] != "web, edge" || values[1] != "worker" {
		t.Fatalf("values = %#v", values)
	}
}

func TestOptionRejectsDuplicateMetadataSections(t *testing.T) {
	_, _, err := Option("[X-Otter]\nLogFile=/tmp/a\n\n[X-Otter]\nLogFile=/tmp/b\n", "LogFile")
	if err == nil || err.Error() != "only one service metadata section is allowed" {
		t.Fatalf("expected duplicate metadata error, got %v", err)
	}
}
