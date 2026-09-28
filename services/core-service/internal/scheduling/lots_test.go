package scheduling

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSplitIntoLots_WholeLots(t *testing.T) {
	t.Parallel()

	// The case from the plan grid: 360 pairs at a 60-pair doff is six doffs, not five
	// and a bit.
	assert.Equal(t, []float64{60, 60, 60, 60, 60, 60}, SplitIntoLots(360, 60))
	assert.Equal(t, []float64{60, 60, 60, 60, 60, 60, 60}, SplitIntoLots(420, 60))
}

func TestSplitIntoLots_RemainderTrailsAsShortLot(t *testing.T) {
	t.Parallel()

	lots := SplitIntoLots(370, 60)

	// The short doff goes last because it is the one that gets cut when the week runs
	// late; buried in the middle it would strand the full lots behind it.
	assert.Equal(t, []float64{60, 60, 60, 60, 60, 60, 10}, lots)

	var total float64
	for _, lot := range lots {
		total += lot
	}
	assert.InDelta(t, 370, total, 1e-9, "splitting must conserve the planned quantity")
}

func TestSplitIntoLots_ConservesQuantity(t *testing.T) {
	t.Parallel()

	for _, quantity := range []float64{1, 59, 60, 61, 119, 360, 1234.5} {
		var total float64
		for _, lot := range SplitIntoLots(quantity, 60) {
			total += lot
		}
		assert.InDelta(t, quantity, total, 1e-9, "quantity %v must be conserved", quantity)
	}
}

// A campaign that is conceptually six whole doffs can arrive as 359.9999999 after decimal
// round-tripping. Emitting a seventh batch of a millionth of a unit would put a batch on
// the floor that nobody can run.
func TestSplitIntoLots_AbsorbsFloatingPointDust(t *testing.T) {
	t.Parallel()

	assert.Len(t, SplitIntoLots(359.9999999999, 60), 6)
	assert.Len(t, SplitIntoLots(360.0000000001, 60), 6)
}

func TestSplitIntoLots_SmallerThanOneLot(t *testing.T) {
	t.Parallel()

	// Half a doff is still a doff's worth of setup, so it is one batch rather than none.
	assert.Equal(t, []float64{30}, SplitIntoLots(30, 60))
	assert.Equal(t, []float64{60}, SplitIntoLots(60, 60))
}

func TestSplitIntoLots_UnlottedItem(t *testing.T) {
	t.Parallel()

	// No lot size configured means the item is not run in lots, so the campaign is one
	// batch rather than an error or an empty run.
	assert.Equal(t, []float64{500}, SplitIntoLots(500, 0))
	assert.Equal(t, []float64{500}, SplitIntoLots(500, -1))
}

func TestSplitIntoLots_NothingPlanned(t *testing.T) {
	t.Parallel()

	assert.Nil(t, SplitIntoLots(0, 60))
	assert.Nil(t, SplitIntoLots(-5, 60))
}

func TestCountLots_MatchesSplit(t *testing.T) {
	t.Parallel()

	for _, quantity := range []float64{0, 30, 60, 370, 3600} {
		assert.Equal(t, len(SplitIntoLots(quantity, 60)), CountLots(quantity, 60))
	}
}

func TestSplitIntoLots_FractionalRemainderIsExact(t *testing.T) {
	t.Parallel()

	// 370.3 − 6 × 60 in float64 is 10.300000000000011; the short lot is a batch quantity and has
	// to read 10.3 when it reaches the ledger.
	lots := SplitIntoLots(370.3, 60)

	assert.Len(t, lots, 7)
	assert.Equal(t, "10.3", decimal.NewFromFloat(lots[6]).String())
}
