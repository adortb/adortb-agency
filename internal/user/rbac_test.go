package user

import (
	"testing"
)

func TestHasPermission(t *testing.T) {
	tests := []struct {
		role       string
		permission string
		want       bool
	}{
		{RoleAgencyAdmin, PermViewCampaigns, true},
		{RoleAgencyAdmin, PermEditCampaigns, true},
		{RoleAgencyAdmin, PermManageUsers, true},
		{RoleAgencyAdmin, PermManageBilling, true},
		{RoleMediaBuyer, PermViewCampaigns, true},
		{RoleMediaBuyer, PermEditCampaigns, true},
		{RoleMediaBuyer, PermManageUsers, false},
		{RoleMediaBuyer, PermManageBilling, false},
		{RoleAnalyst, PermViewReports, true},
		{RoleAnalyst, PermEditCampaigns, false},
		{RoleAnalyst, PermManageUsers, false},
		{"unknown_role", PermViewReports, false},
	}

	for _, tt := range tests {
		t.Run(tt.role+":"+tt.permission, func(t *testing.T) {
			got := HasPermission(tt.role, tt.permission)
			if got != tt.want {
				t.Errorf("HasPermission(%q, %q) = %v, want %v", tt.role, tt.permission, got, tt.want)
			}
		})
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleAgencyAdmin == "" || RoleMediaBuyer == "" || RoleAnalyst == "" {
		t.Error("role constants must not be empty")
	}
}
