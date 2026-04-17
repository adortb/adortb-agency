package reporting

import (
	"database/sql"
	"errors"
	"fmt"
)

type WhiteLabelConfig struct {
	AgencyID     int64
	Domain       string
	LogoURL      string
	PrimaryColor string
	AgencyName   string
}

type WhiteLabelRepo struct {
	db *sql.DB
}

func NewWhiteLabelRepo(db *sql.DB) *WhiteLabelRepo {
	return &WhiteLabelRepo{db: db}
}

func (r *WhiteLabelRepo) GetConfig(agencyID int64) (*WhiteLabelConfig, error) {
	var cfg WhiteLabelConfig
	err := r.db.QueryRow(`
		SELECT id, COALESCE(white_label_domain,''), COALESCE(white_label_logo_url,''),
		       COALESCE(white_label_primary_color,'#1677ff'), name
		FROM agencies WHERE id=$1`, agencyID,
	).Scan(&cfg.AgencyID, &cfg.Domain, &cfg.LogoURL, &cfg.PrimaryColor, &cfg.AgencyName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("agency not found")
		}
		return nil, fmt.Errorf("get white label config: %w", err)
	}
	return &cfg, nil
}

func (r *WhiteLabelRepo) UpdateConfig(agencyID int64, domain, logoURL, primaryColor string) error {
	res, err := r.db.Exec(`
		UPDATE agencies SET white_label_domain=$1, white_label_logo_url=$2, white_label_primary_color=$3
		WHERE id=$4`, domain, logoURL, primaryColor, agencyID)
	if err != nil {
		return fmt.Errorf("update white label: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agency not found")
	}
	return nil
}
