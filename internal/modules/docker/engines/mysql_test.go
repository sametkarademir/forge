package engines

import (
	"strings"
	"testing"
)

func TestMysqlConnectionInfo(t *testing.T) {
	m := &mysql{}
	ci := m.ConnectionInfo(ConnArgs{Host: "localhost", HostPort: 3306, User: "app", Password: "s3cr3t", Database: "mydb"})
	wantPrimary := "mysql://app:s3cr3t@localhost:3306/mydb"
	if ci.Primary != wantPrimary {
		t.Errorf("Primary = %q, want %q", ci.Primary, wantPrimary)
	}
	if strings.Contains(ci.MaskedPrimary, "s3cr3t") {
		t.Errorf("MaskedPrimary should not contain password, got %q", ci.MaskedPrimary)
	}
	if !strings.Contains(ci.MaskedPrimary, "****") {
		t.Errorf("MaskedPrimary should contain ****, got %q", ci.MaskedPrimary)
	}
	if ci.Endpoints != nil {
		t.Errorf("Endpoints should be nil for mysql, got %v", ci.Endpoints)
	}
}
