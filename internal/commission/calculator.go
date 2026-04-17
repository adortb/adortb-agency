package commission

import (
	"database/sql"
	"fmt"
	"time"
)

type CommissionRecord struct {
	ID                   int64
	AgencyID             int64
	PeriodMonth          time.Time
	TotalAdvertiserSpend float64
	CommissionEarned     float64
	Status               string
}

type Calculator struct {
	db *sql.DB
}

func NewCalculator(db *sql.DB) *Calculator {
	return &Calculator{db: db}
}

// EstimateCurrentMonth 预估当月返佣（基于代理商费率）
func (c *Calculator) EstimateCurrentMonth(agencyID int64) (*CommissionRecord, error) {
	var rate float64
	err := c.db.QueryRow(`SELECT commission_rate FROM agencies WHERE id=$1`, agencyID).Scan(&rate)
	if err != nil {
		return nil, fmt.Errorf("get commission rate: %w", err)
	}

	// 当月第一天
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var rec CommissionRecord
	err = c.db.QueryRow(`
		SELECT id, agency_id, period_month, total_advertiser_spend, commission_earned, status
		FROM agency_commissions WHERE agency_id=$1 AND period_month=$2`,
		agencyID, periodStart,
	).Scan(&rec.ID, &rec.AgencyID, &rec.PeriodMonth, &rec.TotalAdvertiserSpend, &rec.CommissionEarned, &rec.Status)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get commission record: %w", err)
	}
	if err == sql.ErrNoRows {
		return &CommissionRecord{
			AgencyID:    agencyID,
			PeriodMonth: periodStart,
			Status:      "pending",
		}, nil
	}
	return &rec, nil
}

// Calculate 计算返佣金额
func Calculate(totalSpend, rate float64) float64 {
	return totalSpend * rate
}

// ListHistory 历史返佣列表
func (c *Calculator) ListHistory(agencyID int64) ([]CommissionRecord, error) {
	rows, err := c.db.Query(`
		SELECT id, agency_id, period_month, total_advertiser_spend, commission_earned, status
		FROM agency_commissions WHERE agency_id=$1 ORDER BY period_month DESC`, agencyID)
	if err != nil {
		return nil, fmt.Errorf("list commissions: %w", err)
	}
	defer rows.Close()

	var result []CommissionRecord
	for rows.Next() {
		var r CommissionRecord
		if err := rows.Scan(&r.ID, &r.AgencyID, &r.PeriodMonth, &r.TotalAdvertiserSpend, &r.CommissionEarned, &r.Status); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// GetByPeriod 按月份查询
func (c *Calculator) GetByPeriod(agencyID int64, period time.Time) (*CommissionRecord, error) {
	var rec CommissionRecord
	err := c.db.QueryRow(`
		SELECT id, agency_id, period_month, total_advertiser_spend, commission_earned, status
		FROM agency_commissions WHERE agency_id=$1 AND period_month=$2`,
		agencyID, period,
	).Scan(&rec.ID, &rec.AgencyID, &rec.PeriodMonth, &rec.TotalAdvertiserSpend, &rec.CommissionEarned, &rec.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("commission record not found")
		}
		return nil, fmt.Errorf("get commission: %w", err)
	}
	return &rec, nil
}
