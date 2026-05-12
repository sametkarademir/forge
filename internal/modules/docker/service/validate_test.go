package service

import "testing"

func TestValidateProjectName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "myproject", false},
		{"valid with hyphens", "my-project-1", false},
		{"valid single char", "a", false},
		{"empty", "", true},
		{"uppercase", "MyProject", true},
		{"leading hyphen", "-project", true},
		{"leading digit then hyphen", "1-project", false},
		{"too long (64 chars)", "a123456789012345678901234567890123456789012345678901234567890123", true},
		{"max length (63 chars)", "a12345678901234567890123456789012345678901234567890123456789012", false},
		{"contains underscore", "my_project", true},
		{"contains space", "my project", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProjectName(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for input %q, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error for input %q, got: %v", tc.input, err)
			}
		})
	}
}
