package batch

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type mockCampaignClient struct {
	pauseCalled  atomic.Int64
	resumeCalled atomic.Int64
	budgetCalled atomic.Int64
	failIDs      map[int64]bool
}

func (m *mockCampaignClient) PauseCampaign(_ context.Context, id int64) error {
	m.pauseCalled.Add(1)
	if m.failIDs[id] {
		return errors.New("pause failed")
	}
	return nil
}

func (m *mockCampaignClient) ResumeCampaign(_ context.Context, id int64) error {
	m.resumeCalled.Add(1)
	if m.failIDs[id] {
		return errors.New("resume failed")
	}
	return nil
}

func (m *mockCampaignClient) UpdateBudget(_ context.Context, id int64, _ float64) error {
	m.budgetCalled.Add(1)
	if m.failIDs[id] {
		return errors.New("budget update failed")
	}
	return nil
}

func TestBatchPause_AllSuccess(t *testing.T) {
	client := &mockCampaignClient{}
	op := NewOperator(client, 5)
	ids := []int64{1, 2, 3, 4, 5}
	results := op.Execute(context.Background(), BatchRequest{
		CampaignIDs: ids,
		Action:      ActionPause,
	})
	if len(results) != len(ids) {
		t.Fatalf("expected %d results, got %d", len(ids), len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("campaign %d: expected success", r.CampaignID)
		}
	}
	if got := client.pauseCalled.Load(); got != int64(len(ids)) {
		t.Errorf("pauseCalled = %d, want %d", got, len(ids))
	}
}

func TestBatchPause_PartialFailure(t *testing.T) {
	client := &mockCampaignClient{failIDs: map[int64]bool{2: true, 4: true}}
	op := NewOperator(client, 5)
	results := op.Execute(context.Background(), BatchRequest{
		CampaignIDs: []int64{1, 2, 3, 4, 5},
		Action:      ActionPause,
	})
	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}
	if successCount != 3 {
		t.Errorf("expected 3 successes, got %d", successCount)
	}
	if failCount != 2 {
		t.Errorf("expected 2 failures, got %d", failCount)
	}
}

func TestBatchBudgetUpdate(t *testing.T) {
	client := &mockCampaignClient{}
	op := NewOperator(client, 3)
	budget := 500.0
	results := op.Execute(context.Background(), BatchRequest{
		CampaignIDs: []int64{10, 20, 30},
		Action:      ActionBudgetUpdate,
		NewBudget:   &budget,
	})
	for _, r := range results {
		if !r.Success {
			t.Errorf("campaign %d: %s", r.CampaignID, r.Error)
		}
	}
}

func TestBatchBudgetUpdate_MissingBudget(t *testing.T) {
	client := &mockCampaignClient{}
	op := NewOperator(client, 3)
	results := op.Execute(context.Background(), BatchRequest{
		CampaignIDs: []int64{1},
		Action:      ActionBudgetUpdate,
		NewBudget:   nil,
	})
	if results[0].Success {
		t.Error("expected failure when NewBudget is nil")
	}
}

func TestBatchUnknownAction(t *testing.T) {
	client := &mockCampaignClient{}
	op := NewOperator(client, 2)
	results := op.Execute(context.Background(), BatchRequest{
		CampaignIDs: []int64{1},
		Action:      CampaignAction("unknown"),
	})
	if results[0].Success {
		t.Error("expected failure for unknown action")
	}
}

func TestNewOperator_DefaultWorkers(t *testing.T) {
	op := NewOperator(&NoopCampaignClient{}, 0)
	if op.maxWorkers != 10 {
		t.Errorf("expected default maxWorkers=10, got %d", op.maxWorkers)
	}
}
