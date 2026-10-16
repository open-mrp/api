package versiontransforms

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

const forgeSalesOrderJSON = `{
	"id":"or_1","object":"sales_order","number":"SO-1","status":"draft","currency":"usd",
	"customer_purchase_order_number":"PO-1","instructions":"Pack carefully","payment_status":"unpaid",
	"acknowledgment_status":"not_sent","customer":null,"site":null,
	"sales_rep":{"id":"acus_1","object":"account_user","user":{"id":"us_1","object":"user","name":"Rep","email":"rep@x.test","image_url":null}},
	"created_by":null,
	"bill_to":{"name":"HQ","line_1":"1 Main","line_2":null,"city":"X","state":"NC","postal_code":"27601","country":"US","phone":null,"email":null,"address":null},
	"ship_to":null,
	"bill_to_address":{"id":"ad_1","object":"address"},"ship_to_address":null,"contacts":null,
	"freight":{"carrier":{"id":"car_1","object":"carrier"},"service_level":null,"carrier_billing":{"type":"third_party","account_number":"123"}},
	"payment_term":null,"shipping_term":null,"order_discount":null,"email_recipients":null,
	"commitment":{"promised_on":null,"lead_time_override":{"value":"5","unit":{"id":"day","object":"unit"}},"ship_by_override_on":null,
		"firmness":"hard","ship_by_on":"2026-08-20","promised_lead_time":{"value":"5","unit":{"id":"day","object":"unit"}},
		"promised_lead_time_source":"override","lead_time_source":"order_lead_time","transit_days":null,"transit_source":null,
		"calendar_adjustment_days":0,"ship_by_cutoff_at":null},
	"lines":null,"line_count":1,"totals":null,
	"related":{"object":"sales_order_related","pick":null,"jobs":{"object":"list","data":[{"id":"pr_1","object":"record"}]},"shipments":null,"invoices":null},
	"issued_at":"2026-08-13T15:04:05Z","fulfilled_at":null,"first_shipped_at":null,"expires_on":"2026-09-01",
	"metadata":{},"created_at":"2026-08-01T00:00:00Z","updated_at":"2026-08-01T00:00:00Z"}`

func TestPreview9_SalesOrderShape(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeSalesOrder, forgeSalesOrderJSON)
	assert.Equal(t, "estimate", got["status"])
	assert.Equal(t, "Pack carefully", got["note"])
	assert.Nil(t, got["priority"])
	assert.Equal(t, []any{}, got["payment_intent_ids"])
	for _, k := range []string{"instructions", "currency", "site", "bill_to", "ship_to", "email_recipients", "expires_on", "fulfilled_at", "first_shipped_at"} {
		assert.NotContains(t, got, k)
	}
	assert.Equal(t, map[string]any{"id": "acus_1", "object": "actor", "type": "user", "name": "Rep", "handle": "rep@x.test", "avatar_url": nil, "role": nil}, got["sales_rep"])
	assert.Equal(t, map[string]any{
		"object": "freight", "policy": nil, "carrier": got["freight"].(map[string]any)["carrier"], "service_level": nil,
		"billing_type": "third_party", "billing_account_number": "123",
	}, got["freight"])
	assert.Equal(t, "ad_1", got["bill_to_address"].(map[string]any)["id"])
	assert.Equal(t, "2026-09-01T00:00:00Z", got["expired_at"])
	assert.Contains(t, got, "completed_at")
	assert.Contains(t, got, "first_ship_at")

	c := got["commitment"].(map[string]any)
	assert.Equal(t, "commitment", c["object"])
	assert.Equal(t, 5, c["lead_time_override_days"])
	assert.Equal(t, "2026-08-20T00:00:00Z", c["ship_by_date"])
	assert.Equal(t, 7, c["lead_time_days"])
	assert.Equal(t, "order_lead_time", c["lead_time_source"])
	assert.Nil(t, c["promised_at"])
	assert.NotContains(t, c, "firmness")
	assert.NotContains(t, c, "promised_lead_time")

	related := got["related"].(map[string]any)
	assert.Equal(t, "pr_1", related["production_run"].(map[string]any)["id"])
	assert.NotContains(t, related, "jobs")
}

func TestPreview9_SalesOrderCanceledReadsFulfilled(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeSalesOrder, `{"id":"or_1","object":"sales_order","status":"canceled","currency":"usd","freight":{},
		"commitment":{"ship_by_on":null,"ship_by_cutoff_at":"2026-08-20T19:00:00Z"},"sales_rep":null,"related":null}`)
	assert.Equal(t, "fulfilled", got["status"])
	assert.Nil(t, got["sales_rep"])
	assert.Equal(t, "2026-08-20T19:00:00Z", got["commitment"].(map[string]any)["ship_by_date"])
	assert.Nil(t, got["freight"].(map[string]any)["billing_type"])
}

