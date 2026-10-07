//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dashCustomersBulkDeletePath   = customersPath + "/actions/bulk-delete"
	dashCustomersRegistrationPath = customersPath + "/registration"
	dashCustomersZeroTime         = "0001-01-01T00:00:00Z"
)

func dashCustomersMergePath(targetID string) string {
	return customersPath + "/" + targetID + "/actions/merge"
}

// dashCustomersNew creates a customer from validCustomerBody plus extra and returns its id.
func dashCustomersNew(t *testing.T, prefix string, extra map[string]any) string {
	t.Helper()
	body := validCustomerBody(uniqueName(prefix))
	for k, v := range extra {
		body[k] = v
	}
	return jsonField(createAndCleanup(t, customersPath, body), "id")
}

// dashCustomersRead reads a customer with the given comma-separated includes.
func dashCustomersRead(t *testing.T, client *Client, customerID, include string) (int, map[string]any) {
	t.Helper()
	params := url.Values{}
	if include != "" {
		params.Set("include", include)
	}
	status, body, err := client.GetListRaw(customersPath+"/"+customerID, params)
	require.NoError(t, err)
	require.Less(t, status, 500, "reading a customer must not 5xx: %s", body)
	return status, parseJSON(body)
}

func dashCustomersRequireExists(t *testing.T, customerID string) map[string]any {
	t.Helper()
	status, got := dashCustomersRead(t, apiClient, customerID, "")
	require.Equal(t, 200, status, "customer %s must still exist", customerID)
	return got
}

// dashCustomersRefused asserts a request naming a reference the account does not have was refused
// as a client error naming the field.
func dashCustomersRefused(t *testing.T, status int, body []byte, param string) {
	t.Helper()
	require.Less(t, status, 500, "a bad reference must not 5xx: %s", body)
	assert.Contains(t, []int{400, 404}, status, "a reference the account does not have is refused: %s", body)
	assert.True(t, strings.HasPrefix(errorParam(body), param), "the error names %s: %s", param, body)
}

// dashCustomersNoRetry is client minus the transient-5xx retries, for requests whose 5xx is the bug
// under test.
func dashCustomersNoRetry(client *Client) *Client {
	c := *client
	c.retries = 0
	return &c
}

// dashCustomersTenantBGroup is an account group another tenant owns.
func dashCustomersTenantBGroup(t *testing.T, groupType string) string {
	t.Helper()
	other := getTenantBClient()
	status, body, err := other.Post(accountGroupsPath, map[string]any{"name": uniqueName("e2e-dc-b-grp"), "type": groupType}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	groupID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = other.Delete(accountGroupsPath + "/" + groupID) })
	return groupID
}

// dashCustomersNewPerson registers a person with no account and returns their session. The dashboard
// calls registration before the person belongs to any account, so no OpenMRP-Account is sent.
func dashCustomersNewPerson(t *testing.T) (client *Client, email string) {
	t.Helper()
	_, email = covAuthPasswordsRegisterUser(t, "e2e-dc-person")
	return loginAsUser(t, email, covAuthUsersPassword, ""), email
}

