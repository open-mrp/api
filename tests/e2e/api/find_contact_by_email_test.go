//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const contactsFindByEmailPath = "/v1/sales/contacts/actions/find-by-email"

// createSelfContact creates an account user on the test account itself with a unique
// email and returns its id + email. find-by-email should then resolve that email to a
// `self` contact match. Cleaned up automatically.
func createSelfContact(t *testing.T) (accountUserID, email string) {
	t.Helper()
	name := uniqueName("e2e-contact")
	email = name + "@e2e-test.augno.com"

	resp, err := apiClient.PostFull(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	accountUserID = jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, accountUserID)
	t.Cleanup(func() { removeAccountUser(accountUserID) })
	return accountUserID, email
}

// findContacts POSTs the find-by-email action and returns the list items as maps.
// params carries query-string filters/expansions (relationships, include).
func findContacts(t *testing.T, email string, params url.Values) []map[string]any {
	t.Helper()
	path := contactsFindByEmailPath
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	status, body, err := apiClient.Post(path, map[string]any{"email": email}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	require.Equal(t, "list", jsonField(m, "object"))

	raw := jsonArray(m, "data")
	out := make([]map[string]any, 0, len(raw))
	for _, it := range raw {
		out = append(out, it.(map[string]any))
	}
	return out
}

func findContactMatchByID(items []map[string]any, id string) map[string]any {
	for _, it := range items {
		if jsonField(it, "id") == id {
			return it
		}
	}
	return nil
}

// requireSelfMatch polls find-by-email until the just-created self contact is visible
// (create -> read is same-DB, but poll to absorb any propagation) and returns the match.
func requireSelfMatch(t *testing.T, email, accountUserID string, params url.Values) map[string]any {
	t.Helper()
	var match map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if m := findContactMatchByID(findContacts(t, email, params), accountUserID); m != nil {
			match = m
			return nil
		}
		return fmt.Errorf("self contact %s not found by email yet", accountUserID)
	})
	return match
}

func TestContacts_FindByEmail_SelfMatch(t *testing.T) {
	t.Parallel()
	accountUserID, email := createSelfContact(t)

	match := requireSelfMatch(t, email, accountUserID, nil)

	assert.Equal(t, "contact_match", jsonField(match, "object"))
	assert.Equal(t, accountUserID, jsonField(match, "id"))
	assert.Equal(t, "self", jsonField(match, "relationship"))
	assert.Equal(t, email, jsonField(match, "email"))

	// Expandables are null until requested.
	assert.Nil(t, match["account_user"], "account_user should be null without ?include")
	assert.Nil(t, match["account"], "account should be null without ?include")
}

func TestContacts_FindByEmail_Includes(t *testing.T) {
	t.Parallel()
	accountUserID, email := createSelfContact(t)
	requireSelfMatch(t, email, accountUserID, nil)

	// account_user alone: present, but its nested user stays null (gating is per-key).
	match := findContactMatchByID(
		findContacts(t, email, url.Values{"include": {"account_user"}}),
		accountUserID,
	)
	require.NotNil(t, match)
	au := jsonObject(match, "account_user")
	require.NotNil(t, au, "account_user should be expanded with ?include=account_user")
	assert.Equal(t, "account_user", jsonField(au, "object"))
	assert.Equal(t, accountUserID, jsonField(au, "id"))
	assert.Nil(t, au["user"], "account_user.user should be null without ?include=account_user.user")
	assert.Nil(t, match["account"], "account should be null without ?include=account")

	// account_user.user + account: nested user and the account both hydrate.
	match = findContactMatchByID(
		findContacts(t, email, url.Values{"include": {"account_user,account_user.user,account"}}),
		accountUserID,
	)
	require.NotNil(t, match)

	au = jsonObject(match, "account_user")
	require.NotNil(t, au, "account_user should be expanded")
	user := jsonObject(au, "user")
	require.NotNil(t, user, "account_user.user should be expanded with ?include=account_user.user")
	assert.Equal(t, "user", jsonField(user, "object"))
	assert.Equal(t, email, jsonField(user, "email"))

	acct := jsonObject(match, "account")
	require.NotNil(t, acct, "account should be expanded with ?include=account")
	assert.Equal(t, "account", jsonField(acct, "object"))
	assert.Equal(t, SeedAccountID, jsonField(acct, "id"), "a self contact's account is the caller's own account")
}

func TestContacts_FindByEmail_RelationshipFilter(t *testing.T) {
	t.Parallel()
	accountUserID, email := createSelfContact(t)
	requireSelfMatch(t, email, accountUserID, nil)

	// Filtering to the match's own relationship keeps it (and every returned row matches).
	selfItems := findContacts(t, email, url.Values{"relationships": {"self"}})
	require.NotNil(t, findContactMatchByID(selfItems, accountUserID),
		"self contact should be returned when filtering relationships=self")
	for _, it := range selfItems {
		assert.Equal(t, "self", jsonField(it, "relationship"))
	}

	// Filtering to other relationships excludes the self match.
	customerItems := findContacts(t, email, url.Values{"relationships": {"customer"}})
	assert.Nil(t, findContactMatchByID(customerItems, accountUserID),
		"self contact should be excluded when filtering relationships=customer")

	// Multiple relationship values are OR'd; neither is 'self', so still excluded.
	multiItems := findContacts(t, email, url.Values{"relationships": {"customer", "supplier"}})
	assert.Nil(t, findContactMatchByID(multiItems, accountUserID),
		"self contact should be excluded when filtering relationships=customer,supplier")
}

func TestContacts_FindByEmail_NoMatch(t *testing.T) {
	t.Parallel()
	email := uniqueName("e2e-nomatch") + "@e2e-test.augno.com"
	assert.Empty(t, findContacts(t, email, nil),
		"an email with no related contact returns an empty list, not an error")
}

func TestContacts_FindByEmail_Validation(t *testing.T) {
	t.Parallel()

	missingStatus, missingBody, err := apiClient.Post(contactsFindByEmailPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, missingStatus, missingBody)

	invalidStatus, invalidBody, err := apiClient.Post(contactsFindByEmailPath, map[string]any{"email": "not-an-email"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, invalidStatus, invalidBody)
}
