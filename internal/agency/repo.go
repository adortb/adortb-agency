package agency

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("agency not found")
var ErrDuplicate = errors.New("agency already exists")

type Agency struct {
	ID                  int64
	Name                string
	LegalEntity         string
	ContactEmail        string
	WhiteLabelDomain    string
	WhiteLabelLogoURL   string
	WhiteLabelPrimColor string
	CommissionRate      float64
	Status              string
	CreatedAt           time.Time
}

type CreateAgencyReq struct {
	Name             string
	LegalEntity      string
	ContactEmail     string
	CommissionRate   float64
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(req CreateAgencyReq) (*Agency, error) {
	rate := req.CommissionRate
	if rate <= 0 {
		rate = 0.10
	}
	var a Agency
	err := r.db.QueryRow(`
		INSERT INTO agencies (name, legal_entity, contact_email, commission_rate)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, legal_entity, contact_email,
		          COALESCE(white_label_domain,''), COALESCE(white_label_logo_url,''),
		          COALESCE(white_label_primary_color,'#1677ff'),
		          commission_rate, status, created_at`,
		req.Name, req.LegalEntity, req.ContactEmail, rate,
	).Scan(&a.ID, &a.Name, &a.LegalEntity, &a.ContactEmail,
		&a.WhiteLabelDomain, &a.WhiteLabelLogoURL, &a.WhiteLabelPrimColor,
		&a.CommissionRate, &a.Status, &a.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("create agency: %w", err)
	}
	return &a, nil
}

func (r *Repo) GetByID(id int64) (*Agency, error) {
	var a Agency
	err := r.db.QueryRow(`
		SELECT id, name, legal_entity, contact_email,
		       COALESCE(white_label_domain,''), COALESCE(white_label_logo_url,''),
		       COALESCE(white_label_primary_color,'#1677ff'),
		       commission_rate, status, created_at
		FROM agencies WHERE id = $1`, id,
	).Scan(&a.ID, &a.Name, &a.LegalEntity, &a.ContactEmail,
		&a.WhiteLabelDomain, &a.WhiteLabelLogoURL, &a.WhiteLabelPrimColor,
		&a.CommissionRate, &a.Status, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get agency: %w", err)
	}
	return &a, nil
}

func (r *Repo) UpdateWhiteLabel(id int64, domain, logoURL, primColor string) error {
	res, err := r.db.Exec(`
		UPDATE agencies SET white_label_domain=$1, white_label_logo_url=$2, white_label_primary_color=$3
		WHERE id=$4`, domain, logoURL, primColor, id)
	if err != nil {
		return fmt.Errorf("update white label: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "unique") || contains(err.Error(), "duplicate"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexString(s, sub) >= 0)
}

func indexString(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