// dashCustomersVendorAccounts lists the customer accounts a person belongs to under SeedAccountID.
func dashCustomersVendorAccounts(t *testing.T, person *Client) []string {
	t.Helper()
	status, body, err := person.GetListRaw("/v1/identity/me/tenancy/customer-accounts/"+SeedAccountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var ids []string
	for _, raw := range jsonArray(parseJSON(body), "data") {
		ids = append(ids, jsonField(raw.(map[string]any), "id"))
	}
	return ids
}

func dashCustomersCustomerIDsNamed(t *testing.T, name string) []string {
	t.Helper()
	list, status, err := apiClient.GetList(customersPath, url.Values{"q": {name}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	var ids []string
	for _, raw := range list.Data {
		ids = append(ids, DataItemField(raw, "id"))
	}
	return ids
}

// ---------------------------------------------------------------------------
// Customers — create and update
// ---------------------------------------------------------------------------

// The customer number is how the portal finds an existing customer, so two customers of one
// account may never share one, whether set on create or on edit.
func TestDashCustomers_NumberIsUniqueWithinTheAccount(t *testing.T) {
	t.Parallel()
	number := searchToken("e2edcnum")
	first := dashCustomersNew(t, "e2e-dc-num-a", map[string]any{"number": number})
	assert.Equal(t, number, jsonField(dashCustomersRequireExists(t, first), "number"))

	duplicate := validCustomerBody(uniqueName("e2e-dc-num-b"))
	duplicate["number"] = number
	status, body, err := apiClient.Post(customersPath, duplicate, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		apiClient.Delete(customersPath + "/" + jsonField(parseJSON(body), "id"))
	}
	requireStatus(t, 409, status, body)
	assert.Equal(t, "number", errorParam(body))

	other := dashCustomersNew(t, "e2e-dc-num-c", nil)
	otherNumber := jsonField(dashCustomersRequireExists(t, other), "number")
	status, body, err = apiClient.Patch(customersPath+"/"+other, map[string]any{"number": number}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assert.Equal(t, "number", errorParam(body))
	assert.Equal(t, otherNumber, jsonField(dashCustomersRequireExists(t, other), "number"), "the refused edit changed nothing")

	// Re-saving a customer's own number is not a conflict with itself.
	status, body, err = apiClient.Patch(customersPath+"/"+first, map[string]any{"number": number}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// The customer form clears each optional default by sending null; a cleared default must read back
// as null and stay cleared, while the defaults that were not sent are kept.
func TestDashCustomers_UpdateClearsEveryClearableDefault(t *testing.T) {
	t.Parallel()
	calendarID := createCalendar(t, "receive", "1111100", nil)
	customerID := dashCustomersNew(t, "e2e-dc-clear", map[string]any{
		"lead_time_days":           9,
		"fulfillment_policy":       "make_to_order",
		"default_service_level_id": SeedServiceLevelID,
		"carrier_billing_type":     "third_party",
		"carrier_billing_account":  "DC-ACCT-1",
	})
	asCustomer := apiClient.WithAccountID(customerID)
	status, body, err := asCustomer.Post(addressesPath, map[string]any{"name": uniqueName("e2e-dc-clear-dock"), "country": "US"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	dockID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = asCustomer.Delete(addressesPath + "/" + dockID) })

	const include = "defaults,defaults.payment_term,freight_preferences,freight_preferences.service_level,bill_to_address,ship_to_address"
	status, body, err = apiClient.Patch(customersPath+"/"+customerID, map[string]any{
		"bill_to_address_id":  dockID,
		"ship_to_address_id":  dockID,
		"receive_calendar_id": calendarID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	_, set := dashCustomersRead(t, apiClient, customerID, include)
	defaults := jsonObject(set, "defaults")
	freight := jsonObject(set, "freight_preferences")
	require.NotNil(t, defaults)
	require.NotNil(t, freight)
	assert.Equal(t, "9", jsonField(defaults, "lead_time_days"))
	assert.Equal(t, calendarID, jsonField(defaults, "receive_calendar_id"))
	assert.Equal(t, "make_to_order", jsonField(defaults, "fulfillment_policy"))
	assert.Equal(t, SeedServiceLevelID, jsonField(jsonObject(freight, "service_level"), "id"))
	assert.Equal(t, "DC-ACCT-1", jsonField(freight, "billing_account"))
	assert.Equal(t, dockID, jsonField(jsonObject(set, "bill_to_address"), "id"))
	assert.Equal(t, dockID, jsonField(jsonObject(set, "ship_to_address"), "id"))

	status, body, err = apiClient.Patch(customersPath+"/"+customerID+"?include="+include, map[string]any{
		"lead_time_days":           nil,
		"receive_calendar_id":      nil,
		"fulfillment_policy":       nil,
		"default_service_level_id": nil,
		"carrier_billing_account":  nil,
		"bill_to_address_id":       nil,
		"ship_to_address_id":       nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	_, reread := dashCustomersRead(t, apiClient, customerID, include)

	for label, got := range map[string]map[string]any{"the response": parseJSON(body), "a later read": reread} {
		defaults := jsonObject(got, "defaults")
		freight := jsonObject(got, "freight_preferences")
		require.NotNil(t, defaults, label)
		require.NotNil(t, freight, label)
		assert.Nil(t, defaults["lead_time_days"], label)
		assert.Nil(t, defaults["receive_calendar_id"], label)
		assert.Nil(t, defaults["fulfillment_policy"], label)
		assert.Nil(t, freight["service_level"], label)
		assert.Nil(t, freight["billing_account"], label)
		assert.Nil(t, got["bill_to_address"], label)
		assert.Nil(t, got["ship_to_address"], label)
		assert.Equal(t, "third_party", jsonField(freight, "billing_type"), "%s keeps the billing type that was not sent", label)
		assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(defaults, "payment_term"), "id"), "%s keeps the payment term", label)
	}
}

// The receive calendar is a default like the lead time: an edit that does not send it keeps it.
func TestDashCustomers_AnEditKeepsTheReceiveCalendar(t *testing.T) {
	t.Parallel()
	calendarID := createCalendar(t, "receive", "1111100", nil)
	customerID := dashCustomersNew(t, "e2e-dc-cal", map[string]any{"receive_calendar_id": calendarID, "lead_time_days": 4})
	// The calendar refuses deletion while referenced, and it is cleaned up after the customer.
	t.Cleanup(func() {
		_, _, _ = apiClient.Patch(customersPath+"/"+customerID, map[string]any{"receive_calendar_id": nil}, newIdempotencyKey())
	})
	_, created := dashCustomersRead(t, apiClient, customerID, "defaults")
	require.Equal(t, calendarID, jsonField(jsonObject(created, "defaults"), "receive_calendar_id"))

	status, body, err := apiClient.Patch(customersPath+"/"+customerID+"?include=defaults", map[string]any{"note": "e2e unrelated edit"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, calendarID, jsonField(jsonObject(parseJSON(body), "defaults"), "receive_calendar_id"), "the edit response keeps the calendar")
	_, got := dashCustomersRead(t, apiClient, customerID, "defaults")
	defaults := jsonObject(got, "defaults")
	assert.Equal(t, calendarID, jsonField(defaults, "receive_calendar_id"), "a later read keeps the calendar")
	assert.Equal(t, "4", jsonField(defaults, "lead_time_days"))
}

// Fields that are required on a customer cannot be nulled or blanked by an edit, and enum and range
// checks hold on edit as on create.
func TestDashCustomers_UpdateRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	customerID := dashCustomersNew(t, "e2e-dc-badupd", nil)
	before := dashCustomersRequireExists(t, customerID)

	cases := []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"null name", map[string]any{"name": nil}, "name"},
		{"blank name", map[string]any{"name": ""}, "name"},
		{"null number", map[string]any{"number": nil}, "number"},
		{"blank number", map[string]any{"number": ""}, "number"},
		{"null status", map[string]any{"status": nil}, "status"},
		{"unknown status", map[string]any{"status": "bogus"}, "status"},
		{"unknown priority", map[string]any{"default_priority": "bogus"}, "default_priority"},
		{"unknown fulfillment policy", map[string]any{"fulfillment_policy": "bogus"}, "fulfillment_policy"},
		{"unknown carrier billing type", map[string]any{"carrier_billing_type": "bogus"}, "carrier_billing_type"},
		{"null carrier", map[string]any{"default_carrier_id": nil}, "default_carrier_id"},
		{"null payment term", map[string]any{"default_payment_term_id": nil}, "default_payment_term_id"},
		{"null shipping term", map[string]any{"default_shipping_term_id": nil}, "default_shipping_term_id"},
		{"null type group", map[string]any{"customer_type_group_id": nil}, "customer_type_group_id"},
		{"null price groups", map[string]any{"customer_price_group_ids": nil}, "customer_price_group_ids"},
		{"negative lead time", map[string]any{"lead_time_days": -1}, "lead_time_days"},
		{"lead time over ten years", map[string]any{"lead_time_days": 3651}, "lead_time_days"},
		{"lead time as text", map[string]any{"lead_time_days": "7"}, "lead_time_days"},
		{"email over 255 characters", map[string]any{"email": strings.Repeat("a", 256)}, "email"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Patch(customersPath+"/"+customerID, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			assert.Equal(t, tc.param, errorParam(body))
		})
	}

	after := dashCustomersRequireExists(t, customerID)
	assert.Equal(t, jsonField(before, "name"), jsonField(after, "name"), "the refused edits changed nothing")
	assert.Equal(t, jsonField(before, "number"), jsonField(after, "number"))
	assert.Equal(t, jsonField(before, "status"), jsonField(after, "status"))
}

// The customer table holds bare ids with no foreign keys, so an edit naming a carrier, term, group,
// calendar or unit the account does not have, including another tenant's, has to be refused as a
// client error, and the customer stays readable with its references as they were.
func TestDashCustomers_UpdateRefusesReferencesTheAccountDoesNotHave(t *testing.T) {
	t.Parallel()
	foreignType := dashCustomersTenantBGroup(t, "type_group")
	foreignPricing := dashCustomersTenantBGroup(t, "pricing_group")

	cases := []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"unknown carrier", map[string]any{"default_carrier_id": mustGenID(t, id.CarrierIDPrefix)}, "default_carrier_id"},
		{"unknown payment term", map[string]any{"default_payment_term_id": mustGenID(t, id.PaymentTermIDPrefix)}, "default_payment_term_id"},
		{"unknown shipping term", map[string]any{"default_shipping_term_id": mustGenID(t, id.ShippingTermIDPrefix)}, "default_shipping_term_id"},
		{"unknown service level", map[string]any{"default_service_level_id": mustGenID(t, id.ServiceLevelIDPrefix)}, "default_service_level_id"},
		{"unknown receive calendar", map[string]any{"receive_calendar_id": mustGenID(t, id.OperatingCalendarIDPrefix)}, "receive_calendar_id"},
		{"unknown credit limit unit", map[string]any{"credit_limit": map[string]any{"value": "10", "unit_id": mustGenID(t, id.UnitIDPrefix)}}, "credit_limit"},
		{"another tenant's type group", map[string]any{"customer_type_group_id": foreignType}, "customer_type_group_id"},
		{"another tenant's pricing group", map[string]any{"customer_price_group_ids": []string{foreignPricing}}, "customer_price_group_ids"},
		{"another tenant's sales rep", map[string]any{"default_sales_rep_id": SeedTenantBAccountUserID}, "default_sales_rep_id"},
	}
	client := dashCustomersNoRetry(apiClient)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			customerID := dashCustomersNew(t, "e2e-dc-badref", nil)
			status, body, err := client.Patch(customersPath+"/"+customerID, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			dashCustomersRefused(t, status, body, tc.param)

			status, got := dashCustomersRead(t, client, customerID, "type,price_groups,defaults.payment_term,freight_preferences.carrier,credit_limit")
			require.Equal(t, 200, status, "the customer is still readable")
			assert.Equal(t, SeedCustomerGroupID, jsonField(jsonObject(got, "type"), "id"), "the type group is unchanged")
			assert.Empty(t, jsonArray(jsonObject(got, "price_groups"), "data"), "no price group was linked")
			assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(jsonObject(got, "defaults"), "payment_term"), "id"))
			assert.Equal(t, SeedCarrierID, jsonField(jsonObject(jsonObject(got, "freight_preferences"), "carrier"), "id"))
			assertNilField(t, got, "credit_limit")
		})
	}
}

// dashCustomersDeleteNamed deletes ownerAccountID's customers named name. A create that fails with a
// 5xx can still have committed, and its response carries no id to clean up by.
func dashCustomersDeleteNamed(t *testing.T, client *Client, ownerAccountID, name string) {
	t.Helper()
	rows, err := authDB(t).Query("SELECT counterparty_account_id FROM account_relation WHERE owner_account_id = ? AND alias = ?", ownerAccountID, name)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var customerID string
		require.NoError(t, rows.Scan(&customerID))
		_, _, _ = client.Delete(customersPath + "/" + customerID)
	}
}

// The same references are refused on create, and another tenant cannot build a customer out of this
// account's payment term and carrier.
func TestDashCustomers_CreateRefusesReferencesTheAccountDoesNotHave(t *testing.T) {
	t.Parallel()
	foreignType := dashCustomersTenantBGroup(t, "type_group")
	foreignPricing := dashCustomersTenantBGroup(t, "pricing_group")
	other := getTenantBClient()

	cases := []struct {
		name   string
		client *Client
		owner  string
		extra  map[string]any
		param  string
	}{
		{"unknown carrier", apiClient, SeedAccountID, map[string]any{"default_carrier_id": mustGenID(t, id.CarrierIDPrefix)}, "default_carrier_id"},
		{"unknown payment term", apiClient, SeedAccountID, map[string]any{"default_payment_term_id": mustGenID(t, id.PaymentTermIDPrefix)}, "default_payment_term_id"},
		{"unknown shipping term", apiClient, SeedAccountID, map[string]any{"default_shipping_term_id": mustGenID(t, id.ShippingTermIDPrefix)}, "default_shipping_term_id"},
		{"unknown service level", apiClient, SeedAccountID, map[string]any{"default_service_level_id": mustGenID(t, id.ServiceLevelIDPrefix)}, "default_service_level_id"},
		{"unknown receive calendar", apiClient, SeedAccountID, map[string]any{"receive_calendar_id": mustGenID(t, id.OperatingCalendarIDPrefix)}, "receive_calendar_id"},
		{"unknown credit limit unit", apiClient, SeedAccountID, map[string]any{"credit_limit": map[string]any{"value": "10", "unit_id": mustGenID(t, id.UnitIDPrefix)}}, "credit_limit"},
		{"another tenant's type group", apiClient, SeedAccountID, map[string]any{"customer_type_group_id": foreignType}, "customer_type_group_id"},
		{"another tenant's pricing group", apiClient, SeedAccountID, map[string]any{"customer_price_group_ids": []string{foreignPricing}}, "customer_price_group_ids"},
		{"another tenant's sales rep", apiClient, SeedAccountID, map[string]any{"default_sales_rep_id": SeedTenantBAccountUserID}, "default_sales_rep_id"},
		// Tenant B names its own type group, but validCustomerBody's payment term and carrier are SeedAccountID's.
		{"this account's terms used by another tenant", other, SeedTenantBAccountID, map[string]any{"customer_type_group_id": foreignType}, "default_"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name := uniqueName("e2e-dc-badref-new")
			body := validCustomerBody(name)
			for k, v := range tc.extra {
				body[k] = v
			}
			status, resp, err := dashCustomersNoRetry(tc.client).Post(customersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			dashCustomersDeleteNamed(t, tc.client, tc.owner, name)
			dashCustomersRefused(t, status, resp, tc.param)
		})
	}
}

