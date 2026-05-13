package engines

import (
	"strings"
	"testing"
)

func TestMssqlConnectionInfo(t *testing.T) {
	m := &mssql{}
	ci := m.ConnectionInfo(ConnArgs{Host: "localhost", HostPort: 1433, Password: "P@ssw0rd!", Database: "mydb"})
	wantPrimary := "Server=localhost,1433;Database=mydb;User Id=sa;Password=P@ssw0rd!;TrustServerCertificate=true"
	if ci.Primary != wantPrimary {
		t.Errorf("Primary = %q, want %q", ci.Primary, wantPrimary)
	}
	if strings.Contains(ci.MaskedPrimary, "P@ssw0rd!") {
		t.Errorf("MaskedPrimary should not contain password, got %q", ci.MaskedPrimary)
	}
	if !strings.Contains(ci.MaskedPrimary, "Password=****") {
		t.Errorf("MaskedPrimary should contain Password=****, got %q", ci.MaskedPrimary)
	}
	// Ensure other fields are not mangled
	if !strings.Contains(ci.MaskedPrimary, "Database=mydb") {
		t.Errorf("MaskedPrimary should retain Database field, got %q", ci.MaskedPrimary)
	}
	if ci.Endpoints != nil {
		t.Errorf("Endpoints should be nil for mssql, got %v", ci.Endpoints)
	}
}
