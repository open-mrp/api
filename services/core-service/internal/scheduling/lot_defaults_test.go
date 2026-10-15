package scheduling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sockCategory   = "ic_socks"
	sleeveCategory = "ic_sleeves"
	pairs          = "un_pairs"
	eaches         = "un_eaches"
)

func baseLotInput() LotResolutionInput {
	return LotResolutionInput{
		ItemOverrides:     map[string]float64{},
		CategoryLotByItem: map[string]CategoryLot{},
		AccountLotUnits:   60,
		UnitByItem:        map[string]string{},
	}
}

// The same 60 means pairs for socks and eaches for armsleeves, and the category's unit is what separates them.
func TestResolveLotDefault_UnitComesFromTheCategory(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.CategoryLotByItem["it_sock"] = CategoryLot{ItemCategoryID: sockCategory, Quantity: 60, UnitID: pairs}
	in.CategoryLotByItem["it_sleeve"] = CategoryLot{ItemCategoryID: sleeveCategory, Quantity: 60, UnitID: eaches}

	sock, ok := ResolveLotDefault("it_sock", in)
	require.True(t, ok)
	assert.Equal(t, float64(60), sock.Quantity)
	assert.Equal(t, pairs, sock.UnitID)
	assert.Equal(t, LotSourceItemCategory, sock.Source)
	assert.Equal(t, sockCategory, sock.ItemCategoryID)

	sleeve, ok := ResolveLotDefault("it_sleeve", in)
	require.True(t, ok)
	assert.Equal(t, eaches, sleeve.UnitID)
}

func TestResolveLotDefault_ItemOverrideWins(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.CategoryLotByItem["it_sock"] = CategoryLot{ItemCategoryID: sockCategory, Quantity: 60, UnitID: pairs}
	in.ItemOverrides["it_sock"] = 120
	in.UnitByItem["it_sock"] = eaches

	lot, ok := ResolveLotDefault("it_sock", in)
	require.True(t, ok)
	assert.Equal(t, float64(120), lot.Quantity)
	assert.Equal(t, eaches, lot.UnitID)
	assert.Equal(t, LotSourceItemOverride, lot.Source)
	assert.Empty(t, lot.ItemCategoryID)
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

func TestResolveLotDefault_ZeroCategoryLotIsSkipped(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.CategoryLotByItem["it_sock"] = CategoryLot{ItemCategoryID: sockCategory, Quantity: 0, UnitID: pairs}

	lot, ok := ResolveLotDefault("it_sock", in)
	require.True(t, ok)
	assert.Equal(t, LotSourceAccountDefault, lot.Source)
}

func TestResolveLotDefault_NoLotAtAll(t *testing.T) {
	t.Parallel()

	in := baseLotInput()
	in.AccountLotUnits = 0

	_, ok := ResolveLotDefault("it_orphan", in)
	assert.False(t, ok)
}