// ---------------------------------------------------------------------------
// Customers — who may act
// ---------------------------------------------------------------------------

// A role that may look at customers but not change them is refused every customer write the
// customer pages offer, and nothing changes.
func TestDashCustomers_AReadOnlyRoleCannotChangeCustomers(t *testing.T) {
	t.Parallel()
	customerID := dashCustomersNew(t, "e2e-dc-ro", nil)
	sourceID := dashCustomersNew(t, "e2e-dc-ro-src", nil)
	readOnly := customRoleClient(t, "customers:read")

	status, _ := dashCustomersRead(t, readOnly, customerID, "")
	require.Equal(t, 200, status, "customers:read may read the customer")

	writes := map[string]func() (int, []byte, error){
		"edit": func() (int, []byte, error) {
			return readOnly.Patch(customersPath+"/"+customerID, map[string]any{"note": "e2e read-only edit"}, newIdempotencyKey())
		},
		"merge": func() (int, []byte, error) {
			return readOnly.Post(dashCustomersMergePath(customerID), map[string]any{"source_customer_ids": []string{sourceID}}, newIdempotencyKey())
		},
		"notification recipients": func() (int, []byte, error) {
			return readOnly.Patch(notifRecipientsPath(customerID), notifRecipientBody(), newIdempotencyKey())
		},
		"link a child account": func() (int, []byte, error) {
			return readOnly.WithAccountID(customerID).Put(childAccountsPath+"/"+sourceID, nil)
		},
		"add an address to the customer": func() (int, []byte, error) {
			return readOnly.WithAccountID(customerID).Post(addressesPath, map[string]any{"name": uniqueName("e2e-dc-ro-dock"), "country": "US"}, newIdempotencyKey())
		},
	}
	for name, do := range writes {
		status, body, err := do()
		require.NoError(t, err)
		assert.Equal(t, 403, status, "%s: %s", name, body)
	}

	// Reading the customer's addresses and children needs only customers:read.
	status, body, err := readOnly.WithAccountID(customerID).GetListRaw(addressesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = readOnly.WithAccountID(customerID).GetListRaw(childAccountsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assertNilField(t, dashCustomersRequireExists(t, customerID), "note")
	dashCustomersRequireExists(t, sourceID)
	children, _, err := apiClient.WithAccountID(customerID).GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.Empty(t, children.Data, "the refused link changed nothing")
}

// The reference documents merging as needing customers:update and customers:delete together, not either.
func TestDashCustomers_MergeDocumentsBothPermissions(t *testing.T) {
	t.Parallel()
	for _, op := range loadDashboardOperations(t) {
		if op.operationID == "merge-customers" {
			assert.ElementsMatch(t, []string{"customers:update", "customers:delete"}, op.permissions)
			assert.True(t, op.allOf, "the permissions are documented as all-of")
			return
		}
	}
	t.Fatal("merge-customers is not in the spec")
}

// Merging deletes the sources, so the service takes customers:update and customers:delete together.
func TestDashCustomers_MergeNeedsUpdateAndDeleteTogether(t *testing.T) {
	t.Parallel()
	targetID := dashCustomersNew(t, "e2e-dc-mrgperm", nil)
	sourceID := dashCustomersNew(t, "e2e-dc-mrgperm-src", nil)

	for missing, perms := range map[string][]string{
		"customers:delete": {"customers:read", "customers:update"},
		"customers:update": {"customers:read", "customers:delete"},
	} {
		status, body, err := customRoleClient(t, perms...).Post(dashCustomersMergePath(targetID),
			map[string]any{"source_customer_ids": []string{sourceID}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
		apiErr := jsonObject(parseJSON(body), "error")
		assert.Contains(t, jsonField(apiErr, "message"), missing, "the refusal names the permission the role lacks")
		dashCustomersRequireExists(t, sourceID)
	}

	status, body, err := customRoleClient(t, "customers:read", "customers:update", "customers:delete").Post(dashCustomersMergePath(targetID),
		map[string]any{"source_customer_ids": []string{sourceID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, _ = dashCustomersRead(t, apiClient, sourceID, "")
	assert.Equal(t, 404, status, "the merged source is gone")
}

// The customer portal reads its own customer record and nothing else, and changes no customer.
func TestDashCustomers_PortalActorReachesOnlyItsOwnCustomer(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()
	otherID := dashCustomersNew(t, "e2e-dc-portal-other", nil)

	status, own := dashCustomersRead(t, portal, SeedCustomerAccountID, "")
	require.Equal(t, 200, status)
	assert.Equal(t, SeedCustomerAccountID, jsonField(own, "id"))

	for name, path := range map[string]string{
		"another customer":                         customersPath + "/" + otherID,
		"another customer's frequent products":     customersPath + "/" + otherID + "/frequently-ordered-products",
		"another customer's notification contacts": notifRecipientsPath(otherID),
	} {
		status, body, err := portal.GetListRaw(path, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status, "%s: %s", name, body)
	}

	refused := map[string]func() (int, []byte, error){
		"list": func() (int, []byte, error) { return portal.GetListRaw(customersPath, nil) },
		"create": func() (int, []byte, error) {
			return portal.Post(customersPath, validCustomerBody(uniqueName("e2e-dc-portal")), newIdempotencyKey())
		},
		"edit its own": func() (int, []byte, error) {
			return portal.Patch(customersPath+"/"+SeedCustomerAccountID, map[string]any{"note": "e2e portal edit"}, newIdempotencyKey())
		},
		"edit another": func() (int, []byte, error) {
			return portal.Patch(customersPath+"/"+otherID, map[string]any{"note": "e2e portal edit"}, newIdempotencyKey())
		},
		"delete another": func() (int, []byte, error) { return portal.Delete(customersPath + "/" + otherID) },
		"bulk delete": func() (int, []byte, error) {
			return portal.Post(dashCustomersBulkDeletePath, map[string]any{"customer_ids": []string{otherID}}, newIdempotencyKey())
		},
		"lead time": func() (int, []byte, error) {
			return portal.GetListRaw(customersPath+"/"+SeedCustomerAccountID+"/lead-time", nil)
		},
	}
	for name, do := range refused {
		status, body, err := do()
		require.NoError(t, err)
		if name == "create" && status == 201 {
			apiClient.Delete(customersPath + "/" + jsonField(parseJSON(body), "id"))
		}
		assert.Equal(t, 403, status, "%s: %s", name, body)
	}

	assertNilField(t, dashCustomersRequireExists(t, otherID), "note")
	_, own = dashCustomersRead(t, apiClient, SeedCustomerAccountID, "")
	assert.NotEqual(t, "e2e portal edit", jsonField(own, "note"))
}

// Another tenant cannot merge, bulk-delete, or reach the frequent products or notification contacts
// of this account's customer, and learns nothing about it existing.
func TestDashCustomers_AnotherTenantCannotReachCustomerActions(t *testing.T) {
	t.Parallel()
	other := getTenantBClient()
	customerID := dashCustomersNew(t, "e2e-dc-iso", nil)
	sourceID := dashCustomersNew(t, "e2e-dc-iso-src", nil)

	calls := map[string]func() (int, []byte, error){
		"merge into it": func() (int, []byte, error) {
			return other.Post(dashCustomersMergePath(customerID), map[string]any{"source_customer_ids": []string{sourceID}}, newIdempotencyKey())
		},
		"bulk delete": func() (int, []byte, error) {
			return other.Post(dashCustomersBulkDeletePath, map[string]any{"customer_ids": []string{customerID}}, newIdempotencyKey())
		},
		"frequent products": func() (int, []byte, error) {
			return other.GetListRaw(customersPath+"/"+customerID+"/frequently-ordered-products", nil)
		},
		"read notification contacts": func() (int, []byte, error) { return other.GetListRaw(notifRecipientsPath(customerID), nil) },
		"replace notification contacts": func() (int, []byte, error) {
			return other.Patch(notifRecipientsPath(customerID), notifRecipientBody(), newIdempotencyKey())
		},
	}
	for name, do := range calls {
		status, body, err := do()
		require.NoError(t, err)
		assert.Equal(t, 404, status, "%s: %s", name, body)
	}
	dashCustomersRequireExists(t, customerID)
	dashCustomersRequireExists(t, sourceID)
}

// Only the owner is told a customer was already deleted; to another tenant the id never existed.
func TestDashCustomers_AnotherTenantCannotTellADeletedCustomerExisted(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, validCustomerBody(uniqueName("e2e-dc-gone")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	customerID := jsonField(parseJSON(body), "id")
	status, body, err = apiClient.Delete(customersPath + "/" + customerID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = getTenantBClient().Delete(customersPath + "/" + customerID)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "another tenant must not learn the customer existed: %s", body)

	status, body, err = apiClient.Delete(customersPath + "/" + customerID)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)
	requireErrorResponse(t, body, "resource_gone", "invalid_request_error")
}

// ---------------------------------------------------------------------------
// Merge and bulk delete
// ---------------------------------------------------------------------------

// Merging moves the source's orders and product line access onto the target before the source is
// deleted, which is why the dashboard offers it in place of a delete that would 409.
func TestDashCustomers_MergeMovesTheSourcesOrdersToTheTarget(t *testing.T) {
	t.Parallel()
	targetID := dashCustomersNew(t, "e2e-dc-mrg-target", nil)
	sourceID := setupOrderCustomer(t)
	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, sourceID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")
	deleteOrder(t, orderID)

	status, body, err = apiClient.Post(dashCustomersMergePath(targetID), map[string]any{"source_customer_ids": []string{sourceID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(customerAccessPath + "/" + targetID) })

	order := retrieveSalesOrder(t, orderID, "customer")
	assert.Equal(t, targetID, jsonField(jsonObject(order, "customer"), "id"), "the source's order now belongs to the target")
	status, _ = dashCustomersRead(t, apiClient, sourceID, "")
	assert.Equal(t, 404, status, "the source is gone")

	status, access := readAccess(t, customerAccessPath+"/"+targetID)
	require.Equal(t, 200, status, "the source's product line access moved to the target")
	assert.Equal(t, []string{SeedProductLineID}, grantedLines(t, access))
}

// Only this account's customers can be merged: a supplier or another tenant's account is not found.
func TestDashCustomers_MergeRefusesAccountsThatAreNotTheSellersCustomers(t *testing.T) {
	t.Parallel()
	targetID := dashCustomersNew(t, "e2e-dc-mrg-notcust", nil)

	for name, sourceID := range map[string]string{"a supplier": SeedSupplierAccountID, "another tenant": SeedTenantBAccountID} {
		status, body, err := apiClient.Post(dashCustomersMergePath(targetID), map[string]any{"source_customer_ids": []string{sourceID}}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 404, status, "%s: %s", name, body)
	}
	status, body, err := apiClient.Post(dashCustomersMergePath(SeedSupplierAccountID), map[string]any{"source_customer_ids": []string{targetID}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "a supplier is not a merge target: %s", body)
	dashCustomersRequireExists(t, targetID)
}

// The customer list's bulk delete removes every customer it names, and a retried request replays the
// first answer instead of failing on customers it already deleted.
func TestDashCustomers_BulkDeleteRemovesEveryNamedCustomer(t *testing.T) {
	t.Parallel()
	a := dashCustomersNew(t, "e2e-dc-bulk-a", nil)
	b := dashCustomersNew(t, "e2e-dc-bulk-b", nil)
	body := map[string]any{"customer_ids": []string{a, b}}
	key := newIdempotencyKey()

	first, err := apiClient.PostFull(dashCustomersBulkDeletePath, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)
	for _, customerID := range []string{a, b} {
		status, _ := dashCustomersRead(t, apiClient, customerID, "")
		assert.Equal(t, 404, status, "customer %s was deleted", customerID)
	}

	replay, err := apiClient.PostFull(dashCustomersBulkDeletePath, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, replay.StatusCode, replay.Body)
	assert.Equal(t, "true", replay.Header.Get("Idempotent-Replayed"))
	assert.JSONEq(t, string(first.Body), string(replay.Body))

	status, resp, err := apiClient.Post(dashCustomersBulkDeletePath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{404, 410}, status, "a fresh request for deleted customers finds nothing: %s", resp)
}

// A bulk delete is all or nothing: one id it cannot delete leaves every named customer in place.
func TestDashCustomers_BulkDeleteIsAllOrNothing(t *testing.T) {
	t.Parallel()
	keep := dashCustomersNew(t, "e2e-dc-bulk-keep", nil)

	cases := []struct {
		name   string
		client *Client
		ids    []string
		status int
	}{
		{"an unknown customer", apiClient, []string{keep, mustGenID(t, id.AccountIDPrefix)}, 404},
		{"a customer with sales orders", apiClient, []string{keep, SeedCustomerAccountID}, 409},
		{"a supplier", apiClient, []string{keep, SeedSupplierAccountID}, 404},
		{"another tenant's request", getTenantBClient(), []string{keep}, 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := tc.client.Post(dashCustomersBulkDeletePath, map[string]any{"customer_ids": tc.ids}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, tc.status, status, body)
			dashCustomersRequireExists(t, keep)
		})
	}

	status, body, err := apiClient.Post(dashCustomersBulkDeletePath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Equal(t, "customer_ids", errorParam(body))
}

// ---------------------------------------------------------------------------
// Frequently ordered products
// ---------------------------------------------------------------------------

// The portal's reorder shortcut counts the customer's own order lines, so a new customer has none and
// one order puts its product on the list.
func TestDashCustomers_FrequentlyOrderedProductsCountTheCustomersOrders(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	path := customersPath + "/" + customerID + "/frequently-ordered-products"

	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Empty(t, jsonArray(parseJSON(body), "data"), "a customer with no orders has no frequent products")

	status, body, err = apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	deleteOrder(t, jsonField(parseJSON(body), "id"))

	status, body, err = apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	data := jsonArray(parseJSON(body), "data")
	require.Len(t, data, 1, "the ordered product is listed: %s", body)
	row := data[0].(map[string]any)
	assert.Equal(t, "frequently_ordered_product", jsonField(row, "object"))
	assert.Equal(t, SeedItemID, jsonField(jsonObject(row, "item"), "id"))
	assert.Equal(t, "1", jsonField(row, "order_count"))
}

// On the seller's own account a customer's frequent products need items:read, as legacy required.
func TestDashCustomers_FrequentlyOrderedProductsNeedItemsReadOnTheOwnAccount(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	deleteOrder(t, jsonField(parseJSON(body), "id"))
	path := customersPath + "/" + customerID + "/frequently-ordered-products"
	want := parseJSON(mustGetAs(t, apiClient, path, nil))
	require.NotEmpty(t, jsonArray(want, "data"), "the customer has ordered")

	assert.Equal(t, want, parseJSON(mustGetAs(t, customRoleClient(t, "items:read"), path, nil)), "an items reader sees what the admin sees")

	for _, perms := range [][]string{{"customers:read"}, {"customers:read", "sales_orders:create"}} {
		status, body, err := customRoleClient(t, perms...).GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, http.StatusForbidden, status, body)
		apiErr := jsonObject(parseJSON(body), "error")
		assert.Contains(t, jsonField(apiErr, "message"), "items:read", "%v is refused for want of items:read", perms)
	}
}

// ---------------------------------------------------------------------------
// Customer registration
// ---------------------------------------------------------------------------

// Registration links the signed-in person to a customer account, so it needs a person: an API key has
// no user to link, and without credentials there is nobody at all.
func TestDashCustomers_RegistrationNeedsASignedInPerson(t *testing.T) {
	t.Parallel()
	body := map[string]any{"account_slug": SeedAccountSlug, "is_existing_customer": true, "customer_number": searchToken("e2edcnone")}

	status, resp, err := apiClient.WithBearerToken("", SeedAccountID).Post(dashCustomersRegistrationPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, resp)

	for name, client := range map[string]*Client{"an admin API key": apiClient, "the customer portal key": getCustomerPortalClient()} {
		status, resp, err := client.Post(dashCustomersRegistrationPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 403, status, "%s is not a person to register: %s", name, resp)
	}
}

// A new buyer registering on the portal becomes a new customer of the seller, numbered by the seller
// and reachable by the buyer; a retried request registers them once.
func TestDashCustomers_RegisteringANewCustomer(t *testing.T) {
	t.Parallel()
	person, email := dashCustomersNewPerson(t)
	name := uniqueName("e2e-dc-reg-new")
	body := map[string]any{
		"account_slug":         SeedAccountSlug,
		"is_existing_customer": false,
		"customer_name":        name,
		"customer_group_id":    SeedCustomerGroupID,
		"phone":                "555-0100",
		"address":              map[string]any{"name": name + " HQ", "country": "US", "street_line_1": "1 Main St", "locality": "Austin", "state": "TX", "postal_code": "78701"},
		"shipping_term_id":     SeedShippingTermID,
		"payment_term_id":      SeedPaymentTermID,
	}
	key := newIdempotencyKey()

	status, resp, err := person.Post(dashCustomersRegistrationPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	ids := dashCustomersCustomerIDsNamed(t, name)
	for _, customerID := range ids {
		t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + customerID) })
	}
	require.Len(t, ids, 1, "the registration created one customer")
	customerID := ids[0]

	replay, err := person.PostFull(dashCustomersRegistrationPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, replay.StatusCode, replay.Body)
	assert.Equal(t, "true", replay.Header.Get("Idempotent-Replayed"))
	assert.Len(t, dashCustomersCustomerIDsNamed(t, name), 1, "a replay registers nobody twice")

	status, got := dashCustomersRead(t, apiClient, customerID, "contact_info,type,defaults.payment_term,defaults.shipping_term,bill_to_address,ship_to_address")
	require.Equal(t, 200, status)
	assert.NotEmpty(t, jsonField(got, "number"), "the seller numbers the new customer")
	contact := jsonObject(got, "contact_info")
	assert.Equal(t, email, jsonField(contact, "email"), "the registrant's email is the customer's contact email")
	assert.Equal(t, "555-0100", jsonField(contact, "phone"))
	assert.Equal(t, SeedCustomerGroupID, jsonField(jsonObject(got, "type"), "id"))
	defaults := jsonObject(got, "defaults")
	assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(defaults, "payment_term"), "id"))
	assert.Equal(t, SeedShippingTermID, jsonField(jsonObject(defaults, "shipping_term"), "id"))
	assert.Equal(t, name+" HQ", jsonField(jsonObject(got, "bill_to_address"), "name"))
	assert.Equal(t, name+" HQ", jsonField(jsonObject(got, "ship_to_address"), "name"))

	assert.Contains(t, dashCustomersVendorAccounts(t, person), customerID, "the registrant belongs to the new customer")
}

// An existing customer's buyer joins that customer by its number, once.
func TestDashCustomers_RegisteringAsAnExistingCustomer(t *testing.T) {
	t.Parallel()
	person, _ := dashCustomersNewPerson(t)
	number := searchToken("e2edcreg")
	customerID := dashCustomersNew(t, "e2e-dc-reg-existing", map[string]any{"number": number})
	body := map[string]any{"account_slug": SeedAccountSlug, "is_existing_customer": true, "customer_number": number}

	status, resp, err := person.Post(dashCustomersRegistrationPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	assert.Equal(t, []string{customerID}, dashCustomersVendorAccounts(t, person))

	status, resp, err = person.Post(dashCustomersRegistrationPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, resp)
	assert.Equal(t, []string{customerID}, dashCustomersVendorAccounts(t, person), "registering twice links once")

	for name, b := range map[string]map[string]any{
		"an unknown number": {"account_slug": SeedAccountSlug, "is_existing_customer": true, "customer_number": searchToken("e2edcnone")},
		"an unknown seller": {"account_slug": searchToken("e2edcnoslug"), "is_existing_customer": true, "customer_number": number},
	} {
		status, resp, err := person.Post(dashCustomersRegistrationPath, b, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 404, status, "%s: %s", name, resp)
	}
	status, resp, err = person.Post(dashCustomersRegistrationPath, map[string]any{"account_slug": SeedAccountSlug, "is_existing_customer": true}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "an existing customer is found by its number: %s", resp)
}

// A new customer needs a name, an address, terms and a group, and the group and terms must be the
// seller's: the registrant picks them, so a foreign id would be stored on the seller's customer.
func TestDashCustomers_RegisteringANewCustomerNeedsTheSellersDetails(t *testing.T) {
	t.Parallel()
	person, _ := dashCustomersNewPerson(t)
	foreignGroup := dashCustomersTenantBGroup(t, "type_group")

	cases := []struct {
		name string
		edit func(map[string]any)
	}{
		{"no name", func(b map[string]any) { delete(b, "customer_name") }},
		{"no address", func(b map[string]any) { delete(b, "address") }},
		{"no shipping term", func(b map[string]any) { delete(b, "shipping_term_id") }},
		{"no payment term", func(b map[string]any) { delete(b, "payment_term_id") }},
		{"no customer group", func(b map[string]any) { delete(b, "customer_group_id") }},
		{"another tenant's group", func(b map[string]any) { b["customer_group_id"] = foreignGroup }},
		{"an unknown payment term", func(b map[string]any) { b["payment_term_id"] = mustGenID(t, id.PaymentTermIDPrefix) }},
		{"an unknown shipping term", func(b map[string]any) { b["shipping_term_id"] = mustGenID(t, id.ShippingTermIDPrefix) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name := uniqueName("e2e-dc-reg-bad")
			body := map[string]any{
				"account_slug":         SeedAccountSlug,
				"is_existing_customer": false,
				"customer_name":        name,
				"customer_group_id":    SeedCustomerGroupID,
				"address":              map[string]any{"name": "HQ", "country": "US"},
				"shipping_term_id":     SeedShippingTermID,
				"payment_term_id":      SeedPaymentTermID,
			}
			tc.edit(body)
			status, resp, err := person.Post(dashCustomersRegistrationPath, body, newIdempotencyKey())
			require.NoError(t, err)
			for _, customerID := range dashCustomersCustomerIDsNamed(t, name) {
				_, _, _ = apiClient.Delete(customersPath + "/" + customerID)
			}
			require.Less(t, status, 500, "%s", resp)
			assert.Contains(t, []int{400, 404}, status, "the registration is refused: %s", resp)
		})
	}
}

// ---------------------------------------------------------------------------
// Addresses
// ---------------------------------------------------------------------------

// The customer page keeps a customer's addresses on the customer's account by targeting it; they are
// not the seller's own addresses.
func TestDashCustomers_SellerManagesACustomersAddresses(t *testing.T) {
	t.Parallel()
	customerID := dashCustomersNew(t, "e2e-dc-addr", nil)
	asCustomer := apiClient.WithAccountID(customerID)
	name := searchToken("e2edcdock")

	status, body, err := asCustomer.Post(addressesPath, map[string]any{
		"name": name, "phone": "555-0101", "country": "US", "street_line_1": "9 Dock Rd", "locality": "Reno", "state": "NV", "postal_code": "89501",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	addressID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = asCustomer.Delete(addressesPath + "/" + addressID) })

	status, body, err = asCustomer.GetListRaw(addressesPath+"/"+addressID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, name, jsonField(parseJSON(body), "name"))
	status, body, err = apiClient.GetListRaw(addressesPath+"/"+addressID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "the customer's address is not the seller's: %s", body)

	listed, _, err := asCustomer.GetList(addressesPath, url.Values{"q": {name}})
	require.NoError(t, err)
	require.Len(t, listed.Data, 1)
	assert.Equal(t, addressID, DataItemField(listed.Data[0], "id"))
	own, _, err := apiClient.GetList(addressesPath, url.Values{"q": {name}})
	require.NoError(t, err)
	assert.Empty(t, own.Data, "the seller's own list does not include it")

	status, body, err = asCustomer.Patch(addressesPath+"/"+addressID, map[string]any{"name": name + "x", "phone": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	updated := parseJSON(body)
	assert.Equal(t, name+"x", jsonField(updated, "name"))
	assertNilField(t, updated, "phone")

	status, body, err = asCustomer.Delete(addressesPath + "/" + addressID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = asCustomer.GetListRaw(addressesPath+"/"+addressID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "%s", body)
}

// On a customer's account, address access follows the customers domain: addresses:* alone is refused
// and customers:update is enough to write.
func TestDashCustomers_CustomerAddressesFollowCustomersPermissions(t *testing.T) {
	t.Parallel()
	customerID := dashCustomersNew(t, "e2e-dc-addrperm", nil)
	addressID := addressOn(t, apiClient, customerID, uniqueName("e2e-dc-addrperm-dock"))
	t.Cleanup(func() { _, _, _ = apiClient.WithAccountID(customerID).Delete(addressesPath + "/" + addressID) })

	addressesOnly := customRoleClient(t, "addresses:create", "addresses:read", "addresses:update", "addresses:delete").WithAccountID(customerID)
	refused := map[string]func() (int, []byte, error){
		"list": func() (int, []byte, error) { return addressesOnly.GetListRaw(addressesPath, nil) },
		"read": func() (int, []byte, error) { return addressesOnly.GetListRaw(addressesPath+"/"+addressID, nil) },
		"create": func() (int, []byte, error) {
			return addressesOnly.Post(addressesPath, map[string]any{"name": "e2e", "country": "US"}, newIdempotencyKey())
		},
		"edit": func() (int, []byte, error) {
			return addressesOnly.Patch(addressesPath+"/"+addressID, map[string]any{"name": "e2e-hijacked"}, newIdempotencyKey())
		},
		"delete": func() (int, []byte, error) { return addressesOnly.Delete(addressesPath + "/" + addressID) },
	}
	for name, do := range refused {
		status, body, err := do()
		require.NoError(t, err)
		assert.Equal(t, 403, status, "addresses:* on a customer's account, %s: %s", name, body)
	}

	customersWriter := customRoleClient(t, "customers:read", "customers:update").WithAccountID(customerID)
	status, body, err := customersWriter.Post(addressesPath, map[string]any{"name": uniqueName("e2e-dc-addrperm-new"), "country": "US"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := jsonField(parseJSON(body), "id")
	status, body, err = customersWriter.Patch(addressesPath+"/"+created, map[string]any{"phone": "555-0102"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = customersWriter.Delete(addressesPath + "/" + created)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.WithAccountID(customerID).GetListRaw(addressesPath+"/"+addressID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEqual(t, "e2e-hijacked", jsonField(parseJSON(body), "name"), "the refused edit changed nothing")
}

// Another tenant reaches neither the seller's addresses nor its customers' addresses.
func TestDashCustomers_AnotherTenantCannotReachAddresses(t *testing.T) {
	t.Parallel()
	other := getTenantBClient()
	name := searchToken("e2edcisoaddr")
	ownID := createE2EAddress(t, name)
	customerID := dashCustomersNew(t, "e2e-dc-isoaddr", nil)

	for label, do := range map[string]func() (int, []byte, error){
		"read": func() (int, []byte, error) { return other.GetListRaw(addressesPath+"/"+ownID, nil) },
		"edit": func() (int, []byte, error) {
			return other.Patch(addressesPath+"/"+ownID, map[string]any{"name": "e2e-hijacked"}, newIdempotencyKey())
		},
		"delete": func() (int, []byte, error) { return other.Delete(addressesPath + "/" + ownID) },
	} {
		status, body, err := do()
		require.NoError(t, err)
		assert.Equal(t, 404, status, "%s: %s", label, body)
	}
	listed, _, err := other.GetList(addressesPath, url.Values{"q": {name}})
	require.NoError(t, err)
	assert.Empty(t, listed.Data, "another tenant's list does not include the address")

	asOurCustomer := other.WithAccountID(customerID)
	status, body, err := asOurCustomer.GetListRaw(addressesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "another tenant cannot target this seller's customer: %s", body)
	status, body, err = asOurCustomer.Post(addressesPath, map[string]any{"name": "e2e", "country": "US"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "%s", body)

	status, body, err = apiClient.GetListRaw(addressesPath+"/"+ownID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, name, jsonField(parseJSON(body), "name"))
}

// The portal's addresses are the buyer's own; the seller's addresses are not found from it.
func TestDashCustomers_PortalAddressesAreTheBuyersOwn(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()
	name := searchToken("e2edcportaladdr")
	sellerID := createE2EAddress(t, name)

	listed, _, err := portal.GetList(addressesPath, url.Values{"q": {name}})
	require.NoError(t, err)
	assert.Empty(t, listed.Data, "the portal does not list the seller's address")
	for label, do := range map[string]func() (int, []byte, error){
		"read": func() (int, []byte, error) { return portal.GetListRaw(addressesPath+"/"+sellerID, nil) },
		"edit": func() (int, []byte, error) {
			return portal.Patch(addressesPath+"/"+sellerID, map[string]any{"name": "e2e-hijacked"}, newIdempotencyKey())
		},
		"delete": func() (int, []byte, error) { return portal.Delete(addressesPath + "/" + sellerID) },
	} {
		status, body, err := do()
		require.NoError(t, err)
		assert.Equal(t, 404, status, "%s: %s", label, body)
	}

	status, body, err := apiClient.GetListRaw(addressesPath+"/"+sellerID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, name, jsonField(parseJSON(body), "name"))
}

// ---------------------------------------------------------------------------
// Child accounts
// ---------------------------------------------------------------------------

// dashCustomersLink links childID under parentID and registers the unlink.
func dashCustomersLink(t *testing.T, parentID, childID string) (int, []byte) {
	t.Helper()
	status, body, err := apiClient.WithAccountID(parentID).Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "linking must not 5xx: %s", body)
	t.Cleanup(func() { _, _, _ = apiClient.WithAccountID(parentID).Delete(childAccountsPath + "/" + childID) })
	return status, body
}

// Linking a child is a PUT: repeating it leaves one link, and the link names the child the way the
// customer page shows it.
func TestDashCustomers_ChildAccountLinkIsRepeatable(t *testing.T) {
	t.Parallel()
	parentID := dashCustomersNew(t, "e2e-dc-link-parent", nil)
	child := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-dc-link-child")))
	childID := jsonField(child, "id")

	status, first := dashCustomersLink(t, parentID, childID)
	requireStatus(t, 200, status, first)
	status, second := dashCustomersLink(t, parentID, childID)
	requireStatus(t, 200, status, second)

	link := parseJSON(second)
	assertIDFormat(t, jsonField(link, "id"), id.AccountRelationIDPrefix)
	assert.Equal(t, jsonField(parseJSON(first), "id"), jsonField(link, "id"), "the repeat names the same link")
	assert.Equal(t, "child_account", jsonField(link, "object"))
	assert.Equal(t, jsonField(child, "number"), jsonField(link, "external_number"))
	assertValidTimestamp(t, jsonField(link, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(link, "updated_at"), "updated_at")
	account := jsonObject(link, "account")
	require.NotNil(t, account)
	assert.Equal(t, childID, jsonField(account, "id"))
	assert.Equal(t, "account", jsonField(account, "object"))
	assert.Equal(t, jsonField(child, "name"), jsonField(account, "name"))
	assert.NotEqual(t, dashCustomersZeroTime, jsonField(account, "created_at"), "the account is the real record, not a stub")

	list, _, err := apiClient.WithAccountID(parentID).GetList(childAccountsPath, nil)
	require.NoError(t, err)
	require.Len(t, list.Data, 1, "a repeated link is one child")
	assert.NotEqual(t, dashCustomersZeroTime, jsonField(jsonObject(parseJSON(list.Data[0]), "account"), "created_at"),
		"listed children carry the real account record")
}

// A hierarchy is a tree: an account can be neither its own parent nor its grandchild's child.
func TestDashCustomers_ChildAccountsCannotFormACycle(t *testing.T) {
	t.Parallel()
	a := dashCustomersNew(t, "e2e-dc-cycle-a", nil)
	b := dashCustomersNew(t, "e2e-dc-cycle-b", nil)
	c := dashCustomersNew(t, "e2e-dc-cycle-c", nil)

	status, body := dashCustomersLink(t, a, a)
	assert.Equal(t, 409, status, "an account cannot be its own parent: %s", body)

	status, body = dashCustomersLink(t, a, b)
	requireStatus(t, 200, status, body)
	status, body = dashCustomersLink(t, b, c)
	requireStatus(t, 200, status, body)
	status, body = dashCustomersLink(t, c, a)
	assert.Equal(t, 409, status, "a -> b -> c -> a is a cycle: %s", body)

	underC, _, err := apiClient.WithAccountID(c).GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.Nil(t, childEntry(underC, a), "the refused link left a where it was")
}

// Child accounts are the seller's customer hierarchy: a supplier or another tenant's account cannot be
// linked into it, and another tenant cannot link the seller's customers.
func TestDashCustomers_ChildAccountMustBeTheSellersCustomer(t *testing.T) {
	t.Parallel()
	parentID := dashCustomersNew(t, "e2e-dc-notcust-parent", nil)
	childID := dashCustomersNew(t, "e2e-dc-notcust-child", nil)

	supplierID := createSupplier(t)

	for name, candidate := range map[string]string{"a supplier": supplierID, "another tenant": SeedTenantBAccountID} {
		status, body := dashCustomersLink(t, parentID, candidate)
		assert.Contains(t, []int{403, 404}, status, "%s is not a customer to link: %s", name, body)
	}

	other := getTenantBClient()
	status, body, err := other.Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status, "another tenant cannot link this seller's customer under itself: %s", body)
	status, body, err = other.WithAccountID(parentID).Put(childAccountsPath+"/"+childID, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status, "another tenant cannot link under this seller's customer: %s", body)

	list, _, err := apiClient.WithAccountID(parentID).GetList(childAccountsPath, nil)
	require.NoError(t, err)
	assert.Empty(t, list.Data, "nothing was linked under the parent")
}

// The customer portal manages no hierarchy.
func TestDashCustomers_PortalCannotManageChildAccounts(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()
	childID := dashCustomersNew(t, "e2e-dc-portal-child", nil)

	for name, do := range map[string]func() (int, []byte, error){
		"list":   func() (int, []byte, error) { return portal.GetListRaw(childAccountsPath, nil) },
		"link":   func() (int, []byte, error) { return portal.Put(childAccountsPath+"/"+childID, nil) },
		"unlink": func() (int, []byte, error) { return portal.Delete(childAccountsPath + "/" + childID) },
	} {
		status, body, err := do()
		require.NoError(t, err)
		assert.Equal(t, 403, status, "%s: %s", name, body)
	}
}

// ---------------------------------------------------------------------------
// Product line access
// ---------------------------------------------------------------------------

// Naming a product line twice grants it once or is refused as invalid; it is not a conflict with a
// record that does not exist yet.
func TestDashCustomers_ProductLineAccessNamingALineTwice(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(customerAccessPath + "/" + f.customerID)
		_, _, _ = apiClient.Delete(accountGroupAccessPath + "/" + f.groupID)
	})

	for _, target := range []struct{ path, key, id string }{
		{customerAccessPath, "customer_id", f.customerID},
		{accountGroupAccessPath, "account_group_id", f.groupID},
	} {
		status, body, err := apiClient.Post(target.path, map[string]any{target.key: target.id, "product_line_ids": []string{f.lineA, f.lineA}}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "%s", body)
		if status == 201 {
			assert.Equal(t, []string{f.lineA}, grantedLines(t, parseJSON(body)))
		} else {
			assert.Equal(t, 400, status, "%s: a repeated line on grant: %s", target.key, body)
			grantAccess(t, target.path, map[string]any{target.key: target.id, "product_line_ids": []string{f.lineA}})
		}

		status, body, err = apiClient.Patch(target.path+"/"+target.id, map[string]any{"product_line_ids": []string{f.lineB, f.lineB}}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "%s", body)
		if status == 200 {
			assert.Equal(t, []string{f.lineB}, grantedLines(t, parseJSON(body)))
		} else {
			assert.Equal(t, 400, status, "%s: a repeated line on edit: %s", target.key, body)
		}
	}
}

// A retried grant replays the first answer rather than reporting the record it created as a conflict.
func TestDashCustomers_ProductLineAccessGrantReplays(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(customerAccessPath + "/" + f.customerID)
		_, _, _ = apiClient.Delete(accountGroupAccessPath + "/" + f.groupID)
	})

	for _, target := range []struct{ path, key, id string }{
		{customerAccessPath, "customer_id", f.customerID},
		{accountGroupAccessPath, "account_group_id", f.groupID},
	} {
		body := map[string]any{target.key: target.id, "product_line_ids": []string{f.lineA}}
		key := newIdempotencyKey()
		first, err := apiClient.PostFull(target.path, body, key)
		require.NoError(t, err)
		requireStatus(t, 201, first.StatusCode, first.Body)
		replay, err := apiClient.PostFull(target.path, body, key)
		require.NoError(t, err)
		requireStatus(t, 201, replay.StatusCode, replay.Body)
		assert.Equal(t, "true", replay.Header.Get("Idempotent-Replayed"))
		assert.JSONEq(t, string(first.Body), string(replay.Body))

		status, read := readAccess(t, target.path+"/"+target.id)
		require.Equal(t, 200, status)
		assert.Equal(t, []string{f.lineA}, grantedLines(t, read))
	}
}

// The access record's customer, group and product lines are the real records, not stubs with blank
// enums and zero timestamps.
func TestDashCustomers_ProductLineAccessCarriesTheRealRecords(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(customerAccessPath + "/" + f.customerID)
		_, _, _ = apiClient.Delete(accountGroupAccessPath + "/" + f.groupID)
	})

	status, body, err := apiClient.Post(customerAccessPath, map[string]any{"customer_id": f.customerID, "product_line_ids": []string{f.lineA}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	customer := jsonObject(created, "customer")
	assert.Equal(t, "customer", jsonField(customer, "object"))
	assert.Equal(t, "normal", jsonField(customer, "status"))
	assert.NotEqual(t, dashCustomersZeroTime, jsonField(customer, "created_at"))
	lines := jsonArray(jsonObject(created, "product_lines"), "data")
	require.Len(t, lines, 1)
	line := lines[0].(map[string]any)
	assert.Equal(t, "commission_applied", jsonField(line, "commission_policy"))
	assert.NotEqual(t, dashCustomersZeroTime, jsonField(line, "created_at"))

	status, body, err = apiClient.Post(accountGroupAccessPath, map[string]any{"account_group_id": f.groupID, "product_line_ids": []string{f.lineA}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	group := jsonObject(parseJSON(body), "account_group")
	assert.Equal(t, "account_group", jsonField(group, "object"))
	assert.Equal(t, "type_group", jsonField(group, "type"))
	assert.NotEqual(t, dashCustomersZeroTime, jsonField(group, "created_at"))
}

// Another tenant's access lists never include this account's records, it cannot grant access to this
// account's group, and the portal manages no access at all.
func TestDashCustomers_ProductLineAccessListsStayWithinTheAccount(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	grantAccess(t, customerAccessPath, map[string]any{"customer_id": f.customerID, "product_line_ids": []string{f.lineA}})
	grantAccess(t, accountGroupAccessPath, map[string]any{"account_group_id": f.groupID, "product_line_ids": []string{f.lineA}})
	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(customerAccessPath + "/" + f.customerID)
		_, _, _ = apiClient.Delete(accountGroupAccessPath + "/" + f.groupID)
	})
	other := getTenantBClient()

	for _, list := range []struct{ path, key, id string }{
		{customerAccessPath, "customer", f.customerID},
		{accountGroupAccessPath, "account_group", f.groupID},
	} {
		params := url.Values{"limit": {"100"}}
		for page := 0; page < 50; page++ {
			got, status, err := other.GetList(list.path, params)
			require.NoError(t, err)
			requireStatus(t, 200, status, nil)
			for _, raw := range got.Data {
				assert.NotEqual(t, list.id, jsonField(jsonObject(parseJSON(raw), list.key), "id"), "another tenant's %s list leaks a record", list.key)
			}
			cursor := got.PageInfo.NextCursor()
			if !got.PageInfo.HasNextPage || cursor == nil {
				break
			}
			params.Set("cursor", *cursor)
		}
	}

	status, body, err := other.Post(accountGroupAccessPath, map[string]any{"account_group_id": f.groupID, "product_line_ids": []string{f.lineB}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "another tenant cannot grant access to this account's group: %s", body)
	_, read := readAccess(t, accountGroupAccessPath+"/"+f.groupID)
	assert.Equal(t, []string{f.lineA}, grantedLines(t, read))

	status, body, err = getCustomerPortalClient().GetListRaw(customerAccessPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "the portal does not manage product line access: %s", body)
}

// ---------------------------------------------------------------------------
// Territories
// ---------------------------------------------------------------------------

// Territories are the seller's own: the portal is refused, a path naming another account finds nothing
// to list or create under, and a role that may only read cannot edit one.
func TestDashCustomers_TerritoriesAreOnlyForTheSellersStaff(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-dc-terr")
	territoryID := jsonField(createTerritory(t, map[string]any{"state": state, "sales_rep_id": SeedAccountUserID}), "id")
	portal := getCustomerPortalClient()

	status, body, err := portal.GetListRaw(territoriesPath(), nil)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "%s", body)
	status, body, err = portal.Post(territoriesPath(), map[string]any{"state": state, "sales_rep_id": SeedAccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "%s", body)

	elsewhere := territoriesPathFor(SeedTenantBAccountID)
	status, body, err = apiClient.GetListRaw(elsewhere, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "listing under another account's path: %s", body)
	status, body, err = apiClient.Post(elsewhere, map[string]any{"state": state, "sales_rep_id": SeedAccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		apiClient.Delete(elsewhere + "/" + jsonField(parseJSON(body), "id"))
	}
	assert.Equal(t, 404, status, "creating under another account's path: %s", body)

	status, body, err = customRoleClient(t, "sales_rep_territories:read").Patch(territoryPath(territoryID), map[string]any{"state": "e2e-ro"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "%s", body)

	assert.Equal(t, []string{territoryID}, searchTerritoryIDs(t, apiClient, territoriesPath(), state), "only the one territory exists")
	assert.Equal(t, state, jsonField(getTerritory(t, territoryID), "state"), "the refused edit changed nothing")
}

// ---------------------------------------------------------------------------
// Sales targets
// ---------------------------------------------------------------------------

func dashCustomersTargetPath(repID, targetID string) string {
	return salesTargetsPathFor(repID) + "/" + targetID
}

func dashCustomersTargetBody(starts, ends, amount string) map[string]any {
	return map[string]any{"starts_at": starts, "ends_at": ends, "amount_value": amount, "amount_unit_id": SeedUnitID}
}

// dashCustomersFindTarget walks a rep's targets for targetID, returning it or nil.
func dashCustomersFindTarget(t *testing.T, client *Client, repID, targetID string) map[string]any {
	t.Helper()
	list, status, err := client.GetList(salesTargetsPathFor(repID), url.Values{"limit": {"100"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	for page := 0; page < 100; page++ {
		for _, raw := range list.Data {
			if DataItemField(raw, "id") == targetID {
				return parseJSON(raw)
			}
		}
		if !list.PageInfo.HasNextPage || list.PageInfo.NextPageURL == nil {
			return nil
		}
		list, _, err = client.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err)
	}
	return nil
}

// The sales target page saves with a PUT on an id it chose: the first creates the target, a repeat
// updates the same one, and a later save changes the amount but keeps the period.
func TestDashCustomers_SalesTargetUpsertCreatesThenUpdatesTheAmount(t *testing.T) {
	t.Parallel()
	targetID := mustGenID(t, id.TargetIDPrefix)
	path := dashCustomersTargetPath(SeedAccountUserID, targetID)
	body := dashCustomersTargetBody("2032-01-01T00:00:00Z", "2032-03-31T00:00:00Z", "100")

	status, resp, err := apiClient.Put(path, body)
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	created := parseJSON(resp)
	assert.Equal(t, targetID, jsonField(created, "id"))
	assert.Equal(t, "sales_target", jsonField(created, "object"))
	assert.Equal(t, SeedAccountUserID, jsonField(jsonObject(created, "sales_rep"), "id"))
	assert.Equal(t, "100", jsonField(jsonObject(created, "amount"), "value"))
	assert.Equal(t, SeedUnitID, jsonField(jsonObject(jsonObject(created, "amount"), "unit"), "id"))

	status, resp, err = apiClient.Put(path, body)
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	assert.Equal(t, targetID, jsonField(parseJSON(resp), "id"))
	assert.Equal(t, jsonField(jsonObject(created, "amount"), "id"), jsonField(jsonObject(parseJSON(resp), "amount"), "id"), "a repeat updates the same amount")

	status, resp, err = apiClient.Put(path, dashCustomersTargetBody("2033-01-01T00:00:00Z", "2033-03-31T00:00:00Z", "250"))
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	updated := parseJSON(resp)
	assert.Equal(t, "250", jsonField(jsonObject(updated, "amount"), "value"))
	assert.Equal(t, jsonField(created, "start_at"), jsonField(updated, "start_at"), "an update keeps the period")
	assert.Equal(t, jsonField(created, "end_at"), jsonField(updated, "end_at"))

	listed := dashCustomersFindTarget(t, apiClient, SeedAccountUserID, targetID)
	require.NotNil(t, listed, "the target is listed under its rep")
	assert.Equal(t, "250", jsonField(jsonObject(listed, "amount"), "value"))
}

// A target with no period or amount, an amount that is not a number, or a period that ends before it
// starts is refused as a client error rather than stored or failed as a 500.
func TestDashCustomers_SalesTargetUpsertValidatesTheBody(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body map[string]any
	}{
		{"an empty body", map[string]any{}},
		{"no amount", map[string]any{"starts_at": "2034-01-01T00:00:00Z", "ends_at": "2034-03-31T00:00:00Z", "amount_unit_id": SeedUnitID}},
		{"an amount that is not a number", dashCustomersTargetBody("2034-01-01T00:00:00Z", "2034-03-31T00:00:00Z", "abc")},
		{"no unit", map[string]any{"starts_at": "2034-01-01T00:00:00Z", "ends_at": "2034-03-31T00:00:00Z", "amount_value": "10"}},
		{"no period", map[string]any{"amount_value": "10", "amount_unit_id": SeedUnitID}},
		{"a period that ends before it starts", dashCustomersTargetBody("2034-03-31T00:00:00Z", "2034-01-01T00:00:00Z", "10")},
	}
	client := dashCustomersNoRetry(apiClient)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			targetID := mustGenID(t, id.TargetIDPrefix)
			status, body, err := client.Put(dashCustomersTargetPath(SeedAccountUserID, targetID), tc.body)
			require.NoError(t, err)
			require.Less(t, status, 500, "%s", body)
			assert.Equal(t, 400, status, "%s", body)
			assert.Nil(t, dashCustomersFindTarget(t, apiClient, SeedAccountUserID, targetID), "nothing was stored")
		})
	}

	status, body, err := apiClient.Put(dashCustomersTargetPath(SeedAccountUserID, mustGenID(t, id.TargetIDPrefix)),
		map[string]any{"starts_at": "2034-01-01T00:00:00Z", "ends_at": "2034-03-31T00:00:00Z", "amount_value": "10", "amount_unit_id": mustGenID(t, id.UnitIDPrefix)})
	require.NoError(t, err)
	assert.Contains(t, []int{400, 404}, status, "an unknown unit is refused: %s", body)
}

// A target is filed under one rep: a save through another rep's path finds nothing rather than
// changing the first rep's target.
func TestDashCustomers_SalesTargetBelongsToTheRepItIsFiledUnder(t *testing.T) {
	t.Parallel()
	targetID := mustGenID(t, id.TargetIDPrefix)
	status, body, err := apiClient.Put(dashCustomersTargetPath(SeedAccountUserID, targetID), dashCustomersTargetBody("2035-01-01T00:00:00Z", "2035-03-31T00:00:00Z", "100"))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Put(dashCustomersTargetPath(SeedAccountUser2ID, targetID), dashCustomersTargetBody("2035-01-01T00:00:00Z", "2035-03-31T00:00:00Z", "999"))
	require.NoError(t, err)
	assert.Equal(t, 404, status, "another rep's path does not reach the target: %s", body)

	got := dashCustomersFindTarget(t, apiClient, SeedAccountUserID, targetID)
	require.NotNil(t, got)
	assert.Equal(t, "100", jsonField(jsonObject(got, "amount"), "value"), "the first rep's target is unchanged")
	assert.Nil(t, dashCustomersFindTarget(t, apiClient, SeedAccountUser2ID, targetID))
}

// Another tenant can neither read nor change this account's targets, nor measure its own targets in
// this account's unit; an unknown rep is not found and the portal has no targets.
func TestDashCustomers_SalesTargetsStayWithinTheAccount(t *testing.T) {
	t.Parallel()
	other := getTenantBClient()
	targetID := mustGenID(t, id.TargetIDPrefix)
	status, body, err := apiClient.Put(dashCustomersTargetPath(SeedAccountUserID, targetID), dashCustomersTargetBody("2036-01-01T00:00:00Z", "2036-03-31T00:00:00Z", "100"))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = other.GetListRaw(salesTargetsPathFor(SeedAccountUserID), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "another tenant cannot list this account's rep: %s", body)
	status, body, err = other.Put(dashCustomersTargetPath(SeedTenantBAccountUserID, targetID), dashCustomersTargetBody("2036-01-01T00:00:00Z", "2036-03-31T00:00:00Z", "999"))
	require.NoError(t, err)
	assert.Equal(t, 404, status, "another tenant cannot save over this account's target: %s", body)
	got := dashCustomersFindTarget(t, apiClient, SeedAccountUserID, targetID)
	require.NotNil(t, got)
	assert.Equal(t, "100", jsonField(jsonObject(got, "amount"), "value"))

	status, body, err = apiClient.Put(dashCustomersTargetPath(mustGenID(t, id.AccountUserIDPrefix), mustGenID(t, id.TargetIDPrefix)),
		dashCustomersTargetBody("2036-01-01T00:00:00Z", "2036-03-31T00:00:00Z", "1"))
	require.NoError(t, err)
	assert.Equal(t, 404, status, "an unknown rep: %s", body)

	status, body, err = other.Put(dashCustomersTargetPath(SeedTenantBAccountUserID, mustGenID(t, id.TargetIDPrefix)),
		dashCustomersTargetBody("2036-01-01T00:00:00Z", "2036-03-31T00:00:00Z", "1"))
	require.NoError(t, err)
	assert.Contains(t, []int{400, 404}, status, "another tenant cannot use this account's unit: %s", body)

	status, body, err = getCustomerPortalClient().GetListRaw(salesTargetsPathFor(SeedAccountUserID), nil)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "the portal has no sales targets: %s", body)
}
