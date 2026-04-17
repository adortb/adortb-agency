package user

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound = errors.New("user not found")
var ErrInvalidCredentials = errors.New("invalid credentials")

type AgencyUser struct {
	ID          int64
	AgencyID    int64
	Email       string
	Name        string
	Role        string
	Status      string
	LastLoginAt *time.Time
	CreatedAt   time.Time
}

type CreateUserReq struct {
	AgencyID int64
	Email    string
	Name     string
	Role     string
	Password string
}

type SubaccountRepo struct {
	db *sql.DB
}

func NewSubaccountRepo(db *sql.DB) *SubaccountRepo {
	return &SubaccountRepo{db: db}
}

func (r *SubaccountRepo) Create(req CreateUserReq) (*AgencyUser, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	var u AgencyUser
	err = r.db.QueryRow(`
		INSERT INTO agency_users (agency_id, email, name, role, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, agency_id, email, name, role, status, last_login_at, created_at`,
		req.AgencyID, req.Email, req.Name, req.Role, string(hash),
	).Scan(&u.ID, &u.AgencyID, &u.Email, &u.Name, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *SubaccountRepo) ListByAgency(agencyID int64) ([]AgencyUser, error) {
	rows, err := r.db.Query(`
		SELECT id, agency_id, email, name, role, status, last_login_at, created_at
		FROM agency_users WHERE agency_id=$1 AND status='active' ORDER BY created_at DESC`,
		agencyID)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var result []AgencyUser
	for rows.Next() {
		var u AgencyUser
		if err := rows.Scan(&u.ID, &u.AgencyID, &u.Email, &u.Name, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (r *SubaccountRepo) Authenticate(agencyID int64, email, password string) (*AgencyUser, error) {
	var u AgencyUser
	var hash string
	err := r.db.QueryRow(`
		SELECT id, agency_id, email, name, role, status, last_login_at, created_at, password_hash
		FROM agency_users WHERE agency_id=$1 AND email=$2 AND status='active'`,
		agencyID, email,
	).Scan(&u.ID, &u.AgencyID, &u.Email, &u.Name, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	// update last_login_at
	_, _ = r.db.Exec(`UPDATE agency_users SET last_login_at=NOW() WHERE id=$1`, u.ID)
	return &u, nil
}

func (r *SubaccountRepo) GetByID(id int64) (*AgencyUser, error) {
	var u AgencyUser
	err := r.db.QueryRow(`
		SELECT id, agency_id, email, name, role, status, last_login_at, created_at
		FROM agency_users WHERE id=$1`, id,
	).Scan(&u.ID, &u.AgencyID, &u.Email, &u.Name, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}
