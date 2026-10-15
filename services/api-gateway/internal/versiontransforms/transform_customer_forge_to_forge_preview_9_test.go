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

const forgeCustomerJSON = `{
	"id":"ac_1","object":"customer","status":"archived","standing":"hold_all","name":"Acme","number":"100",
	"relationship_type":"standalone","instructions":"Ship Tuesdays","commission_policy":"exempt","freight_policy":"free",
	"credit_limit":null,
	"contact":{"email":"a@acme.test","phone":null,"url":null},
	"carrier_billing":{"type":"third_party","account_number":"123"},
	"defaults":{
		"carrier":{"id":"car_1","object":"carrier","type":"ltl"},
		"service_level":null,
		"payment_term":{"id":"pt_1","object":"payment_term"},
		"shipping_term":null,
		"sales_rep":null,
		"bill_to_address":{"id":"ad_1","object":"address","name":"HQ","line_1":"1 Main","line_2":null,"city":"X","state":"NC","postal_code":"27601","country":"US"},
		"ship_to_address":null,
		"receive_calendar":{"id":"opcal_1","object":"operating_calendar"},
		"lead_time":{"value":"7","unit":{"id":"day","object":"unit"}},
		"supply":"make"
	},
	"group":{"id":"acgp_1","object":"account_group","type":"type_group","name":"Dist","defaults":{"lead_time":null}},
	"price_groups":null,"parent_customer":null,"child_customers":null,
	"notification_recipients":{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[{"account_user":null,"documents":["acknowledgment","invoice"]}]},
	"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`

func TestPreview9_CustomerShape(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCustomer, forgeCustomerJSON)
	assert.Equal(t, "hold_all", got["status"])
	assert.NotContains(t, got, "standing")
	assert.Equal(t, "Ship Tuesdays", got["note"])
	assert.NotContains(t, got, "instructions")
	assert.Equal(t, map[string]any{"object": "customer_contact_info", "email": "a@acme.test", "phone": nil, "url": nil}, got["contact_info"])
	assert.NotContains(t, got, "contact")
	assert.Equal(t, "commission_exempt", got["commission_policy"])

	freight := got["freight_preferences"].(map[string]any)
	assert.Equal(t, "customer_freight_preferences", freight["object"])
	assert.Equal(t, "free_freight", freight["status"])
	assert.Equal(t, "third_party", freight["billing_type"])
	assert.Equal(t, "123", freight["billing_account"])
	assert.Equal(t, "car_1", freight["carrier"].(map[string]any)["id"])
	assert.Nil(t, freight["service_level"])
	assert.NotContains(t, got, "freight_policy")
	assert.NotContains(t, got, "carrier_billing")

	defaults := got["defaults"].(map[string]any)
	assert.Equal(t, map[string]any{
		"object": "customer_defaults", "payment_term": map[string]any{"id": "pt_1", "object": "payment_term"}, "shipping_term": nil,
		"priority": nil, "sales_rep": nil, "lead_time_days": 7, "receive_calendar_id": "opcal_1", "fulfillment_policy": "make_to_order",
	}, defaults)
	assert.Equal(t, "ad_1", got["bill_to_address"].(map[string]any)["id"])
	assert.Nil(t, got["ship_to_address"])

	group := got["type"].(map[string]any)
	assert.Equal(t, "acgp_1", group["id"])
	assert.Contains(t, group, "default_lead_time_days")
	assert.Nil(t, group["default_lead_time_days"])
	assert.NotContains(t, group, "defaults")
	assert.NotContains(t, got, "group")
	assert.Contains(t, got, "parent_account")
	assert.Contains(t, got, "child_accounts")
	assert.Equal(t, map[string]any{"object": "customer_notification_preferences", "accepts_invoice_emails": true}, got["notification_preferences"])
	assert.NotContains(t, got, "notification_recipients")
}

func TestPreview9_CustomerWithoutRecipientsHasNoPreferences(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCustomer, `{"id":"ac_1","object":"customer","standing":"normal","notification_recipients":null,"defaults":{"lead_time":null,"supply":"buy"}}`)
	assert.Nil(t, got["notification_preferences"])
	defaults := got["defaults"].(map[string]any)
	assert.Nil(t, defaults["lead_time_days"])
	assert.Nil(t, defaults["fulfillment_policy"])
}

func TestPreview9_CustomerLeadTime(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCustomerLeadTime, `{"object":"customer_lead_time","customer":{"id":"ac_1","object":"customer"},
		"lead_time":{"value":"12","unit":{"id":"day","object":"unit"}},"source":"account_group","account_group":null,"parent_customer":null}`)
	assert.Equal(t, 12, got["days"])
	assert.NotContains(t, got, "lead_time")
	assert.Equal(t, map[string]any{"id": "ac_1", "object": "entity", "type": "customer", "name": nil, "handle": nil}, got["customer"])
}

