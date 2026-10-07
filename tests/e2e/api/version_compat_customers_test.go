//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// preview.6 made the commission settings on customers, account groups and account users nullable, since they are withheld from customer and supplier portal users. preview.5 always carried a value, so a portal pinned to it reads "" for a withheld commission policy and false for a withheld commission eligibility; the seller's own callers read the real values on both versions.

func TestVersionCompat_Customers_PortalReadsWithheldCommissionAsPreview5ZeroValues(t *testing.T) {
	t.Parallel()
	path := customersPath + "/" + SeedCustomerAccountID
	include := url.Values{"include": {"type", "price_groups", "defaults.sales_rep"}}

	own := parseJSON(mustGetAs(t, getCustomerPortalClient().WithAPIVersion(preview5APIVersion), path, include))
	assert.Equal(t, "", own["commission_policy"], "a withheld customer commission policy reads as an empty string")
	assert.Nil(t, own["note"], "the note was already nullable and stays null")
	assert.Equal(t, "", jsonObject(own, "type")["commission_policy"], "a withheld group commission policy reads as an empty string")
	for _, group := range jsonListData(own, "price_groups") {
		assert.Equal(t, "", group.(map[string]any)["commission_policy"])
	}
	rep := jsonObject(jsonObject(own, "defaults"), "sales_rep")
	require.NotNil(t, rep, "the seed customer has a sales rep")
	assert.Equal(t, false, rep["is_commission_eligible"], "a withheld commission eligibility reads as false")
	assert.Nil(t, rep["last_used_at"], "last_used_at was already nullable and stays null")

	for _, apiVersion := range []string{preview5APIVersion, defaultAPIVersion} {
		staff := parseJSON(mustGetAs(t, apiClient.WithAPIVersion(apiVersion), path, include))
		assert.Contains(t, []string{"commission_applied", "commission_exempt"}, staff["commission_policy"], apiVersion)
		assert.Contains(t, []string{"commission_applied", "commission_exempt"}, jsonObject(staff, "type")["commission_policy"], apiVersion)
		_, isBool := jsonObject(jsonObject(staff, "defaults"), "sales_rep")["is_commission_eligible"].(bool)
		assert.True(t, isBool, "%s: the seller reads its rep's commission eligibility", apiVersion)
	}
}
