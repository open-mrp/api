//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The customer notification-recipients endpoints manage the default order-notification
// recipients (order acknowledgement / invoice) for a seller<->customer relationship:
//
//	GET   /v1/sales/customers/{id}/notification-recipients
//	PATCH /v1/sales/customers/{id}/notification-recipients   (full replace)
//
// Recipients are account users on the CUSTOMER (buyer) account. `account_user` is expandable
// via ?include=account_user. Both internal (seller) actors and customer-portal (buyer) actors
// may read; the seller and the buyer (self) may replace. No preferences are seeded, so tests
// set state then reset it.

func notifRecipientsPath(customerID string) string {
	return "/v1/sales/customers/" + customerID + "/notification-recipients"
}

// notifRecipientsInclude expands the account user on reads.
var notifRecipientsInclude = url.Values{"include": {"account_user"}}

// notifRecipientBody builds a PATCH body for the given (accountUserID -> types) recipients.
func notifRecipientBody(recipients ...map[string]any) map[string]any {
	if recipients == nil {
		recipients = []map[string]any{}
	}
	return map[string]any{"recipients": recipients}
}

// findNotifRecipient returns the recipient row in a list `data` array whose account_user.id
// matches, or nil if absent. Requires the response to have been fetched with ?include=account_user.
func findNotifRecipient(data []any, accountUserID string) map[string]any {
	for _, d := range data {
		item, ok := d.(map[string]any)
		if !ok {
			continue
		}
		au := jsonObject(item, "account_user")
		if au != nil && jsonField(au, "id") == accountUserID {
			return item
		}
	}
	return nil
}

// notifRecipientTypes extracts the notification_types string slice from a recipient row.
func notifRecipientTypes(t *testing.T, item map[string]any) []string {
	t.Helper()
	raw := jsonArray(item, "notification_types")
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		require.True(t, ok, "notification_types must be strings, got %T", v)
		out = append(out, s)
	}
	return out
}

// TestCustomerNotificationRecipients_Lifecycle is the single stateful flow: it is the only
// test that mutates the seeded relation's recipients, so it runs its subtests sequentially
// and resets to empty on cleanup. It covers set -> list -> buyer-read -> buyer-self-update
// -> clear across both the seller and customer-portal actors.
func TestCustomerNotificationRecipients_Lifecycle(t *testing.T) {
	t.Parallel()

	path := notifRecipientsPath(SeedCustomerAccountID)
	pathWithInclude := path + "?include=account_user"
	buyer := getCustomerPortalClient()

	// Always leave the relation with no configured recipients.
	t.Cleanup(func() {
		_, _, _ = apiClient.Patch(path, notifRecipientBody(), newIdempotencyKey())
	})

	// Establish a clean baseline regardless of prior runs.
	resetStatus, resetBody, err := apiClient.Patch(path, notifRecipientBody(), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resetStatus, resetBody)

	t.Run("SellerStartsEmpty", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		parsed := parseJSON(body)
		assert.Equal(t, "list", jsonField(parsed, "object"))
		assert.Empty(t, jsonArray(parsed, "data"), "no recipients should be configured yet")
	})

	t.Run("SellerSetRecipient", func(t *testing.T) {
		status, body, err := apiClient.Patch(pathWithInclude, notifRecipientBody(map[string]any{
			"account_user_id":    SeedCustomerAccountUserID,
			"notification_types": []string{"order_acknowledgement", "invoice"},
		}), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		parsed := parseJSON(body)
		assert.Equal(t, "list", jsonField(parsed, "object"))
		data := jsonArray(parsed, "data")
		require.Len(t, data, 1, "exactly one recipient should be configured")

		recipient := findNotifRecipient(data, SeedCustomerAccountUserID)
		require.NotNil(t, recipient, "seeded buyer account user should be a recipient")
		assert.Equal(t, "order_notification_recipient", jsonField(recipient, "object"))

		au := jsonObject(recipient, "account_user")
		require.NotNil(t, au, "account_user must be hydrated with ?include=account_user")
		assert.Equal(t, SeedCustomerAccountUserID, jsonField(au, "id"))
		assert.Equal(t, "account_user", jsonField(au, "object"))

		assert.ElementsMatch(t, []string{"order_acknowledgement", "invoice"}, notifRecipientTypes(t, recipient))
	})

	t.Run("SellerGetReflectsSet", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(path, notifRecipientsInclude)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		data := jsonArray(parseJSON(body), "data")
		require.Len(t, data, 1)
		recipient := findNotifRecipient(data, SeedCustomerAccountUserID)
		require.NotNil(t, recipient)
		assert.ElementsMatch(t, []string{"order_acknowledgement", "invoice"}, notifRecipientTypes(t, recipient))
	})

	t.Run("BuyerCanReadDefaults", func(t *testing.T) {
		// The customer-portal actor targets the vendor account but reads its own relation.
		status, body, err := buyer.GetListRaw(path, notifRecipientsInclude)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		data := jsonArray(parseJSON(body), "data")
		require.Len(t, data, 1, "buyer should see the seller-configured defaults")
		recipient := findNotifRecipient(data, SeedCustomerAccountUserID)
		require.NotNil(t, recipient)
		assert.ElementsMatch(t, []string{"order_acknowledgement", "invoice"}, notifRecipientTypes(t, recipient))
	})

	t.Run("BuyerSelfServiceUpdate", func(t *testing.T) {
		// Buyer narrows their own defaults to invoice-only.
		status, body, err := buyer.Patch(pathWithInclude, notifRecipientBody(map[string]any{
			"account_user_id":    SeedCustomerAccountUserID,
			"notification_types": []string{"invoice"},
		}), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		data := jsonArray(parseJSON(body), "data")
		require.Len(t, data, 1)
		recipient := findNotifRecipient(data, SeedCustomerAccountUserID)
		require.NotNil(t, recipient)
		assert.ElementsMatch(t, []string{"invoice"}, notifRecipientTypes(t, recipient))
	})

	t.Run("SellerGetReflectsBuyerUpdate", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(path, notifRecipientsInclude)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		data := jsonArray(parseJSON(body), "data")
		require.Len(t, data, 1)
		recipient := findNotifRecipient(data, SeedCustomerAccountUserID)
		require.NotNil(t, recipient)
		assert.ElementsMatch(t, []string{"invoice"}, notifRecipientTypes(t, recipient),
			"buyer's self-service change must be visible to the seller")
	})

	t.Run("SellerClearsAll", func(t *testing.T) {
		status, body, err := apiClient.Patch(path, notifRecipientBody(), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Empty(t, jsonArray(parseJSON(body), "data"), "clearing should remove all recipients")

		getStatus, getBody, err := apiClient.GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)
		assert.Empty(t, jsonArray(parseJSON(getBody), "data"))
	})
}