func TestPreview9_CustomerCreateMovesFieldsIntoSections(t *testing.T) {
	t.Parallel()

	got := upgradeBody(constants.ObjectTypeCustomer, http.MethodPost, customersRoute, map[string]any{
		"name": "Acme", "status": "hold_shipment", "note": "n", "email": "a@acme.test", "phone": "1",
		"commission_policy": "commission_applied", "freight_policy": "billed_freight", "default_priority": "high",
		"default_carrier_id": "car_1", "default_payment_term_id": "pt_1", "default_shipping_term_id": "st_1",
		"lead_time_days": float64(3), "fulfillment_policy": "make_to_stock", "receive_calendar_id": "opcal_1",
		"carrier_billing_type": "sender", "customer_type_group_id": "acgp_1", "customer_price_group_ids": []any{"acgp_2"},
	})
	assert.Equal(t, map[string]any{
		"name": "Acme", "standing": "hold_shipment", "instructions": "n",
		"commission_policy": "applied", "freight_policy": "billed",
		"contact":         map[string]any{"email": "a@acme.test", "phone": "1"},
		"carrier_billing": map[string]any{"type": "sender"},
		"defaults": map[string]any{
			"carrier_id": "car_1", "payment_term_id": "pt_1", "shipping_term_id": "st_1", "receive_calendar_id": "opcal_1",
			"lead_time": map[string]any{"value": "3", "unit_id": "day"}, "supply": "stock",
		},
		"group_id": "acgp_1", "price_group_ids": []any{"acgp_2"},
	}, got)
}

func TestPreview9_CustomerUpdateClearsThroughSections(t *testing.T) {
	t.Parallel()

	got := upgradeBody(constants.ObjectTypeCustomer, http.MethodPatch, customerRoute, map[string]any{
		"lead_time_days": nil, "fulfillment_policy": nil, "default_service_level_id": nil, "bill_to_address_id": "ad_2",
		"carrier_billing_account": nil, "url": nil,
	})
	assert.Equal(t, map[string]any{
		"defaults":        map[string]any{"lead_time": nil, "supply": nil, "service_level_id": nil, "bill_to_address_id": "ad_2"},
		"carrier_billing": map[string]any{"account_number": nil},
		"contact":         map[string]any{"url": nil},
	}, got)
}

func TestPreview9_AccountGroupLeadTime(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeAccountGroup, `{"id":"acgp_1","object":"account_group","defaults":{"lead_time":{"value":"4","unit":{"id":"day","object":"unit"}}}}`)
	assert.Equal(t, 4, got["default_lead_time_days"])
	assert.NotContains(t, got, "defaults")

	body := upgradeBody(constants.ObjectTypeAccountGroup, http.MethodPatch, accountGroupRoute, map[string]any{"default_lead_time_days": nil})
	assert.Equal(t, map[string]any{"defaults": map[string]any{"lead_time": nil}}, body)
	body = upgradeBody(constants.ObjectTypeAccountGroup, http.MethodPost, accountGroupsRoute, map[string]any{"name": "G", "default_lead_time_days": float64(2)})
	assert.Equal(t, map[string]any{"name": "G", "defaults": map[string]any{"lead_time": map[string]any{"value": "2", "unit_id": "day"}}}, body)
}

func TestPreview9_CustomerListFiltersAndIncludes(t *testing.T) {
	RegisterRouteIncludes(customersRoute, []string{"group", "parent_customer", "defaults.carrier", "defaults.bill_to_address", "notification_recipients"})

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeCustomer, customersRoute, url.Values{
		"status_codes[]":        {"preferred", "hold_all"},
		"customer_group_ids":    {"acgp_1"},
		"parent_account_status": {"non_parent"},
		"starts_at":             {"2026-01-01T00:00:00Z"},
		"include[]":             {"type", "freight_preferences.carrier", "freight_preferences", "contact_info", "bill_to_address", "notification_preferences"},
	})
	assert.Equal(t, []string{"hold_all"}, q["standings[]"])
	assert.Equal(t, []string{"acgp_1"}, q["group_ids"])
	assert.Equal(t, []string{"standalone", "child"}, q["relationship_types[]"])
	assert.Equal(t, []string{"2026-01-01T00:00:00Z"}, q["created_after"])
	require.NotContains(t, q, "status_codes[]")
	assert.Equal(t, []string{"group", "defaults.carrier", "defaults.bill_to_address", "notification_recipients"}, q["include[]"])
}