func TestPreview9_SalesOrderNestedUnderAnotherResource(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeInvoice, `{"id":"inv_1","object":"invoice","order":{"id":"or_1","object":"sales_order","status":"issued","currency":"usd","freight":{},"commitment":null}}`)
	order := got["order"].(map[string]any)
	assert.Equal(t, "issued", order["status"])
	assert.NotContains(t, order, "currency")
	assert.Nil(t, order["commitment"])
}

func TestPreview9_SalesOrderCreateMovesFieldsIntoSections(t *testing.T) {
	t.Parallel()

	got := upgradeBody(constants.ObjectTypeSalesOrder, http.MethodPost, salesOrdersRoute, map[string]any{
		"buyer_account_id": "ac_1", "note": "n", "priority_code": "normal",
		"carrier_id": "car_1", "service_level_id": "sl_1", "carrier_billing_type": "third_party", "carrier_billing_account_number": "9",
		"promised_at": "2026-05-20T00:00:00Z", "bill_to_address_id": "ad_1",
		"ship_to_address":                map[string]any{"id": "ad_2", "name": "Dock", "country": "US", "is_drop_ship": true, "line_2": nil},
		"acknowledgement_email_contacts": []any{map[string]any{"account_user_id": "acus_1"}},
		"invoice_email_contacts":         []any{map[string]any{"account_user_id": "acus_1"}, map[string]any{"account_user_id": "acus_2"}},
		"lines":                          []any{},
	})
	assert.Equal(t, map[string]any{
		"customer_id": "ac_1", "instructions": "n",
		"freight": map[string]any{
			"carrier_id": "car_1", "service_level_id": "sl_1",
			"carrier_billing": map[string]any{"type": "third_party", "account_number": "9"},
		},
		"commitment": map[string]any{"promised_on": "2026-05-20"},
		"bill_to":    map[string]any{"address_id": "ad_1"},
		"ship_to":    map[string]any{"name": "Dock", "country": "US"},
		"email_recipients": []any{
			map[string]any{"account_user_id": "acus_1", "documents": []any{"acknowledgment", "invoice"}},
			map[string]any{"account_user_id": "acus_2", "documents": []any{"invoice"}},
		},
		"lines": []any{},
	}, got)
}

func TestPreview9_SalesOrderUpdateClearsThroughSections(t *testing.T) {
	t.Parallel()

	got := upgradeBody(constants.ObjectTypeSalesOrder, http.MethodPatch, salesOrderRoute, map[string]any{
		"note": nil, "carrier_billing_type": nil, "lead_time_override_days": float64(4), "ship_by_override_date": nil,
		"shipping_address_id": "ad_3", "invoice_email_contacts": []any{},
	})
	assert.Equal(t, map[string]any{
		"instructions": nil,
		"freight":      map[string]any{"carrier_billing": nil},
		"commitment":   map[string]any{"lead_time_override": map[string]any{"value": "4", "unit_id": "day"}, "ship_by_override_on": nil},
		"ship_to":      map[string]any{"address_id": "ad_3"},
		// One kind of recipient is replaced alone, as preview.9 did.
		"invoice_email_contacts": []any{},
	}, got)
}

func TestPreview9_SalesOrderListFiltersAndIncludes(t *testing.T) {
	RegisterRouteIncludes(salesOrdersRoute, []string{"freight.carrier", "freight.service_level", "related.jobs", "bill_to_address", "contacts"})

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeSalesOrder, salesOrdersRoute, url.Values{
		"status_codes[]":     {"estimate", "fulfilled", "all"},
		"customer_group_ids": {"acgp_1"},
		"starts_at":          {"2026-01-01"},
		"include[]":          {"freight", "related.production_run", "bill_to_address", "contacts"},
	})
	assert.Equal(t, []string{"draft", "fulfilled"}, q["statuses[]"])
	assert.Equal(t, []string{"acgp_1"}, q["group_ids"])
	assert.Equal(t, []string{"2026-01-01"}, q["created_after"])
	require.NotContains(t, q, "status_codes[]")
	assert.Equal(t, []string{"freight.carrier", "freight.service_level", "related.jobs", "bill_to_address", "contacts"}, q["include[]"])
}
