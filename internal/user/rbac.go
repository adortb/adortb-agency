package user

import (
	"database/sql"
	"fmt"
	"time"
)

// Role constants
const (
	RoleAgencyAdmin = "agency_admin"
	RoleMediaBuyer  = "media_buyer"
	RoleAnalyst     = "analyst"
)

// Permission constants
const (
	PermViewCampaigns  = "view_campaigns"
	PermEditCampaigns  = "edit_campaigns"
	PermViewReports    = "view_reports"
	PermManageUsers    = "manage_users"
	PermManageBilling  = "manage_billing"
)

// rolePermissions defines what each role can do by default.
var rolePermissions = map[string][]string{
	RoleAgencyAdmin: {PermViewCampaigns, PermEditCampaigns, PermViewReports, PermManageUsers, PermManageBilling},
	RoleMediaBuyer:  {PermViewCampaigns, PermEditCampaigns, PermViewReports},
	RoleAnalyst:     {PermViewReports},
}

// HasPermission checks if a role has a given permission.
func HasPermission(role, permission string) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

type UserPermission struct {
	ID            int64
	AgencyUserID  int64
	AdvertiserID  *int64
	Permission    string
	GrantedAt     time.Time
}

type PermissionRepo struct {
	db *sql.DB
}

func NewPermissionRepo(db *sql.DB) *PermissionRepo {
	return &PermissionRepo{db: db}
}

func (r *PermissionRepo) Grant(agencyUserID int64, advertiserID *int64, permission string) error {
	_, err := r.db.Exec(`
		INSERT INTO user_permissions (agency_user_id, advertiser_id, permission)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`,
		agencyUserID, advertiserID, permission)
	if err != nil {
		return fmt.Errorf("grant permission: %w", err)
	}
	return nil
}

func (r *PermissionRepo) ListByUser(agencyUserID int64) ([]UserPermission, error) {
	rows, err := r.db.Query(`
		SELECT id, agency_user_id, advertiser_id, permission, granted_at
		FROM user_permissions WHERE agency_user_id=$1`, agencyUserID)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()

	var result []UserPermission
	for rows.Next() {
		var p UserPermission
		if err := rows.Scan(&p.ID, &p.AgencyUserID, &p.AdvertiserID, &p.Permission, &p.GrantedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// CanAccessAdvertiser checks fine-grained access: either global (NULL advertiser_id) or specific.
func (r *PermissionRepo) CanAccessAdvertiser(agencyUserID, advertiserID int64, permission string) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(1) FROM user_permissions
		WHERE agency_user_id=$1 AND permission=$2
		  AND (advertiser_id IS NULL OR advertiser_id=$3)`,
		agencyUserID, permission, advertiserID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
