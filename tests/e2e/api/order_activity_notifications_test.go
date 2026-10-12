//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sales-order activity bell alerts go only to the order's owner (its sales rep, else its creator), only
// for milestones someone else reaches, and at most once per order per day: later milestones that day
// refresh the same row without re-marking it unread.

// orderActivityNotifs returns the reader's order.updated notifications linked to the order.
func orderActivityNotifs(t *testing.T, reader *Client, orderID string) []map[string]any {
	t.Helper()
	list, _, err := reader.GetList(notificationsPath, url.Values{
		"limit":    {"100"},
		"category": {"order.updated"},
		"include":  {"resource"},
	})
	require.NoError(t, err)
	var out []map[string]any
	for _, raw := range list.Data {
		var m map[string]any
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		if res := jsonObject(m, "resource"); res != nil && jsonField(res, "id") == orderID {
			out = append(out, m)
		}
	}
	return out
}

// waitForOrderActivity polls until the reader's single notification for the order satisfies match.
func waitForOrderActivity(t *testing.T, reader *Client, orderID string, match func(map[string]any) error) map[string]any {
	t.Helper()
	var found map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		notifs := orderActivityNotifs(t, reader, orderID)
		if len(notifs) != 1 {
			return fmt.Errorf("want one notification for %s, have %d", orderID, len(notifs))
		}
		if err := match(notifs[0]); err != nil {
			return err
		}
		found = notifs[0]
		return nil
	})
	return found
}

func bodyContains(fragment string, changeCount int) func(map[string]any) error {
	return func(n map[string]any) error {
		if !strings.Contains(jsonField(n, "body"), fragment) {
			return fmt.Errorf("body %q does not mention %q yet", jsonField(n, "body"), fragment)
		}
		if got := jsonField(n, "change_count"); got != fmt.Sprint(changeCount) {
			return fmt.Errorf("change_count %s, want %d", got, changeCount)
		}
		return nil
	}
}

// createActivityOrder creates an estimate as client, owned by repAccountUserID, and unissues/deletes it on cleanup.
func createActivityOrder(t *testing.T, client *Client, repAccountUserID string) (string, string) {
	t.Helper()
	body := minimalSalesOrderCreateBody(t, setupOrderCustomer(t))
	body["sales_rep_id"] = repAccountUserID
	status, respBody, err := client.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	order := parseJSON(respBody)
	orderID := jsonField(order, "id")
	deleteOrder(t, orderID)
	return orderID, jsonField(order, "number")
}

func issueAs(t *testing.T, client *Client, orderID string) {
	t.Helper()
	status, body, err := client.Do(http.MethodPut, salesOrdersPath+"/"+orderID+"/actions/issue",
		map[string]any{"notify_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _ = salesOrderAction(t, orderID, "unissue", false) })
}

func patchOrderAs(t *testing.T, client *Client, orderID string, patch map[string]any) {
	t.Helper()
	status, body, err := client.Patch(salesOrdersPath+"/"+orderID, patch, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func addLineAs(t *testing.T, client *Client, orderID string) {
	t.Helper()
	status, body, err := client.Post(salesOrdersPath+"/"+orderID+"/lines", map[string]any{
		"product_id":  SeedProductID,
		"product_sku": uniqueName("E2E-ACTIVITY"),
		"quantity":    map[string]any{"value": "2", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
}

func TestOrderActivity_RepIsToldOfMilestonesOnceADay(t *testing.T) {
	t.Parallel()
	creator := notifUserClient(t)
	rep := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)

	orderID, number := createActivityOrder(t, creator, SeedAccountUser2ID)

	// Estimate-stage edits are not milestones.
	patchOrderAs(t, creator, orderID, map[string]any{"note": "e2e activity note"})
	addLineAs(t, creator, orderID)

	issueAs(t, creator, orderID)
	n := waitForOrderActivity(t, rep, orderID, bodyContains("issued the order", 1))
	assert.Equal(t, "Sales order "+number+" updated", jsonField(n, "title"))
	assert.Equal(t, "order.updated", jsonField(n, "category"))
	assert.Equal(t, "sales_order", jsonField(jsonObject(n, "resource"), "type"))
	id := notifID(t, n)

	mustMark(t, rep, id, "read")

	// A later milestone the same day refreshes the row without resurfacing it.
	patchOrderAs(t, creator, orderID, map[string]any{"promised_at": "2026-12-01T00:00:00Z"})
	n = waitForOrderActivity(t, rep, orderID, bodyContains("changed the promised date", 2))
	assert.Equal(t, id, jsonField(n, "id"), "same-day milestones fold onto one row")
	assert.Equal(t, "read", jsonField(n, "status"), "a refresh keeps the row read")
	assert.NotEmpty(t, jsonField(n, "read_at"))

	addLineAs(t, creator, orderID)
	n = waitForOrderActivity(t, rep, orderID, bodyContains("", 3))
	assert.Equal(t, "read", jsonField(n, "status"))

	// The person making the changes is never told about them.
	assert.Empty(t, orderActivityNotifs(t, creator, orderID))
}

func TestOrderActivity_CreatorOwnsARepLessOrder(t *testing.T) {
	t.Parallel()
	creator := notifUserClient(t)
	formerRep := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)

	orderID, _ := createActivityOrder(t, creator, SeedAccountUser2ID)
	patchOrderAs(t, creator, orderID, map[string]any{"sales_rep_id": nil})

	// An API key is someone other than the owner.
	issueAs(t, apiClient, orderID)
	waitForOrderActivity(t, creator, orderID, bodyContains("issued the order", 1))

	assert.Empty(t, orderActivityNotifs(t, formerRep, orderID), "a cleared rep no longer owns the order")
}
