package batch

import (
	"context"
	"fmt"
	"sync"
)

// CampaignAction 批量操作类型
type CampaignAction string

const (
	ActionPause        CampaignAction = "pause"
	ActionResume       CampaignAction = "resume"
	ActionBudgetUpdate CampaignAction = "budget_update"
)

type BatchRequest struct {
	CampaignIDs []int64
	Action      CampaignAction
	NewBudget   *float64 // 仅 budget_update 时使用
}

type BatchResult struct {
	CampaignID int64
	Success    bool
	Error      string
}

// CampaignClient 外部 campaign 操作接口（生产中对接 adortb-dsp）
type CampaignClient interface {
	PauseCampaign(ctx context.Context, campaignID int64) error
	ResumeCampaign(ctx context.Context, campaignID int64) error
	UpdateBudget(ctx context.Context, campaignID int64, budget float64) error
}

type Operator struct {
	client     CampaignClient
	maxWorkers int
}

func NewOperator(client CampaignClient, maxWorkers int) *Operator {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	return &Operator{client: client, maxWorkers: maxWorkers}
}

// Execute 并发批量操作，限制并发数
func (o *Operator) Execute(ctx context.Context, req BatchRequest) []BatchResult {
	results := make([]BatchResult, len(req.CampaignIDs))

	sem := make(chan struct{}, o.maxWorkers)
	var wg sync.WaitGroup

	for i, id := range req.CampaignIDs {
		wg.Add(1)
		go func(idx int, campaignID int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var err error
			switch req.Action {
			case ActionPause:
				err = o.client.PauseCampaign(ctx, campaignID)
			case ActionResume:
				err = o.client.ResumeCampaign(ctx, campaignID)
			case ActionBudgetUpdate:
				if req.NewBudget == nil {
					err = fmt.Errorf("new_budget required for budget_update")
				} else {
					err = o.client.UpdateBudget(ctx, campaignID, *req.NewBudget)
				}
			default:
				err = fmt.Errorf("unknown action: %s", req.Action)
			}

			results[idx] = BatchResult{
				CampaignID: campaignID,
				Success:    err == nil,
			}
			if err != nil {
				results[idx].Error = err.Error()
			}
		}(i, id)
	}
	wg.Wait()
	return results
}

// NoopCampaignClient 空实现，用于测试/开发
type NoopCampaignClient struct{}

func (n *NoopCampaignClient) PauseCampaign(_ context.Context, _ int64) error  { return nil }
func (n *NoopCampaignClient) ResumeCampaign(_ context.Context, _ int64) error { return nil }
func (n *NoopCampaignClient) UpdateBudget(_ context.Context, _ int64, _ float64) error {
	return nil
}