// TestCustomerNotificationRecipients_AccountUserExpandable proves account_user is gated behind
// ?include. Runs sequentially inside the Lifecycle's cleanup window via its own reset.
func TestCustomerNotificationRecipients_AccountUserExpandable(t *testing.T) {
	// NOT parallel: mutates the shared relation like Lifecycle. Kept as one focused test.
	path := notifRecipientsPath(SeedCustomerAccountID)
	t.Cleanup(func() {
		_, _, _ = apiClient.Patch(path, notifRecipientBody(), newIdempotencyKey())
	})

	setStatus, setBody, err := apiClient.Patch(path, notifRecipientBody(map[string]any{
		"account_user_id":    SeedCustomerAccountUserID,
		"notification_types": []string{"invoice"},
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, setStatus, setBody)

	// Without ?include, account_user is null (expandable), types still present.
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	data := jsonArray(parseJSON(body), "data")
	require.Len(t, data, 1)
	item, ok := data[0].(map[string]any)
	require.True(t, ok)
	assertNilField(t, item, "account_user")
	assert.ElementsMatch(t, []string{"invoice"}, notifRecipientTypes(t, item))

	// With ?include=account_user, it is hydrated.
	status2, body2, err := apiClient.GetListRaw(path, url.Values{"include": {"account_user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	recipient := findNotifRecipient(jsonArray(parseJSON(body2), "data"), SeedCustomerAccountUserID)
	require.NotNil(t, recipient, "account_user should be present with ?include=account_user")
}

// --- Validation (rejected before any write, so safe to run in parallel) ---

func TestCustomerNotificationRecipients_UpdateRejectsUnsupportedType(t *testing.T) {
	t.Parallel()
	// purchase_order_submission is a valid enum value but not an order (invoice/ack) type,
	// so the customer endpoint must reject it.
	status, body, err := apiClient.Patch(notifRecipientsPath(SeedCustomerAccountID), notifRecipientBody(map[string]any{
		"account_user_id":    SeedCustomerAccountUserID,
		"notification_types": []string{"purchase_order_submission"},
	}), newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"unsupported notification type should be rejected, got %d: %s", status, string(body))
}

func TestCustomerNotificationRecipients_UpdateRejectsForeignAccountUser(t *testing.T) {
	t.Parallel()
	// SeedAccountUserID belongs to the SELLER account, not the customer's account, so it
	// is not a valid recipient for this customer.
	status, body, err := apiClient.Patch(notifRecipientsPath(SeedCustomerAccountID), notifRecipientBody(map[string]any{
		"account_user_id":    SeedAccountUserID,
		"notification_types": []string{"invoice"},
	}), newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"account user from another account should be rejected, got %d: %s", status, string(body))
}

func TestCustomerNotificationRecipients_UpdateRejectsMissingTypes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(notifRecipientsPath(SeedCustomerAccountID), notifRecipientBody(map[string]any{
		"account_user_id":    SeedCustomerAccountUserID,
		"notification_types": []string{},
	}), newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"recipient with no notification types should be rejected, got %d: %s", status, string(body))
}

// --- Not found / access scoping (read-only) ---

func TestCustomerNotificationRecipients_ListUnknownCustomerNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(notifRecipientsPath("ac_000000000000000000000000"), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "unknown customer should be 404: %s", string(body))
}

func TestCustomerNotificationRecipients_BuyerCannotAccessOtherCustomer(t *testing.T) {
	t.Parallel()
	// A customer-portal actor may only read its own relationship; targeting a different
	// customer id resolves to not-found.
	status, body, err := getCustomerPortalClient().GetListRaw(
		notifRecipientsPath("ac_000000000000000000000000"), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "buyer must not read another customer's recipients: %s", string(body))
}
