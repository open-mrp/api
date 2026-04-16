//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountPricesPath = "/v1/sales/account-prices"

// ──────────────────────────────────────────────
// AccountPrice — Include Tests
// ──────────────────────────────────────────────

func TestAccountPrices_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["recipient_account"], "recipient_account should be null without include")
	assert.Nil(t, got["product_line"], "product_line should be null without include")
	assert.Nil(t, got["categories"], "categories should be null without include")
	assert.Nil(t, got["attributes"], "attributes should be null without include")
}

func TestAccountPrices_IncludeRecipientAccount(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, url.Values{"include": {"recipient_account"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["recipient_account"]
	assert.True(t, ok, "recipient_account key should be present with ?include=recipient_account")
	if acc := jsonObject(got, "recipient_account"); acc != nil {
		// account or customer object (recipient is typically an account)
		obj := jsonField(acc, "object")
		assert.NotEmpty(t, obj)
		assert.NotEmpty(t, jsonField(acc, "id"))
	}
}

func TestAccountPrices_IncludeProductLine(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["product_line"]
	assert.True(t, ok, "product_line key should be present with ?include=product_line")
	if pl := jsonObject(got, "product_line"); pl != nil {
		assert.Equal(t, "product_line", jsonField(pl, "object"))
	}
}

func TestAccountPrices_IncludeCategories(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, url.Values{"include": {"categories"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cats := jsonObject(got, "categories")
	require.NotNil(t, cats, "categories should be present with ?include=categories")
	assert.Equal(t, "list", jsonField(cats, "object"))
}

func TestAccountPrices_IncludeAttributes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	attrs := jsonObject(got, "attributes")
	require.NotNil(t, attrs, "attributes should be present with ?include=attributes")
	assert.Equal(t, "list", jsonField(attrs, "object"))
}
