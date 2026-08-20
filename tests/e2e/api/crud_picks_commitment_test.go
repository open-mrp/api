//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pins the commitment a pick inherits: the pick page shows when the order must ship without
// fetching the order. Issuing an order is the only way a pick comes into existence.
func TestPicks_CarryTheOrdersCommitment(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-commit", ptrInt(12), "")
	order := issueOrderForCustomer(t, customerID, nil)
	orderShipBy := shipByDate(t, order)
	require.NotEmpty(t, orderShipBy, "the order must carry a commitment for the pick to inherit one")

	pickID := jsonField(jsonObject(jsonObject(order, "related"), "pick"), "id")
	if pickID == "" {
		status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+jsonField(order, "id"),
			url.Values{"include": {"related.pick"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		pickID = jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "pick"), "id")
	}
	require.NotEmpty(t, pickID, "issuing an order creates its pick")

	status, body, err := apiClient.GetListRaw(picksPath+"/"+pickID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	pick := parseJSON(body)

	// The commitment and the rules behind it come from the order, with no include required.
	assert.Equal(t, orderShipBy, shipByDate(t, pick), "the pick's ship-by matches its order's")
	assert.EqualValues(t, 12, pick["lead_time_days"], "lead time rides along from the order")
	assert.Equal(t, "customer", pick["lead_time_source"],
		"a customer-level lead time reports the customer as its source")
}
