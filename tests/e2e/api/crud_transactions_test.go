//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const transactionsPath = "/v1/finance/transactions"

// ──────────────────────────────────────────────
// Transaction — Include Tests
// ──────────────────────────────────────────────

func TestTransactions_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(transactionsPath+"/"+SeedTransactionID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["allocations"], "allocations should be null without ?include=allocations")
	assert.Nil(t, got["customer"], "customer should be null without ?include=customer")
	assert.Nil(t, got["responsible_user"], "responsible_user should be null without ?include=responsible_user")
}

func TestTransactions_IncludeAllocations(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(transactionsPath+"/"+SeedTransactionID, url.Values{"include": {"allocations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["allocations"]
	assert.True(t, ok, "allocations key should be present with ?include=allocations")
	if a := jsonObject(got, "allocations"); a != nil {
		assert.Equal(t, "list", jsonField(a, "object"))
	}
}

func TestTransactions_IncludeCustomer(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(transactionsPath+"/"+SeedTransactionID, url.Values{"include": {"customer"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["customer"]
	assert.True(t, ok, "customer key should be present with ?include=customer")
	if cust := jsonObject(got, "customer"); cust != nil {
		assert.Equal(t, "customer", jsonField(cust, "object"))
	}
}

func TestTransactions_IncludeResponsibleUser(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(transactionsPath+"/"+SeedTransactionID, url.Values{"include": {"responsible_user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["responsible_user"]
	assert.True(t, ok, "responsible_user key should be present with ?include=responsible_user")
	if u := jsonObject(got, "responsible_user"); u != nil {
		assert.Equal(t, "account_user", jsonField(u, "object"))
	}
}
