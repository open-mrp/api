//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// Collects the resource types recorded against one root, which is how the order timeline is read.
func rootScopedResourceTypes(t *testing.T, orderID string) map[string]bool {
	t.Helper()

	list, status, err := apiClient.GetList(auditEventsPath, url.Values{
		"root_resource_type": {"sales_order"},
		"root_resource_id":   {orderID},
		"limit":              {"100"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status, "root-scoped audit query should return 200")

	seen := map[string]bool{}
	for _, raw := range list.Data {
		var ev map[string]any
		require.NoError(t, json.Unmarshal(raw, &ev))
		if rt, ok := ev["resource_type"].(string); ok {
			seen[rt] = true
		}
	}
	return seen
}

// Pins the rollup: issuing an order must put the pick and its lines into the order's own history.
// A descendant that never stamps its root is invisible there and no join can recover it.
func TestAuditRollup_IssueRecordsThePickUnderTheOrder(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-audit-rollup", ptrInt(5), "")
	order := issueOrderForCustomer(t, customerID, nil)
	orderID := jsonField(order, "id")
	require.NotEmpty(t, orderID)

	// Audit events ride the outbox, so they land after the request returns.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		seen := rootScopedResourceTypes(t, orderID)
		for _, want := range []string{"sales_order", "pick", "pick_line"} {
			if !seen[want] {
				return fmt.Errorf("%s missing from the order's rolled-up history (saw %v)", want, seen)
			}
		}
		return nil
	})
}
