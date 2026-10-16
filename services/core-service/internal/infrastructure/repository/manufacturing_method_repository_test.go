package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

func TestPercentFactorConversion(t *testing.T) {
	for factor, percent := range map[string]string{
		"0.150000000000000000000000000000": "15",
		"0":                                "0",
		"0.125":                            "12.5",
		"1.000000000000000000000000000000": "100",
		"-0.05":                            "-5",
		"0.0001":                           "0.01",
	} {
		assert.Equal(t, percent, percentFromFactor(factor), factor)
	}
	for percent, factor := range map[string]string{"15": "0.15", "0": "0", "12.5": "0.125", "100": "1", "-5": "-0.05", "0.5": "0.005"} {
		assert.Equal(t, factor, factorFromPercent(percent), percent)
	}
}

func TestBuildManufacturingMethodListPage(t *testing.T) {
	page := buildManufacturingMethodListPage(domain.ListManufacturingMethodsParams{
		AccountID:     "ac_1",
		ItemIDs:       []string{"it_1"},
		Statuses:      []constants.ManufacturingMethodStatus{constants.ManufacturingMethodStatusActive},
		WorkCenterIDs: []string{"wc_1", "wc_2"},
	}, pagination.DirectionForward, nil, 26)
	query, args := page.sql()
	assert.Contains(t, query, "FORCE INDEX (manufacturing_method_item_id_created_at_id_idx)")
	assert.Contains(t, query, "m.status_code IN (?)")
	assert.Contains(t, query, "ps.department_id IN (?,?)")
	assert.NotContains(t, strings.ToUpper(query), "? IS NULL", "optional filters are built in Go")
	assert.Contains(t, args, "ac_1")
	assert.Contains(t, args, "wc_2")
}

func TestBuildValuePage(t *testing.T) {
	next := func(v int) string { return "n" + string(rune('0'+v)) }
	prev := func(v int) string { return "p" + string(rune('0'+v)) }

	items, pi := buildValuePage([]int{1, 2, 3}, 2, nil, next, prev)
	assert.Equal(t, []int{1, 2}, items)
	assert.Equal(t, "n2", *pi.NextCursor)
	assert.Nil(t, pi.PrevCursor)

	back := pagination.DirectionBackward
	items, pi = buildValuePage([]int{3, 2}, 2, &back, next, prev)
	assert.Equal(t, []int{2, 3}, items, "a backward page is read in reverse and returned in order")
	assert.Equal(t, "n3", *pi.NextCursor)
	assert.Nil(t, pi.PrevCursor)
}
