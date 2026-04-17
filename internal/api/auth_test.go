package api

import (
	"testing"
)

func TestGenerateAndParseToken(t *testing.T) {
	claims := Claims{
		AgencyUserID: 42,
		AgencyID:     7,
		Role:         "agency_admin",
	}
	token, err := generateToken(claims)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	parsed, err := parseToken(token)
	if err != nil {
		t.Fatalf("parseToken: %v", err)
	}
	if parsed.AgencyUserID != claims.AgencyUserID {
		t.Errorf("AgencyUserID: got %d, want %d", parsed.AgencyUserID, claims.AgencyUserID)
	}
	if parsed.AgencyID != claims.AgencyID {
		t.Errorf("AgencyID: got %d, want %d", parsed.AgencyID, claims.AgencyID)
	}
	if parsed.Role != claims.Role {
		t.Errorf("Role: got %q, want %q", parsed.Role, claims.Role)
	}
}

func TestParseToken_InvalidSignature(t *testing.T) {
	token, _ := generateToken(Claims{AgencyUserID: 1, AgencyID: 1, Role: "analyst"})
	// tamper: append character to signature part
	tampered := token[:len(token)-1] + "X"
	_, err := parseToken(tampered)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestParseToken_BadFormat(t *testing.T) {
	_, err := parseToken("notavalidtoken")
	if err == nil {
		t.Error("expected error for bad format token")
	}
}

func TestPathSegments(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		want   []string
	}{
		{"/v1/agencies/123/users", "/v1/agencies/", []string{"123", "users"}},
		{"/v1/agencies/5", "/v1/agencies/", []string{"5"}},
		{"/v1/agencies/", "/v1/agencies/", nil},
	}
	for _, tt := range tests {
		got := pathSegments(tt.path, tt.prefix)
		if len(got) != len(tt.want) {
			t.Errorf("pathSegments(%q, %q) = %v, want %v", tt.path, tt.prefix, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("pathSegments[%d]: got %q, want %q", i, got[i], tt.want[i])
			}
		}
	}
}
