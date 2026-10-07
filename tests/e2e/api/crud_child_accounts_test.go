//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// childEntry is the child list's entry for accountID, or nil.
func childEntry(list *ListResponse, accountID string) map[string]any {
	for _, raw := range list.Data {
		entry := parseJSON(raw)
		if acct := jsonObject(entry, "account"); acct != nil && jsonField(acct, "id") == accountID {
			return entry
		}
	}
	return nil
}

// The customer pages link store locations under a head office: the seller targets the head office
// (the parent customer) and links another of its customers beneath it.
func TestChildAccounts_ASellerLinksAChildUnderItsCustomer(t *testing.T) {
	t.Parallel()
	parentID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-head-office"))), "id")
	child := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-store")))
	childID := jsonField(child, "id")
	asParent := apiClient.WithAccountID(parentID)

	status, body, err := asParent.Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _, _ = asParent.Delete(childAccountsPath + "/" + childID) })

	list, status, err := asParent.GetList(childAccountsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	entry := childEntry(list, childID)
	require.NotNil(t, entry, "the child is listed under its parent")
	assert.Equal(t, "child_account", entry["object"])
	assert.Equal(t, jsonField(child, "name"), jsonField(jsonObject(entry, "account"), "name"))
	assert.Equal(t, jsonField(child, "number"), jsonField(entry, "external_number"),
		"the child carries the seller's number for it, as the customer page shows")
	require.Len(t, list.Data, 1, "a new parent has only the child just linked")

	// The child sits under the parent, not under the seller's own account.
	own, _, err := apiClient.GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.Nil(t, childEntry(own, childID), "the child is not listed under the seller itself")

	// The search matches the child's name.
	found, _, err := asParent.GetList(childAccountsPath, map[string][]string{"q": {jsonField(child, "name")}})
	require.NoError(t, err)
	assert.NotNil(t, childEntry(found, childID))
	missing, _, err := asParent.GetList(childAccountsPath, map[string][]string{"q": {"zzznomatchzzz"}})
	require.NoError(t, err)
	assert.Empty(t, missing.Data)

	status, body, err = asParent.Delete(childAccountsPath + "/" + childID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	list, _, err = asParent.GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.Nil(t, childEntry(list, childID), "an unlinked child leaves its parent's list")

	status, body, err = apiClient.GetListRaw(customersPath+"/"+childID, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status, "unlinking leaves the customer: %s", body)
}

// Moving a child to another parent takes it out of the first parent's list.
func TestChildAccounts_ARelinkMovesTheChild(t *testing.T) {
	t.Parallel()
	first := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-parent-a"))), "id")
	second := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-parent-b"))), "id")
	childID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-moved"))), "id")

	status, body, err := apiClient.WithAccountID(first).Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = apiClient.WithAccountID(second).Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _, _ = apiClient.WithAccountID(second).Delete(childAccountsPath + "/" + childID) })

	underFirst, _, err := apiClient.WithAccountID(first).GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.Nil(t, childEntry(underFirst, childID), "an account has one parent")
	underSecond, _, err := apiClient.WithAccountID(second).GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.NotNil(t, childEntry(underSecond, childID))

	// Removing it from a parent it no longer sits under changes nothing.
	status, body, err = apiClient.WithAccountID(first).Delete(childAccountsPath + "/" + childID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	underSecond, _, err = apiClient.WithAccountID(second).GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.NotNil(t, childEntry(underSecond, childID), "removing under the wrong parent leaves the link")
}

// An account cannot become a child of its own child.
func TestChildAccounts_ACustomerCannotParentItsParent(t *testing.T) {
	t.Parallel()
	parentID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cycle-parent"))), "id")
	childID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cycle-child"))), "id")

	status, body, err := apiClient.WithAccountID(parentID).Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _, _ = apiClient.WithAccountID(parentID).Delete(childAccountsPath + "/" + childID) })

	status, body, err = apiClient.WithAccountID(childID).Put(childAccountsPath+"/"+parentID, nil)
	require.NoError(t, err)
	assert.Equal(t, 409, status, "a cycle is refused: %s", body)
}

// Another seller can neither see nor change this seller's customer hierarchy, and an account that is
// not one of the seller's customers cannot be linked.
func TestChildAccounts_AreTheSellersOwn(t *testing.T) {
	t.Parallel()
	parentID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-own-parent"))), "id")
	childID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-own-child"))), "id")
	status, body, err := apiClient.WithAccountID(parentID).Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _, _ = apiClient.WithAccountID(parentID).Delete(childAccountsPath + "/" + childID) })

	other := getTenantBClient().WithAccountID(parentID)
	status, body, err = other.GetListRaw(childAccountsPath, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status, "another seller cannot list them: %s", body)
	status, body, err = other.Delete(childAccountsPath + "/" + childID)
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status, "another seller cannot unlink them: %s", body)

	status, body, err = apiClient.WithAccountID(parentID).Put(childAccountsPath+"/ac_doesnotexist0000", nil)
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status, "an account that is not a customer cannot be linked: %s", body)

	list, _, err := apiClient.WithAccountID(parentID).GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.NotNil(t, childEntry(list, childID), "nothing the others tried changed the link")
}
