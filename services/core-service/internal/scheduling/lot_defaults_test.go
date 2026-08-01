package scheduling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sockLine   = "pdln_socks"
	sleeveLine = "pdln_sleeves"
	pairs      = "un_pairs"
	eaches     = "un_eaches"
)

func baseLotInput() LotResolutionInput {
	return LotResolutionInput{
		ItemOverrides:     map[string]float64{},
		ProductLineByItem: map[string]string{},
		LotByProductLine: map[string]ProductLineLot{
			sockLine:   {ProductLineID: sockLine, Quantity: 60, UnitID: pairs},
			sleeveLine: {ProductLineID: sleeveLine, Quantity: 60, UnitID: eaches},
		},
		DownstreamByItem: map[string][]FinishedGood{},
		AccountLotUnits:  60,
		UnitByItem:       map[string]string{},
	}
}

// The distinction the whole feature exists to draw: the same 60 means pairs for socks and
// eaches for armsleeves, and the unit is what separates them.
func TestResolveLotDefault_UnitComesFromTheProductLine(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.ProductLineByItem["it_sock"] = sockLine
	in.ProductLineByItem["it_sleeve"] = sleeveLine

	sock, ok := ResolveLotDefault("it_sock", in)
	require.True(t, ok)
	assert.Equal(t, float64(60), sock.Quantity)
	assert.Equal(t, pairs, sock.UnitID)
	assert.Equal(t, LotSourceProductLine, sock.Source)

	sleeve, ok := ResolveLotDefault("it_sleeve", in)
	require.True(t, ok)
	assert.Equal(t, float64(60), sleeve.Quantity)
	assert.Equal(t, eaches, sleeve.UnitID)
}

// Greige is not sold, so it has no product line of its own. It takes its lot from what it
// becomes, which is the case the plan actually runs on.
func TestResolveLotDefault_GreigeInheritsFromWhatItBecomes(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.UnitByItem["it_greige"] = eaches
	in.DownstreamByItem["it_greige"] = []FinishedGood{
		{ItemID: "it_sock_a", ProductLineID: sockLine, Monthly: []MonthlyDemand{{Quantity: 500}}},
	}

	lot, ok := ResolveLotDefault("it_greige", in)
	require.True(t, ok)
	assert.Equal(t, pairs, lot.UnitID,
		"sock greige must be counted in pairs, not in the item's own unit")
	assert.Equal(t, LotSourceDownstreamProductLine, lot.Source)
	assert.Equal(t, sockLine, lot.ProductLineID)
}

func TestResolveLotDefault_CompetingLinesAreDecidedByDemand(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.DownstreamByItem["it_greige"] = []FinishedGood{
		{ItemID: "it_sleeve_a", ProductLineID: sleeveLine, Monthly: []MonthlyDemand{{Quantity: 100}}},
		{ItemID: "it_sock_a", ProductLineID: sockLine, Monthly: []MonthlyDemand{{Quantity: 900}}},
	}

	lot, ok := ResolveLotDefault("it_greige", in)
	require.True(t, ok)
	assert.Equal(t, pairs, lot.UnitID,
		"a greige that mostly becomes socks is knitted in the sock line's doff")
}

// The levelling is deterministic and the lot must not be the thing that makes it wobble.
func TestResolveLotDefault_TiesResolveTheSameWayEveryTime(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.DownstreamByItem["it_greige"] = []FinishedGood{
		{ItemID: "it_sleeve_a", ProductLineID: sleeveLine, Monthly: []MonthlyDemand{{Quantity: 500}}},
		{ItemID: "it_sock_a", ProductLineID: sockLine, Monthly: []MonthlyDemand{{Quantity: 500}}},
	}

	first, ok := ResolveLotDefault("it_greige", in)
	require.True(t, ok)
	for i := 0; i < 50; i++ {
		again, ok := ResolveLotDefault("it_greige", in)
		require.True(t, ok)
		assert.Equal(t, first.ProductLineID, again.ProductLineID)
		assert.Equal(t, first.UnitID, again.UnitID)
	}
}

// A brand-new SKU with no demand history still counts toward its line, or it would be
// invisible to the choice for as long as it takes to accumulate orders.
func TestResolveLotDefault_LineWithNoDemandStillVotes(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.DownstreamByItem["it_greige"] = []FinishedGood{
		{ItemID: "it_sock_new", ProductLineID: sockLine},
	}

	lot, ok := ResolveLotDefault("it_greige", in)
	require.True(t, ok)
	assert.Equal(t, sockLine, lot.ProductLineID)
}

func TestResolveLotDefault_ItemOverrideWins(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.ProductLineByItem["it_sock"] = sockLine
	in.UnitByItem["it_sock"] = pairs
	in.ItemOverrides["it_sock"] = 120

	lot, ok := ResolveLotDefault("it_sock", in)
	require.True(t, ok)
	assert.Equal(t, float64(120), lot.Quantity)
	assert.Equal(t, LotSourceItemOverride, lot.Source)
	// An override changes how big a lot is, not what it is counted in.
	assert.Equal(t, pairs, lot.UnitID)
}

func TestResolveLotDefault_FallsBackToTheAccountDefault(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.UnitByItem["it_orphan"] = eaches

	lot, ok := ResolveLotDefault("it_orphan", in)
	require.True(t, ok)
	assert.Equal(t, float64(60), lot.Quantity)
	assert.Equal(t, eaches, lot.UnitID)
	assert.Equal(t, LotSourceAccountDefault, lot.Source)
}

// A downstream line with no lot configured must not shadow the account default, or
// configuring one line would silently unlot everything that feeds a different one.
func TestResolveLotDefault_UnconfiguredDownstreamLineIsSkipped(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.UnitByItem["it_greige"] = eaches
	in.DownstreamByItem["it_greige"] = []FinishedGood{
		{ItemID: "it_other", ProductLineID: "pdln_unconfigured", Monthly: []MonthlyDemand{{Quantity: 900}}},
	}

	lot, ok := ResolveLotDefault("it_greige", in)
	require.True(t, ok)
	assert.Equal(t, LotSourceAccountDefault, lot.Source)
	assert.Equal(t, eaches, lot.UnitID)
}

func TestResolveLotDefault_NoLotAtAll(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.AccountLotUnits = 0

	_, ok := ResolveLotDefault("it_orphan", in)
	assert.False(t, ok, "an item with no rule anywhere is planned unlotted, not guessed at")
}
