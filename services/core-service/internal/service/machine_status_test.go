package service

import (
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lotRunID(id string) *string { return &id }

func campaign(sku string, week time.Time, released int64, scanned int64, runID *string) domain.MachineCampaign {
	return domain.MachineCampaign{
		SKU:                sku,
		WeekStartDate:      week,
		ReleasedBatchCount: released,
		ScannedBatchCount:  scanned,
		ProductionRunID:    runID,
	}
}

func TestCurrentAndNext_PicksTheReleasedCampaignWithWorkLeft(t *testing.T) {
	t.Parallel()

	week := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	campaigns := []domain.MachineCampaign{
		campaign("LKN", week, 6, 2, lotRunID("prru_1")),
		campaign("SKN", week.AddDate(0, 0, 7), 0, 0, nil),
	}

	current, next := currentAndNext(campaigns, week)
	require.NotNil(t, current)
	assert.Equal(t, "LKN", current.SKU)
	require.NotNil(t, next)
	assert.Equal(t, "SKN", next.SKU)
}

// Released work that is fully scanned is finished, so the queue moves on. This is what makes a floor display advance on its own as the shift progresses rather than sitting on a completed job until someone intervenes.
func TestCurrentAndNext_FinishedWorkStopsBeingCurrent(t *testing.T) {
	t.Parallel()

	week := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	campaigns := []domain.MachineCampaign{
		campaign("LKN", week, 6, 6, lotRunID("prru_1")),
		campaign("SKN", week, 7, 1, lotRunID("prru_1")),
	}

	current, next := currentAndNext(campaigns, week)
	require.NotNil(t, current)
	assert.Equal(t, "SKN", current.SKU, "the finished campaign must hand over to the next one")
	assert.Nil(t, next)
}

// An unreleased campaign is not being worked: nothing has been issued to the floor for it. It is what comes next, not what is running.
func TestCurrentAndNext_UnreleasedWorkIsNextNotCurrent(t *testing.T) {
	t.Parallel()

	week := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	campaigns := []domain.MachineCampaign{
		campaign("LKN", week, 0, 0, nil),
	}

	current, next := currentAndNext(campaigns, week)
	assert.Nil(t, current, "a machine with nothing released to it is not running")
	require.NotNil(t, next)
	assert.Equal(t, "LKN", next.SKU)
}

func TestCurrentAndNext_EverythingDoneLeavesNothingRunning(t *testing.T) {
	t.Parallel()

	week := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	campaigns := []domain.MachineCampaign{
		campaign("LKN", week, 6, 6, lotRunID("prru_1")),
	}

	current, next := currentAndNext(campaigns, week)
	assert.Nil(t, current)
	assert.Nil(t, next, "a finished week queues nothing, rather than re-offering completed work")
}

func TestCurrentAndNext_IdleMachine(t *testing.T) {
	t.Parallel()

	current, next := currentAndNext(nil, time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC))
	assert.Nil(t, current)
	assert.Nil(t, next)
}

// A released campaign that issued no batches — a quantity too small to lot — still counts as work in progress rather than being skipped over as if it were done.
func TestCurrentAndNext_ReleasedWithNoBatchesIsStillCurrent(t *testing.T) {
	t.Parallel()

	week := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	campaigns := []domain.MachineCampaign{
		campaign("LKN", week, 0, 0, lotRunID("prru_1")),
	}

	current, _ := currentAndNext(campaigns, week)
	require.NotNil(t, current)
	assert.Equal(t, "LKN", current.SKU)
}
