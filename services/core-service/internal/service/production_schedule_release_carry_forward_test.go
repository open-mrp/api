package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func carryCandidate(id string, quantity float64) *domain.CarryForwardBatch {
	return &domain.CarryForwardBatch{BatchID: id, ProductionRunID: "pnrn_prior", ProductionRunNumber: "104", Quantity: quantity}
}

func TestTakeCarryForwardBatches_StopsOnceTheCampaignIsCovered(t *testing.T) {
	candidates := []*domain.CarryForwardBatch{
		carryCandidate("b1", 60), carryCandidate("b2", 60), carryCandidate("b3", 60),
	}

	carried := takeCarryForwardBatches(candidates, 120, map[string]bool{})

	// The third doff is work this week did not ask for. Pulling it in would issue production nobody planned.
	require.Len(t, carried, 2)
	require.Equal(t, "b1", carried[0].BatchID)
	require.Equal(t, "b2", carried[1].BatchID)
}

func TestTakeCarryForwardBatches_LastTicketMayOvershoot(t *testing.T) {
	carried := takeCarryForwardBatches([]*domain.CarryForwardBatch{carryCandidate("b1", 60)}, 50, map[string]bool{})

	// A printed 60-doff covering a 50-unit shortfall is still one ticket. Splitting it would mean reprinting, which is the whole thing being avoided.
	require.Len(t, carried, 1)
	require.Equal(t, float64(60), carried[0].Quantity)
}

func TestTakeCarryForwardBatches_ATicketIsClaimedOnlyOnce(t *testing.T) {
	candidates := []*domain.CarryForwardBatch{carryCandidate("b1", 60), carryCandidate("b2", 60)}
	claimed := map[string]bool{}

	// The same item split across two machines reads the same candidate list twice.
	first := takeCarryForwardBatches(candidates, 60, claimed)
	second := takeCarryForwardBatches(candidates, 60, claimed)

	require.Len(t, first, 1)
	require.Len(t, second, 1)
	require.NotEqual(t, first[0].BatchID, second[0].BatchID)
}

func TestTakeCarryForwardBatches_NothingToCarryLeavesTheCampaignWhole(t *testing.T) {
	require.Empty(t, takeCarryForwardBatches(nil, 360, map[string]bool{}))
}

func TestTakeCarryForwardBatches_ZeroQuantityTicketsAreSkipped(t *testing.T) {
	// A zero-quantity batch would be claimed forever without moving the campaign any closer to covered.
	carried := takeCarryForwardBatches([]*domain.CarryForwardBatch{carryCandidate("b0", 0), carryCandidate("b1", 60)}, 60, map[string]bool{})

	require.Len(t, carried, 1)
	require.Equal(t, "b1", carried[0].BatchID)
}
