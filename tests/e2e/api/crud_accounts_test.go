//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountsPath = "/v1/identity/accounts"

// ──────────────────────────────────────────────
// Account — Include Tests
// ──────────────────────────────────────────────
//
// Account GET endpoint whitelists: branding, portal.
// (default_billing_address and default_shipping_address are always populated.)

func TestAccounts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(accountsPath+"/"+SeedAccountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["branding"], "branding should be null without ?include=branding")
	assert.Nil(t, got["portal"], "portal should be null without ?include=portal")
}

func TestAccounts_IncludeBranding(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountsPath+"/"+SeedAccountID, url.Values{"include": {"branding"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	branding := jsonObject(got, "branding")
	require.NotNil(t, branding, "branding should be present with ?include=branding")
	assert.Equal(t, "account_branding", jsonField(branding, "object"))
}

func TestAccounts_IncludePortal(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountsPath+"/"+SeedAccountID, url.Values{"include": {"portal"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	portal := jsonObject(got, "portal")
	require.NotNil(t, portal, "portal should be present with ?include=portal")
	assert.Equal(t, "account_portal", jsonField(portal, "object"))
}
