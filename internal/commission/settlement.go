package commission

import (
	"database/sql"
	"fmt"
	"time"
)

type SettlementService struct {
	db   *sql.DB
	calc *Calculator
}

func NewSettlementService(db *sql.DB, calc *Calculator) *SettlementService {
	return &SettlementService{db: db, calc: calc}
}

// Settle 触发月度结算：创建或更新当月返佣记录
func (s *SettlementService) Settle(agencyID int64, totalSpend float64) (*CommissionRecord, error) {
	var rate float64
	err := s.db.QueryRow(`SELECT commission_rate FROM agencies WHERE id=$1`, agencyID).Scan(&rate)
	if err != nil {
		return nil, fmt.Errorf("get rate: %w", err)
	}

	earned := Calculate(totalSpend, rate)
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var rec CommissionRecord
	err = s.db.QueryRow(`
		INSERT INTO agency_commissions (agency_id, period_month, total_advertiser_spend, commission_earned, status)
		VALUES ($1, $2, $3, $4, 'invoiced')
		ON CONFLICT (agency_id, period_month) DO UPDATE
		  SET total_advertiser_spend = EXCLUDED.total_advertiser_spend,
		      commission_earned = EXCLUDED.commission_earned,
		      status = 'invoiced'
		RETURNING id, agency_id, period_month, total_advertiser_spend, commission_earned, status`,
		agencyID, periodStart, totalSpend, earned,
	).Scan(&rec.ID, &rec.AgencyID, &rec.PeriodMonth, &rec.TotalAdvertiserSpend, &rec.CommissionEarned, &rec.Status)
	if err != nil {
		return nil, fmt.Errorf("settle commission: %w", err)
	}
	return &rec, nil
}
