//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the messaging directory (§12.4): internal users see active account_users in
// their account (including themselves), filtered by a name substring; customer relation actors see
// exactly one "support" contact and never the individual staff.

const messagingContactsPath = "/v1/messaging/contacts"

// contactActorID returns the actor id of a contact. Each contact is itself an actor, so the id lives
// at the top level.
func contactActorID(contact map[string]any) string {
	id, _ := contact["id"].(string)
	return id
}

func TestMessagingContacts_InternalUserSeesAccountUsers(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	list, status, err := user.GetList(messagingContactsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)

	var foundOther bool
	for _, raw := range list.Data {
		var contact map[string]any
		require.NoError(t, json.Unmarshal(raw, &contact))
		assert.Equal(t, "actor", contact["object"])
		if contactActorID(contact) == SeedAccountUser2ID {
			foundOther = true
		}
	}
	assert.True(t, foundOther, "the directory must include the other seeded account_user (user2)")
}

func TestMessagingContacts_QueryFilters(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	// An unfiltered list contains some contacts to filter against.
	all, status, err := user.GetList(messagingContactsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NotEmpty(t, all.Data, "the unfiltered directory must not be empty")

	// Pick a name substring from the first contact and confirm filtering narrows the result.
	var first map[string]any
	require.NoError(t, json.Unmarshal(all.Data[0], &first))
	name, _ := first["name"].(string)
	require.NotEmpty(t, name)
	substr := name[:1]

	filtered, status, err := user.GetList(messagingContactsPath, url.Values{"q": {substr}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	for _, raw := range filtered.Data {
		var contact map[string]any
		require.NoError(t, json.Unmarshal(raw, &contact))
		cname, _ := contact["name"].(string)
		assert.Contains(t, strings.ToLower(cname), strings.ToLower(substr), "filtered contacts must match the query substring")
	}

	// A query that matches nothing returns an empty (but valid) list.
	none, status, err := user.GetList(messagingContactsPath, url.Values{"q": {"zzzznomatchzzzz"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Empty(t, none.Data)
}

func TestMessagingContacts_CustomerSeesOnlySupport(t *testing.T) {
	t.Parallel()
	customer := getCustomerPortalClient()

	list, status, err := customer.GetList(messagingContactsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1, "a customer must see exactly one (support) contact")

	var contact map[string]any
	require.NoError(t, json.Unmarshal(list.Data[0], &contact))
	assert.Equal(t, "actor", contact["object"])
	// The support contact is a shared "Customer Service" group actor — it never exposes an
	// individual staff member (a `user` actor).
	assert.Equal(t, "group", contact["type"], "the support contact must be a shared group actor, not an individual")
}
