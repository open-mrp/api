//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral coverage for sales-order changes on feat/hubspot:
//   - the line input no longer accepts `edi_line_item_id`,
//   - the list endpoint no longer accepts `exclude_internal_orders`,
//   - related-record metadata is still resolvable via include=related.*.

// salesOrdersPath is declared in included_fields_test.go.

func TestSalesOrders_LineRejectsRemovedEdiLineItemID(t *testing.T) {
	t.Parallel()
	body := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
	lines := body["lines"].([]map[string]any)
	// Inject the removed field on the line; unknown-field rejection fires during
	// decoding, before any business validation.
	lines[0]["edi_line_item_id"] = "edi_should_be_rejected"

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "", "invalid_request_error")
	assert.Contains(t, []any{"parameter_unknown", "validation_failed"}, errObj["code"],
		"removed line field should be rejected as unknown: %s", string(respBody))
}

func TestSalesOrders_ListRejectsRemovedExcludeInternalParam(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath, url.Values{"exclude_internal_orders": {"true"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assert.Equal(t, "exclude_internal_orders", errObj["param"],
		"the removed query param should be named in the error")
}

func TestSalesOrders_RelatedIncludesStillResolve(t *testing.T) {
	t.Parallel()
	// The related metadata moved to id strings internally; the response still
	// surfaces related.pick / related.production_run / related.shipments and the
	// include must resolve without error.
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID,
		url.Values{"include": {"related.pick", "related.production_run", "related.shipments"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	related := jsonObject(parseJSON(body), "related")
	require.NotNil(t, related, "related is present when requested via include=related.*")
	assertObjectField(t, related, "sales_order_related")

	// Sub-records, when populated, are lightweight Record objects carrying an id.
	if pick := jsonObject(related, "pick"); pick != nil {
		assert.NotEmpty(t, jsonField(pick, "id"), "related.pick is a record with an id")
	}
	if run := jsonObject(related, "production_run"); run != nil {
		assert.NotEmpty(t, jsonField(run, "id"), "related.production_run is a record with an id")
	}
	if shipments := jsonObject(related, "shipments"); shipments != nil {
		assert.Equal(t, "list", jsonField(shipments, "object"), "related.shipments is a list")
	}
}
