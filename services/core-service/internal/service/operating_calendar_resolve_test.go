package service

import (
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/scheduling"
)

func TestCommitmentAnchor_ProjectsEveryBasisNotJustTheIssueDate(t *testing.T) {
	t.Parallel()

	issued := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	promised := time.Date(2026, time.December, 24, 0, 0, 0, 0, time.UTC)
	pinned := time.Date(2026, time.November, 20, 0, 0, 0, 0, time.UTC)
	override := 45

	cases := []struct {
		name  string
		basis scheduling.CommitmentBasis
		in    scheduling.LeadTimeInput
		want  time.Time
	}{
		{"pinned ship date", scheduling.CommitmentBasis{ShipByOverrideDate: &pinned}, scheduling.LeadTimeInput{}, pinned},
		{"promised delivery", scheduling.CommitmentBasis{PromisedAt: &promised}, scheduling.LeadTimeInput{}, promised},
		// The case a window centred on the issue date would miss: forty-five days out is already past the window's forward edge, so its closures would silently not exist.
		{"lead-time override", scheduling.CommitmentBasis{LeadTimeOverrideDays: &override}, scheduling.LeadTimeInput{}, issued.AddDate(0, 0, 45)},
		{"resolved chain", scheduling.CommitmentBasis{}, scheduling.LeadTimeInput{AccountLeadTimeDays: intPtr(60)}, issued.AddDate(0, 0, 60)},
		{"nothing resolvable", scheduling.CommitmentBasis{}, scheduling.LeadTimeInput{}, issued},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := commitmentAnchor(c.basis, c.in, issued); !got.Equal(c.want) {
				t.Fatalf("got %s, want %s", got.Format(time.DateOnly), c.want.Format(time.DateOnly))
			}
		})
	}
}

// Whatever the anchor, the window it opens has to contain the date the commitment actually lands on, or the walk runs through closures that were never loaded.
func TestCommitmentAnchor_WindowCoversTheResolvedDate(t *testing.T) {
	t.Parallel()

	issued := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)

	for _, days := range []int{0, 1, 30, 45, 90, 365, 3650} {
		override := days
		anchor := commitmentAnchor(scheduling.CommitmentBasis{LeadTimeOverrideDays: &override}, scheduling.LeadTimeInput{}, issued)
		from, to := anchor.Add(-closureWindowBefore), anchor.Add(closureWindowAfter)

		landed := issued.AddDate(0, 0, days)
		if landed.Before(from) || landed.After(to) {
			t.Fatalf("a %d-day lead time lands on %s, outside the window %s..%s", days, landed.Format(time.DateOnly), from.Format(time.DateOnly), to.Format(time.DateOnly))
		}
	}
}

func intPtr(v int) *int { return &v }
