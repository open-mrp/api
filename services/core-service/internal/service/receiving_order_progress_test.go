package service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/pricing"
)

var (
	ratioEach = pricing.UnitRatio{Numerator: decimal.NewFromInt(1), Denominator: decimal.NewFromInt(1)}
	ratioPair = pricing.UnitRatio{Numerator: decimal.NewFromInt(2), Denominator: decimal.NewFromInt(1)}
	t0        = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
)

// progressLine is a receiving line on a purchase order line ordered as 10 each.
func progressLine(id string, value string, unitID string, stocked bool, minute int) domain.ReceivingProgressLine {
	ratio := ratioEach
	if unitID == "pair" {
		ratio = ratioPair
	}
	l := domain.ReceivingProgressLine{
		ID:               id,
		OrderLineID:      "sol_1",
		CreatedAt:        t0.Add(time.Duration(minute) * time.Minute),
		Value:            decimal.RequireFromString(value),
		UnitID:           unitID,
		UnitRatio:        ratio,
		OrderedValue:     decimal.NewFromInt(10),
		OrderedUnitID:    "each",
		OrderedUnitRatio: ratioEach,
	}
	if stocked {
		at := l.CreatedAt
		l.StockedAt = &at
	}
	return l
}

func TestReceiveTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		lines  []domain.ReceivingProgressLine
		want   string
		wantOK bool
	}{
		{
			name:   "an untouched line takes the whole order",
			lines:  []domain.ReceivingProgressLine{progressLine("rol_a", "0", "each", false, 0)},
			want:   "10",
			wantOK: true,
		},
		{
			// Express set this line to 5 (ordered less every line, its own included); finishing a line means it holds the rest.
			name:   "a partly counted line is finished, not halved",
			lines:  []domain.ReceivingProgressLine{progressLine("rol_a", "5", "each", false, 0)},
			want:   "10",
			wantOK: true,
		},
		{
			name: "what earlier deliveries stocked is not received twice",
			lines: []domain.ReceivingProgressLine{
				progressLine("rol_old", "4", "each", true, 0),
				progressLine("rol_a", "0", "each", false, 1),
			},
			want:   "6",
			wantOK: true,
		},
		{
			name: "other lines counted in pairs are converted before they are subtracted",
			lines: []domain.ReceivingProgressLine{
				progressLine("rol_old", "2", "pair", true, 0),
				progressLine("rol_a", "0", "each", false, 1),
			},
			want:   "6",
			wantOK: true,
		},
		{
			name:  "an over-received line is left alone",
			lines: []domain.ReceivingProgressLine{progressLine("rol_a", "12", "each", false, 0)},
		},
		{
			name: "nothing to do once the order is covered",
			lines: []domain.ReceivingProgressLine{
				progressLine("rol_old", "10", "each", true, 0),
				progressLine("rol_a", "0", "each", false, 1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := receiveTarget(progressByOrderLine(tt.lines)["sol_1"], "rol_a")
			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.True(t, got.Equal(decimal.RequireFromString(tt.want)), "got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFollowUpLines(t *testing.T) {
	t.Parallel()

	t.Run("a short stocking opens a line at zero in the ordered unit", func(t *testing.T) {
		t.Parallel()
		groups := progressByOrderLine([]domain.ReceivingProgressLine{progressLine("rol_a", "3", "pair", true, 0)})
		require.Equal(t, []followUpLine{{OrderLineID: "sol_1", UnitID: "each"}}, followUpLines(groups))
	})

	t.Run("a fully stocked order line needs nothing", func(t *testing.T) {
		t.Parallel()
		groups := progressByOrderLine([]domain.ReceivingProgressLine{progressLine("rol_a", "5", "pair", true, 0)})
		require.Empty(t, followUpLines(groups))
	})

	t.Run("an order line that still has an unstocked line needs nothing", func(t *testing.T) {
		t.Parallel()
		groups := progressByOrderLine([]domain.ReceivingProgressLine{
			progressLine("rol_a", "3", "each", true, 0),
			progressLine("rol_b", "0", "each", false, 1),
		})
		require.Empty(t, followUpLines(groups))
	})
}
