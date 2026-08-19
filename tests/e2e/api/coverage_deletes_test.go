//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Delete endpoints the suite had never called.
//
// Each one is driven the same way: make the thing, delete it, then prove it is gone by
// reading it back — a delete that reports success while the row survives is the failure
// mode that matters, and the response body alone cannot tell you which happened.

// ──────────────────────────────────────────────
// Product types
// ──────────────────────────────────────────────

const productTypesPath = "/v1/catalog/product-types"

func TestProductTypes_DeleteRemovesIt(t *testing.T) {
	t.Parallel()

	name := uniqueName("cov-pdtp-del")
	created := createAndCleanup(t, productTypesPath, map[string]any{"name": name, "code": name})
	id := jsonField(created, "id")

	status, body, err := apiClient.Delete(productTypesPath + "/" + id)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(productTypesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "a deleted product type must not read back: %s", string(body))
}

func TestProductTypes_DeletingTwiceReportsItGone(t *testing.T) {
	t.Parallel()

	name := uniqueName("cov-pdtp-twice")
	created := createAndCleanup(t, productTypesPath, map[string]any{"name": name, "code": name})
	id := jsonField(created, "id")

	status, body, err := apiClient.Delete(productTypesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// The second delete has nothing to remove. Reporting success would tell a caller retrying a
	// failed request that it worked; reporting it gone distinguishes "someone deleted this"
	// from the mistyped-ID case below, which is a plain not-found.
	status, body, err = apiClient.Delete(productTypesPath + "/" + id)
	require.NoError(t, err)
	require.Less(t, status, 500, "second delete must not 5xx: %s", string(body))
	assert.Equal(t, 410, status, "deleting an already-deleted product type must report it gone: %s", string(body))
}

func TestProductTypes_DeleteUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(productTypesPath + "/pdtp_doesnotexist00")
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown product type must 404: %s", string(body))
}

// ──────────────────────────────────────────────
// DC locations
// ──────────────────────────────────────────────

func TestDCLocations_DeleteRemovesIt(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, dcLocationsPath, map[string]any{
		"customer_id": SeedCustomerAccountID,
		"location":    uniqueName("cov-dc-del"),
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Delete(dcLocationsPath + "/" + id)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(dcLocationsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "a deleted DC location must not read back: %s", string(body))
}

func TestDCLocations_DeleteUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(dcLocationsPath + "/dclc_doesnotexist00")
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown DC location must 404: %s", string(body))
}

// ──────────────────────────────────────────────
// Consumptions
// ──────────────────────────────────────────────

// newConsumption records a small consumption on the seeded production step and returns its ID.
func newConsumption(t *testing.T) string {
	t.Helper()

	status, body, err := apiClient.Post(consumptionsPath(), map[string]any{
		"item_id":                SeedConsumedItemID,
		"quantity_value":         "1",
		"quantity_unit_id":       SeedConsumptionUnitID,
		"waste_quantity_value":   "0",
		"waste_quantity_unit_id": SeedConsumptionUnitID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(consumptionsPath() + "/" + id) })
	return id
}

func TestConsumptions_DeleteRemovesIt(t *testing.T) {
	t.Parallel()

	id := newConsumption(t)

	status, body, err := apiClient.Delete(consumptionsPath() + "/" + id)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(consumptionsPath()+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "a deleted consumption must not read back: %s", string(body))
}

func TestConsumptions_DeleteUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(consumptionsPath() + "/cp_doesnotexist0000")
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown consumption must 404: %s", string(body))
}

// A consumption belongs to one step. Deleting it through a different step's path would
// let a caller reach across the flow graph using an ID they happened to learn.
func TestConsumptions_DeleteThroughTheWrongStepIs404(t *testing.T) {
	t.Parallel()

	id := newConsumption(t)

	status, body, err := apiClient.Delete("/v1/operations/production-steps/prs_doesnotexist000/consumptions/" + id)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "a consumption must not be reachable through another step: %s", string(body))

	status, body, err = apiClient.GetListRaw(consumptionsPath()+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status, "the consumption must survive the misdirected delete: %s", string(body))
}

// ──────────────────────────────────────────────
// Account-group product line access
// ──────────────────────────────────────────────

const accountGroupAccessPath = "/v1/sales/product-line-access/account-groups"

// newAccountGroup creates a throwaway group so an access grant can be revoked without
// disturbing the seeded groups other tests read.
func newAccountGroup(t *testing.T) string {
	t.Helper()

	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("cov-acgrp"),
		"type": "type_group",
	})
	return jsonField(created, "id")
}

