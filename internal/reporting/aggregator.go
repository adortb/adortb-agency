package reporting

import (
	"database/sql"
	"fmt"
	"time"
)

// AggregatedReport 跨客户聚合报表行
type AggregatedReport struct {
	AdvertiserID   int64
	AdvertiserName string
	Date           time.Time
	Impressions    int64
	Clicks         int64
	Spend          float64
	CTR            float64
}

// AgencySummary 代理商整体汇总
type AgencySummary struct {
	TotalAdvertisers int
	TotalImpressions int64
	TotalClicks      int64
	TotalSpend       float64
	Period           string
}

type Aggregator struct {
	db *sql.DB
}

func NewAggregator(db *sql.DB) *Aggregator {
	return &Aggregator{db: db}
}

// GetAggregatedReport 拉取代理商下所有广告主的聚合报表
// 依赖 agency_advertisers 做权限过滤，实际消耗数据从外部 reporting 服务获取
// 这里基于 agency_advertisers 构建模拟聚合（生产环境需调用 adortb-reporting 服务）
func (a *Aggregator) GetAggregatedReport(agencyID int64, from, to time.Time) ([]AggregatedReport, error) {
	rows, err := a.db.Query(`
		SELECT aa.advertiser_id
		FROM agency_advertisers aa
		WHERE aa.agency_id = $1`, agencyID)
	if err != nil {
		return nil, fmt.Errorf("fetch advertisers: %w", err)
	}
	defer rows.Close()

	var advIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		advIDs = append(advIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 返回空聚合结构（生产中需调用内部 reporting gRPC/HTTP）
	result := make([]AggregatedReport, 0, len(advIDs))
	for _, id := range advIDs {
		result = append(result, AggregatedReport{
			AdvertiserID: id,
			Date:         from,
		})
	}
	return result, nil
}

// GetAgencySummary 获取代理商下所有客户的汇总统计
func (a *Aggregator) GetAgencySummary(agencyID int64) (*AgencySummary, error) {
	var count int
	err := a.db.QueryRow(`SELECT COUNT(1) FROM agency_advertisers WHERE agency_id=$1`, agencyID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("count advertisers: %w", err)
	}
	return &AgencySummary{
		TotalAdvertisers: count,
		Period:           time.Now().Format("2006-01"),
	}, nil
}
