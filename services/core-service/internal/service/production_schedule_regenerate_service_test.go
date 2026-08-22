package service

import (
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/stretchr/testify/assert"
)

var regenerateHorizonStart = time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)

func manualLine(itemID, machineID string, weekIndex int, quantity float64) *domain.ProductionScheduleLine {
	return &domain.ProductionScheduleLine{
		ItemID:          itemID,
		MachineID:       machineID,
		WeekStartDate:   regenerateHorizonStart.AddDate(0, 0, weekIndex*7),
		PlannedQuantity: quantity,
		SourceCode:      domain.ScheduleLineSourceManual,
	}
}

func solverLine(itemID, machineID string, weekIndex int, quantity float64) *domain.ProductionScheduleLine {
	line := manualLine(itemID, machineID, weekIndex, quantity)
	line.SourceCode = domain.ScheduleLineSourceSolver
	return line
}

func plannedCampaign(itemID, machineID string, weekIndex int, units float64) scheduling.Campaign {
	return scheduling.Campaign{ItemID: itemID, MachineID: machineID, WeekIndex: weekIndex, Units: units}
}

// The number exists to warn a planner what replace_all costs them. A hand edit the fresh plan does not ask for is gone.
func TestCountDiscardedByReplaceAll_CountsEditsTheFreshPlanDropsOrResizes(t *testing.T) {
	t.Parallel()

	existing := []*domain.ProductionScheduleLine{
		manualLine("it_dropped", "mc_1", 3, 1234), // the fresh plan never places it
		manualLine("it_resized", "mc_1", 1, 500),  // the fresh plan wants a different size
		manualLine("it_agreed", "mc_1", 0, 800),   // the fresh plan asks for exactly this
		solverLine("it_solver", "mc_1", 2, 600),   // not a hand edit at all
	}
	replaceAll := []scheduling.Campaign{
		plannedCampaign("it_resized", "mc_1", 1, 900),
		plannedCampaign("it_agreed", "mc_1", 0, 800),
		plannedCampaign("it_solver", "mc_1", 2, 600),
	}

	assert.Equal(t, int32(2), countDiscardedByReplaceAll(existing, replaceAll, regenerateHorizonStart))
}

// A plan that keeps every hand edit at its hand-set size costs nothing to replace.
func TestCountDiscardedByReplaceAll_ZeroWhenThePlanReproducesEveryEdit(t *testing.T) {
	t.Parallel()

	existing := []*domain.ProductionScheduleLine{manualLine("it_1", "mc_1", 0, 800)}
	replaceAll := []scheduling.Campaign{plannedCampaign("it_1", "mc_1", 0, 800)}

	assert.Zero(t, countDiscardedByReplaceAll(existing, replaceAll, regenerateHorizonStart))
}

// An empty plan discards everything hand-made, which is the worst case the warning exists for.
func TestCountDiscardedByReplaceAll_EmptyPlanDiscardsEveryEdit(t *testing.T) {
	t.Parallel()

	existing := []*domain.ProductionScheduleLine{
		manualLine("it_1", "mc_1", 0, 800),
		manualLine("it_2", "mc_2", 1, 900),
	}

	assert.Equal(t, int32(2), countDiscardedByReplaceAll(existing, nil, regenerateHorizonStart))
}

// A solve can place two campaigns for the same item, machine and week; they are one campaign as far as the diff is concerned, so the edit they add up to is not discarded.
func TestCountDiscardedByReplaceAll_SumsCampaignsSharingAKey(t *testing.T) {
	t.Parallel()

	existing := []*domain.ProductionScheduleLine{manualLine("it_1", "mc_1", 0, 800)}
	replaceAll := []scheduling.Campaign{
		plannedCampaign("it_1", "mc_1", 0, 300),
		plannedCampaign("it_1", "mc_1", 0, 500),
	}

	assert.Zero(t, countDiscardedByReplaceAll(existing, replaceAll, regenerateHorizonStart))
}

// The count cannot exceed the manual line count the same preview reports, or the preview would claim a regenerate destroys more hand edits than exist.
func TestSummarizePreview_DiscardedNeverExceedsManual(t *testing.T) {
	t.Parallel()

	existing := []*domain.ProductionScheduleLine{
		manualLine("it_1", "mc_1", 0, 800),
		manualLine("it_1", "mc_1", 0, 900), // same key, so one campaign
	}
	lines := diffCampaigns(existing, nil, map[string]string{}, regenerateHorizonStart, false)

	preview := summarizePreview(lines, countDiscardedByReplaceAll(existing, nil, regenerateHorizonStart))

	assert.Equal(t, int32(1), preview.ManualLineCount)
	assert.LessOrEqual(t, preview.DiscardedManualCount, preview.ManualLineCount)
}

// Previewing the non-destructive mode must still report what the destructive one would cost: the pinned plan keeps every hand edit, so summarizePreview cannot infer the answer from its own diff.
func TestSummarizePreview_ReportsReplaceAllCostWhileHandEditsArePinned(t *testing.T) {
	t.Parallel()

	existing := []*domain.ProductionScheduleLine{manualLine("it_1", "mc_1", 3, 1234)}
	// The preserve_manual plan pins the edit, so its own diff says nothing changed.
	pinnedPlan := []scheduling.Campaign{plannedCampaign("it_1", "mc_1", 3, 1234)}
	lines := diffCampaigns(existing, pinnedPlan, map[string]string{}, regenerateHorizonStart, true)
	for _, line := range lines {
		assert.Equal(t, domain.ScheduleDiffUnchanged, line.ChangeCode)
	}

	// The unpinned plan is what replace_all would apply, and it does not want the campaign.
	preview := summarizePreview(lines, countDiscardedByReplaceAll(existing, nil, regenerateHorizonStart))

	assert.Equal(t, int32(1), preview.ManualLineCount)
	assert.Equal(t, int32(1), preview.DiscardedManualCount, "the hand edit is lost under replace_all even though this preview keeps it")
}