func TestAccountGroupProductLineAccess_DeleteRevokesEverything(t *testing.T) {
	t.Parallel()

	groupID := newAccountGroup(t)

	status, body, err := apiClient.Post(accountGroupAccessPath, map[string]any{
		"account_group_id": groupID,
		"product_line_ids": []string{SeedProductLineID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "grant must not 5xx: %s", string(body))
	requireStatus(t, 201, status, body)

	status, body, err = apiClient.Delete(accountGroupAccessPath + "/" + groupID)
	require.NoError(t, err)
	require.Less(t, status, 500, "revoke must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	// Access is a grant list, so revoking it leaves the group with none — the group itself remains.
	status, body, err = apiClient.GetListRaw(accountGroupAccessPath+"/"+groupID, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "read-back must not 5xx: %s", string(body))
	if status == 200 {
		assert.NotContains(t, string(body), SeedProductLineID, "the revoked product line must be gone: %s", string(body))
	} else {
		assert.Equal(t, 404, status, "a fully revoked group must read as 404 or an empty grant: %s", string(body))
	}
}

func TestAccountGroupProductLineAccess_DeleteUnknownGroupIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(accountGroupAccessPath + "/acgr_doesnotexist00")
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "revoking access for an unknown group must 404: %s", string(body))
}

// ──────────────────────────────────────────────
// Settlements and their allocations
// ──────────────────────────────────────────────

// newSettlement records a payment and settles a token amount of it against the seeded
// invoice, returning the settlement and its first allocation ID.
func newSettlement(t *testing.T) (settlementID, allocationID string) {
	t.Helper()

	txn := createAndCleanup(t, transactionsPath, map[string]any{
		"customer_id":         SeedCustomerAccountID,
		"type":                "payment",
		"amount":              "1.00",
		"responsible_user_id": SeedAccountUserID,
	})

	status, body, err := apiClient.Post(settlementsPath, map[string]any{
		"responsible_user_id": SeedAccountUserID,
		"allocations": []map[string]any{{
			"transaction_id": jsonField(txn, "id"),
			"invoice_id":     SeedSettlementInvoiceID,
			"amount":         "1.00",
		}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "settlement create must not 5xx: %s", string(body))
	requireStatus(t, 201, status, body)

	created := parseJSON(body)
	settlementID = jsonField(created, "id")
	require.NotEmpty(t, settlementID)
	t.Cleanup(func() { apiClient.Delete(settlementsPath + "/" + settlementID) })

	// Allocations are expandable, so the create response leaves them null; the read is
	// where the recorded rows become visible.
	status, body, err = apiClient.GetListRaw(settlementsPath+"/"+settlementID, url.Values{"include": {"allocations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	allocations := jsonArray(jsonObject(parseJSON(body), "allocations"), "data")
	require.NotEmpty(t, allocations, "a settlement must report the allocations it recorded: %s", string(body))
	first, ok := allocations[0].(map[string]any)
	require.True(t, ok, "allocations must be objects: %s", string(body))
	allocationID = jsonField(first, "id")
	require.NotEmpty(t, allocationID)

	return settlementID, allocationID
}

func TestSettlements_DeleteRemovesIt(t *testing.T) {
	t.Parallel()

	settlementID, _ := newSettlement(t)

	status, body, err := apiClient.Delete(settlementsPath + "/" + settlementID)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(settlementsPath+"/"+settlementID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "a deleted settlement must not read back: %s", string(body))
}

// The money a settlement applied has to be released with it, or the invoice stays
// credited by a settlement that no longer exists.
func TestSettlements_DeleteReleasesItsAllocations(t *testing.T) {
	t.Parallel()

	settlementID, allocationID := newSettlement(t)

	status, body, err := apiClient.Delete(settlementsPath + "/" + settlementID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Allocations have no retrieve endpoint, so a delete that finds nothing is the read.
	status, body, err = apiClient.Delete("/v1/finance/transaction-allocations/" + allocationID)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an allocation must not outlive its settlement: %s", string(body))
}

func TestSettlements_DeleteUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(settlementsPath + "/sl_doesnotexist00000")
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown settlement must 404: %s", string(body))
}

func TestTransactionAllocations_DeleteRemovesIt(t *testing.T) {
	t.Parallel()

	_, allocationID := newSettlement(t)

	status, body, err := apiClient.Delete("/v1/finance/transaction-allocations/" + allocationID)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	// A deleted allocation is remembered as gone, which is a more useful answer than
	// not-found: the caller learns the row existed and was removed, not that they mistyped.
	status, body, err = apiClient.Delete("/v1/finance/transaction-allocations/" + allocationID)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 410, status, "a deleted allocation must report itself gone: %s", string(body))
}

func TestTransactionAllocations_DeleteUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete("/v1/finance/transaction-allocations/txal_doesnotexist")
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown allocation must 404: %s", string(body))
}

// A settlement applies money that a transaction holds against an invoice. Both have to
// belong to the caller, or a settlement could quietly move another tenant's payment.
func TestSettlements_RejectAllocationsAgainstUnknownRecords(t *testing.T) {
	t.Parallel()

	txn := createAndCleanup(t, transactionsPath, map[string]any{
		"customer_id":         SeedCustomerAccountID,
		"type":                "payment",
		"amount":              "1.00",
		"responsible_user_id": SeedAccountUserID,
	})

	for name, alloc := range map[string]map[string]any{
		"unknown transaction": {"transaction_id": "tx_doesnotexist00000", "invoice_id": SeedSettlementInvoiceID, "amount": "1.00"},
		"unknown invoice":     {"transaction_id": jsonField(txn, "id"), "invoice_id": "iv_doesnotexist00000", "amount": "1.00"},
	} {
		t.Run(name, func(t *testing.T) {
			status, body, err := apiClient.Post(settlementsPath, map[string]any{
				"responsible_user_id": SeedAccountUserID,
				"allocations":         []map[string]any{alloc},
			}, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "%s must not 5xx: %s", name, string(body))
			assert.Contains(t, []int{400, 404}, status, "%s must be rejected: %s", name, string(body))
		})
	}
}

func TestSettlements_RejectAnUnknownResponsibleUser(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(settlementsPath, map[string]any{
		"responsible_user_id": "acus_doesnotexist",
		"allocations":         []map[string]any{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 404}, status, "an unknown responsible user must be rejected: %s", string(body))
}

// ──────────────────────────────────────────────
// Child accounts
// ──────────────────────────────────────────────

const childAccountsPath = "/v1/identity/child-accounts"

// childListHasAccount reports whether the child list links the given account. Each entry is
// identified by its relation ID, so the account being linked is one level down.
func childListHasAccount(list *ListResponse, accountID string) bool {
	for _, entry := range list.Data {
		if acct := jsonObject(parseJSON(entry), "account"); acct != nil && jsonField(acct, "id") == accountID {
			return true
		}
	}
	return false
}

func TestChildAccounts_AddAndRemove(t *testing.T) {
	t.Parallel()

	child := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("cov-child")))
	childID := jsonField(child, "id")

	status, body, err := apiClient.Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "add must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	list, status, err := apiClient.GetList(childAccountsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.True(t, childListHasAccount(list, childID), "the linked account must appear in the child list")

	status, body, err = apiClient.Delete(childAccountsPath + "/" + childID)
	require.NoError(t, err)
	require.Less(t, status, 500, "remove must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	list, status, err = apiClient.GetList(childAccountsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.False(t, childListHasAccount(list, childID), "an unlinked account must leave the child list")

	// Unlinking only breaks the relationship — the customer record itself is the seller's
	// own data and must survive, or removing a store location would erase the customer.
	status, body, err = apiClient.GetListRaw(customersPath+"/"+childID, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status, "the customer must outlive the parent-child link: %s", string(body))
}

// Both calls are documented as idempotent, which is what makes them safe to retry after a
// timeout — the caller cannot tell whether the first attempt landed.
func TestChildAccounts_AddAndRemoveAreIdempotent(t *testing.T) {
	t.Parallel()

	child := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("cov-child-idem")))
	childID := jsonField(child, "id")

	for i := range 2 {
		status, body, err := apiClient.Put(childAccountsPath+"/"+childID, nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "add %d must not 5xx: %s", i, string(body))
		assert.Equal(t, 200, status, "linking an already-linked child must succeed: %s", string(body))
	}

	for i := range 2 {
		status, body, err := apiClient.Delete(childAccountsPath + "/" + childID)
		require.NoError(t, err)
		require.Less(t, status, 500, "remove %d must not 5xx: %s", i, string(body))
		assert.Equal(t, 200, status, "unlinking an unlinked child must succeed: %s", string(body))
	}
}

// An account cannot be its own ancestor, or walking the hierarchy would never terminate.
func TestChildAccounts_RejectACycle(t *testing.T) {
	t.Parallel()

	child := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("cov-child-cycle")))
	childID := jsonField(child, "id")

	status, body, err := apiClient.Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { apiClient.Delete(childAccountsPath + "/" + childID) })

	// Linking the seller's own account beneath its child closes the loop.
	scoped := apiClient.WithAccountID(childID)
	status, body, err = scoped.Put(childAccountsPath+"/"+SeedAccountID, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.NotEqual(t, 200, status, "a circular hierarchy must be rejected: %s", string(body))
}

