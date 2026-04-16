//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const volumeDiscountsPath = "/v1/sales/volume-discounts"

// ──────────────────────────────────────────────
// VolumeDiscount — Include Tests
// ──────────────────────────────────────────────

func TestVolumeDiscounts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["customer_groups"], "customer_groups should be null without include")
	assert.Nil(t, got["product_lines"], "product_lines should be null without include")
	assert.Nil(t, got["categories"], "categories should be null without include")
	assert.Nil(t, got["attributes"], "attributes should be null without include")
	assert.Nil(t, got["acceptable_units"], "acceptable_units should be null without include")
}

func TestVolumeDiscounts_IncludeCustomerGroups(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"customer_groups"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cg := jsonObject(got, "customer_groups")
	require.NotNil(t, cg, "customer_groups should be present with ?include=customer_groups")
	assert.Equal(t, "list", jsonField(cg, "object"))
}

func TestVolumeDiscounts_IncludeProductLines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"product_lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pl := jsonObject(got, "product_lines")
	require.NotNil(t, pl, "product_lines should be present with ?include=product_lines")
	assert.Equal(t, "list", jsonField(pl, "object"))
}

func TestVolumeDiscounts_IncludeCategories(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"categories"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cats := jsonObject(got, "categories")
	require.NotNil(t, cats, "categories should be present with ?include=categories")
	assert.Equal(t, "list", jsonField(cats, "object"))
}

func TestVolumeDiscounts_IncludeAttributes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	attrs := jsonObject(got, "attributes")
	require.NotNil(t, attrs, "attributes should be present with ?include=attributes")
	assert.Equal(t, "list", jsonField(attrs, "object"))
}

func TestVolumeDiscounts_IncludeAcceptableUnits(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"acceptable_units"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	units := jsonObject(got, "acceptable_units")
	require.NotNil(t, units, "acceptable_units should be present with ?include=acceptable_units")
	assert.Equal(t, "list", jsonField(units, "object"))
}
