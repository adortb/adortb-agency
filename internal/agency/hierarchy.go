package agency

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type AgencyAdvertiser struct {
	ID           int64
	AgencyID     int64
	AdvertiserID int64
	Role         string
	AddedAt      time.Time
}

type HierarchyRepo struct {
	db *sql.DB
}

func NewHierarchyRepo(db *sql.DB) *HierarchyRepo {
	return &HierarchyRepo{db: db}
}

func (r *HierarchyRepo) AddAdvertiser(agencyID, advertiserID int64, role string) (*AgencyAdvertiser, error) {
	if role == "" {
		role = "manage"
	}
	var aa AgencyAdvertiser
	err := r.db.QueryRow(`
		INSERT INTO agency_advertisers (agency_id, advertiser_id, role)
		VALUES ($1, $2, $3)
		RETURNING id, agency_id, advertiser_id, role, added_at`,
		agencyID, advertiserID, role,
	).Scan(&aa.ID, &aa.AgencyID, &aa.AdvertiserID, &aa.Role, &aa.AddedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("add advertiser to agency: %w", err)
	}
	return &aa, nil
}

func (r *HierarchyRepo) ListAdvertisers(agencyID int64) ([]AgencyAdvertiser, error) {
	rows, err := r.db.Query(`
		SELECT id, agency_id, advertiser_id, role, added_at
		FROM agency_advertisers WHERE agency_id = $1 ORDER BY added_at DESC`, agencyID)
	if err != nil {
		return nil, fmt.Errorf("list advertisers: %w", err)
	}
	defer rows.Close()

	var result []AgencyAdvertiser
	for rows.Next() {
		var aa AgencyAdvertiser
		if err := rows.Scan(&aa.ID, &aa.AgencyID, &aa.AdvertiserID, &aa.Role, &aa.AddedAt); err != nil {
			return nil, err
		}
		result = append(result, aa)
	}
	return result, rows.Err()
}

func (r *HierarchyRepo) HasAccess(agencyID, advertiserID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(1) FROM agency_advertisers WHERE agency_id=$1 AND advertiser_id=$2`,
		agencyID, advertiserID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *HierarchyRepo) RemoveAdvertiser(agencyID, advertiserID int64) error {
	res, err := r.db.Exec(`
		DELETE FROM agency_advertisers WHERE agency_id=$1 AND advertiser_id=$2`,
		agencyID, advertiserID)
	if err != nil {
		return fmt.Errorf("remove advertiser: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("relation not found")
	}
	return nil
}