func TestChildAccounts_AddUnknownAccountIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(childAccountsPath+"/ac_doesnotexist00000", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{403, 404}, status, "an unknown account cannot be linked: %s", string(body))
}

// ──────────────────────────────────────────────
// Transactions
// ──────────────────────────────────────────────

// Money moving has to be attributable to a person. An API key is not one, so it must name
// who is responsible rather than have the server guess and fail on a lookup that cannot match.
func TestTransactions_RequireAResponsibleUserWhenTheCallerIsNotAUser(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(transactionsPath, map[string]any{
		"customer_id": SeedCustomerAccountID,
		"type":        "payment",
		"amount":      "1.00",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "an API key must be told to name a responsible user: %s", string(body))
	assert.Contains(t, string(body), "responsible_user_id", "the error must name the missing field: %s", string(body))
}

// transaction_allocation carries no foreign keys, so nothing but this check stops a
// settlement from applying another tenant's payment against another tenant's invoice.
func TestSettlements_RejectAnotherTenantsRecords(t *testing.T) {
	t.Parallel()

	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	status, body, err := tenantB.Post(settlementsPath, map[string]any{
		"responsible_user_id": SeedTenantBAccountUserID,
		"allocations": []map[string]any{{
			"transaction_id": SeedTransactionID,
			"invoice_id":     SeedSettlementInvoiceID,
			"amount":         "1.00",
		}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 403, 404}, status, "another tenant's payment must not be settleable: %s", string(body))
}
