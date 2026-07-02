//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes coverage gaps identified for /v1/sales/customers on top of
// the existing crud_customers_test.go / list_customers_test.go / included_fields_test.go /
// crud_partial_includes_test.go / array_filter_union_test.go suites:
//
//   - POST /v1/sales/customers/{id}/actions/merge has zero prior coverage (lifecycle,
//     validation, 404s, idempotency, permission).
//   - 7 ListCustomersRequest query params had no coverage at all: shipping_term_ids,
//     payment_term_ids, freight_status_codes, carrier_ids, service_level_ids,
//     parent_account_status, city/state/postal_code, start_date/end_date.
//   - relationship_type was never asserted as anything but "standalone"; parent_account
//     and child_accounts were never asserted non-nil with real nested data despite seed
//     fixtures existing specifically for this; freight_preferences.service_level was
//     never asserted populated with real data.
//   - DELETE 409-conflict (customer referenced by sales orders) was undocumented/untested.

// covSalesCustomersCreate creates a standalone customer via validCustomerBody merged
// with extra fields, returning the parsed created resource. Registers cleanup.
func covSalesCustomersCreate(t *testing.T, extra map[string]any) map[string]any {
	t.Helper()
	body := validCustomerBody(uniqueName("e2e-covcust"))
	for k, v := range extra {
		body[k] = v
	}
	return createAndCleanup(t, customersPath, body)
}

// ──────────────────────────────────────────────
// Merge Action — Lifecycle & Response Shape
// ──────────────────────────────────────────────

func TestCovSalesCustomers_MergeHappyPath(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, nil)
	targetID := jsonField(target, "id")

	source1Status, source1Body, err := apiClient.Post(customersPath, validCustomerBody(uniqueName("e2e-covcust-merge-src1")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, source1Status, source1Body)
	source1ID := jsonField(parseJSON(source1Body), "id")

	source2Status, source2Body, err := apiClient.Post(customersPath, validCustomerBody(uniqueName("e2e-covcust-merge-src2")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, source2Status, source2Body)
	source2ID := jsonField(parseJSON(source2Body), "id")

	mergeStatus, mergeBody, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{
		"source_customer_ids": []string{source1ID, source2ID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, mergeStatus, mergeBody)

	merged := parseJSON(mergeBody)
	assertIDFormat(t, jsonField(merged, "id"), "ac")
	assertObjectField(t, merged, "customer")
	assert.Equal(t, targetID, jsonField(merged, "id"), "merge response should be the target customer")
	assertValidTimestamp(t, jsonField(merged, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(merged, "updated_at"), "updated_at")

	// Sources are deleted after a successful merge.
	src1Status, _, err := apiClient.GetListRaw(customersPath+"/"+source1ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, src1Status, "source customer 1 should be deleted after merge")

	src2Status, _, err := apiClient.GetListRaw(customersPath+"/"+source2ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, src2Status, "source customer 2 should be deleted after merge")

	// Target still exists and is retrievable normally.
	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+targetID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
}

func TestCovSalesCustomers_MergeConsolidatesPriceGroups(t *testing.T) {
	t.Parallel()

	pg := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covcust-merge-pg"),
		"type": "pricing_group",
	})
	pgID := jsonField(pg, "id")

	target := covSalesCustomersCreate(t, nil)
	targetID := jsonField(target, "id")

	source := covSalesCustomersCreate(t, map[string]any{
		"customer_price_group_ids": []string{pgID},
	})
	sourceID := jsonField(source, "id")

	// Cancel the auto-registered cleanup for the source: it will be deleted by
	// the merge itself, and a post-merge Delete call would 404 harmlessly, but
	// we don't want to rely on that — call merge explicitly instead.
	mergeStatus, mergeBody, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge?include=price_groups", map[string]any{
		"source_customer_ids": []string{sourceID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, mergeStatus, mergeBody)

	merged := parseJSON(mergeBody)
	pgList := jsonObject(merged, "price_groups")
	require.NotNil(t, pgList, "price_groups should be present with ?include=price_groups")
	pgData, _ := pgList["data"].([]any)
	require.Len(t, pgData, 1, "target should have absorbed the source's price group")
	if m, ok := pgData[0].(map[string]any); ok {
		assert.Equal(t, pgID, jsonField(m, "id"))
	}
}

func TestCovSalesCustomers_MergeWithInclude(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, nil)
	targetID := jsonField(target, "id")

	source := covSalesCustomersCreate(t, nil)
	sourceID := jsonField(source, "id")

	mergeStatus, mergeBody, err := apiClient.Post(
		customersPath+"/"+targetID+"/actions/merge?include=type,defaults.payment_term,defaults.shipping_term",
		map[string]any{"source_customer_ids": []string{sourceID}},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, mergeStatus, mergeBody)

	merged := parseJSON(mergeBody)

	typeGroup := jsonObject(merged, "type")
	require.NotNil(t, typeGroup, "type should be present with ?include=type on merge")
	assert.Equal(t, SeedCustomerGroupID, jsonField(typeGroup, "id"))

	defaults := jsonObject(merged, "defaults")
	require.NotNil(t, defaults, "defaults should be present with ?include=defaults.payment_term on merge")

	pt := jsonObject(defaults, "payment_term")
	require.NotNil(t, pt, "defaults.payment_term should be present on merge response")
	assert.Equal(t, SeedPaymentTermID, jsonField(pt, "id"))

	// Not included, should remain null.
	assert.Nil(t, merged["contact_info"], "contact_info should be null without ?include=contact_info")
}

// ──────────────────────────────────────────────
// Merge Action — Validation
// ──────────────────────────────────────────────

func TestCovSalesCustomers_MergeSelfMergeRejected(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, nil)
	targetID := jsonField(target, "id")

	status, body, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{
		"source_customer_ids": []string{targetID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovSalesCustomers_MergeDuplicateSourceIDsRejected(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, nil)
	targetID := jsonField(target, "id")

	source := covSalesCustomersCreate(t, nil)
	sourceID := jsonField(source, "id")

	status, body, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{
		"source_customer_ids": []string{sourceID, sourceID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovSalesCustomers_MergeNonexistentSourceNotFound(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, nil)
	targetID := jsonField(target, "id")

	status, body, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{
		"source_customer_ids": []string{"ac_000000000000000000000000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovSalesCustomers_MergeNonexistentTargetNotFound(t *testing.T) {
	t.Parallel()

	source := covSalesCustomersCreate(t, nil)
	sourceID := jsonField(source, "id")

	status, body, err := apiClient.Post(customersPath+"/ac_000000000000000000000000/actions/merge", map[string]any{
		"source_customer_ids": []string{sourceID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovSalesCustomers_MergeEmptySourceListIsNoOp documents current behavior:
// an empty source_customer_ids array is a well-formed no-op merge (target
// returned unchanged, 200), not rejected by validation. go-playground/validator's
// `required` tag treats a non-nil empty slice as satisfying the constraint. This
// is plausibly intentional (merge is idempotent/a no-op with nothing to merge)
// rather than a bug, so we assert the observed behavior rather than force a 400.
func TestCovSalesCustomers_MergeEmptySourceListIsNoOp(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, nil)
	targetID := jsonField(target, "id")

	status, body, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{
		"source_customer_ids": []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	merged := parseJSON(body)
	assert.Equal(t, targetID, jsonField(merged, "id"))
}

// ──────────────────────────────────────────────
// Merge Action — Idempotency & Permission
// ──────────────────────────────────────────────

func TestCovSalesCustomers_MergeIdempotentReplay(t *testing.T) {
	t.Parallel()

	target := covSalesCustomersCreate(t, nil)
	targetID := jsonField(target, "id")

	source := covSalesCustomersCreate(t, nil)
	sourceID := jsonField(source, "id")

	idemKey := newIdempotencyKey()
	payload := map[string]any{"source_customer_ids": []string{sourceID}}

	status1, body1, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	id2 := jsonField(parseJSON(body2), "id")

	assert.Equal(t, id1, id2, "replaying the same idempotency key should return the same merged target")

	// Source must still be gone (not re-merged / re-deleted a second time).
	srcStatus, _, err := apiClient.GetListRaw(customersPath+"/"+sourceID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, srcStatus)
}

func TestCovSalesCustomers_MergePortalActorForbidden(t *testing.T) {
	t.Parallel()

	customerPortalClient := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)

	status, body, err := customerPortalClient.Post(customersPath+"/"+SeedCustomerAccountID+"/actions/merge", map[string]any{
		"source_customer_ids": []string{"ac_000000000000000000000000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// ──────────────────────────────────────────────
// List Filters — shipping_term_ids / payment_term_ids
// ──────────────────────────────────────────────

func TestCovSalesCustomers_ListFilterByShippingTermID(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"shipping_term_ids": {SeedShippingTermID}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "filter by seeded shipping term should return at least 1 result")
}

func TestCovSalesCustomers_ListFilterByShippingTermID_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"shipping_term_ids": {"shtm_00000000000000000000000000"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "bogus shipping term id should return no results")
}

func TestCovSalesCustomers_ListFilterByPaymentTermID(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"payment_term_ids": {SeedPaymentTermID}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "filter by seeded payment term should return at least 1 result")
}

func TestCovSalesCustomers_ListFilterByPaymentTermID_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"payment_term_ids": {"pytm_00000000000000000000000000"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "bogus payment term id should return no results")
}

// ──────────────────────────────────────────────
// List Filters — carrier_ids / service_level_ids
// ──────────────────────────────────────────────

func TestCovSalesCustomers_ListFilterByCarrierID(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"carrier_ids": {SeedCarrierID}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "filter by seeded carrier should return at least 1 result")
}

func TestCovSalesCustomers_ListFilterByCarrierID_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"carrier_ids": {"bogus_carrier_id_xyz"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "bogus carrier id should return no results")
}

func TestCovSalesCustomers_ListFilterByServiceLevelID(t *testing.T) {
	t.Parallel()
	assertListContainsID(t, customersPath, url.Values{"service_level_ids": {SeedServiceLevelID}}, SeedCustomerAccountID)
}

func TestCovSalesCustomers_ListFilterByServiceLevelID_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"service_level_ids": {"crop_00000000000000000000000000"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "bogus service level id should return no results")
}

// ──────────────────────────────────────────────
// List Filters — freight_status_codes (enum; no bogus value available, so the
// negative case is proven by scoping to a customer created with a known,
// non-matching freight_policy).
// ──────────────────────────────────────────────

func TestCovSalesCustomers_ListFilterByFreightStatusCode(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covcust-freight")
	cust := covSalesCustomersCreate(t, map[string]any{
		"name":           name,
		"freight_policy": "free_freight",
	})

	list, _, err := apiClient.GetList(customersPath, url.Values{
		"freight_status_codes": {"free_freight"},
		"q":                    {name},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "filter by free_freight should return the newly created free_freight customer")
	assert.True(t, listContainsID(list, jsonField(cust, "id")))
}

func TestCovSalesCustomers_ListFilterByFreightStatusCode_NoResults(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covcust-freight-neg")
	covSalesCustomersCreate(t, map[string]any{
		"name":           name,
		"freight_policy": "free_freight",
	})

	// The customer was created with free_freight; filtering for billed_freight
	// scoped to its unique name must exclude it.
	list, _, err := apiClient.GetList(customersPath, url.Values{
		"freight_status_codes": {"billed_freight"},
		"q":                    {name},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "free_freight customer should not match a billed_freight filter")
}

// ──────────────────────────────────────────────
// List Filters — parent_account_status
// ──────────────────────────────────────────────

func TestCovSalesCustomers_ListFilterByParentAccountStatus_Parent(t *testing.T) {
	t.Parallel()
	assertListContainsID(t, customersPath, url.Values{"parent_account_status": {"parent"}}, SeedCustomerAccountID)
}

func TestCovSalesCustomers_ListFilterByParentAccountStatus_NonParent(t *testing.T) {
	t.Parallel()

	standalone := covSalesCustomersCreate(t, nil)
	standaloneID := jsonField(standalone, "id")

	nonParentParams := url.Values{"parent_account_status": {"non_parent"}}
	assertListContainsID(t, customersPath, nonParentParams, standaloneID)

	found := listFindByField(t, customersPath, nonParentParams, "id", SeedCustomerAccountID)
	assert.Nil(t, found, "a customer with child accounts must not appear under parent_account_status=non_parent")
}

// ──────────────────────────────────────────────
// List Filters — city / state / postal_code
// ──────────────────────────────────────────────

func TestCovSalesCustomers_ListFilterByCityStatePostalCode(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covcust-addrfilt")
	city := name + "-City"
	cust := covSalesCustomersCreate(t, map[string]any{
		"name": name,
		"bill_to_address": map[string]any{
			"name":        name + " Billing",
			"country":     "US",
			"locality":    city,
			"state":       "ZZ",
			"postal_code": "99999",
		},
	})
	custID := jsonField(cust, "id")

	cityList, _, err := apiClient.GetList(customersPath, url.Values{"city": {city}})
	require.NoError(t, err)
	assert.True(t, listContainsID(cityList, custID), "city filter should match the newly created customer's bill-to address")

	combinedList, _, err := apiClient.GetList(customersPath, url.Values{"city": {city}, "state": {"ZZ"}, "postal_code": {"99999"}})
	require.NoError(t, err)
	assert.True(t, listContainsID(combinedList, custID), "combined city+state+postal_code filter should match")

	noMatchList, _, err := apiClient.GetList(customersPath, url.Values{"city": {city}, "postal_code": {"00000"}})
	require.NoError(t, err)
	assert.False(t, listContainsID(noMatchList, custID), "mismatched postal_code combined with a matching city must exclude the customer")
}

func TestCovSalesCustomers_ListFilterByCity_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"city": {"zzz-nonexistent-city-e2e-99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "nonexistent city should return no results")
}

// ──────────────────────────────────────────────
// List Filters — start_date / end_date
// ──────────────────────────────────────────────

func TestCovSalesCustomers_ListFilterByDateRange(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covcust-daterange")
	cust := covSalesCustomersCreate(t, map[string]any{"name": name})
	custID := jsonField(cust, "id")

	// Bracket: a wide window covering "now" must include the customer.
	inRange, _, err := apiClient.GetList(customersPath, url.Values{
		"start_date": {"2026-01-01T00:00:00Z"},
		"end_date":   {"2030-01-01T00:00:00Z"},
		"q":          {name},
	})
	require.NoError(t, err)
	assert.True(t, listContainsID(inRange, custID), "wide date range bracketing now should include the newly created customer")

	// start_date in the far future must exclude it.
	afterRange, _, err := apiClient.GetList(customersPath, url.Values{
		"start_date": {"2030-01-01T00:00:00Z"},
		"q":          {name},
	})
	require.NoError(t, err)
	assertEmptyListData(t, afterRange.Data, "start_date in the far future should exclude a customer created now")

	// end_date in the far past must exclude it.
	beforeRange, _, err := apiClient.GetList(customersPath, url.Values{
		"end_date": {"2020-01-01T00:00:00Z"},
		"q":        {name},
	})
	require.NoError(t, err)
	assertEmptyListData(t, beforeRange.Data, "end_date in the far past should exclude a customer created now")
}

// ──────────────────────────────────────────────
// allFields Gap-Closing — relationship_type / parent_account / child_accounts
// ──────────────────────────────────────────────

// TestCovSalesCustomers_RelationshipTypeAndParentChildPopulated closes prodBugSuspect
// #1: SeedCustomerAccountID has both a parent (via acre_01seedhouseacct0000) and two
// seeded children (ac_01seedcustchild00001/2). This asserts real, non-null
// materialized data rather than mere key-presence.
func TestCovSalesCustomers_RelationshipTypeAndParentChildPopulated(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"parent_account,child_accounts"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)

	// The seeded customer has 2 children of its own, so it is classified "parent"
	// even though it also has a parent (the house account) — relationship_type
	// reflects whether the customer itself has children.
	assert.Equal(t, "parent", jsonField(got, "relationship_type"))

	parentAccount := jsonObject(got, "parent_account")
	require.NotNil(t, parentAccount, "parent_account should be populated with real data for a customer with a parent relation")
	assert.Equal(t, SeedAccountID, jsonField(parentAccount, "id"))
	assert.Equal(t, "customer", jsonField(parentAccount, "object"))

	childAccounts := jsonObject(got, "child_accounts")
	require.NotNil(t, childAccounts, "child_accounts should be populated with real data")
	assert.Equal(t, "list", jsonField(childAccounts, "object"))
	childData, _ := childAccounts["data"].([]any)
	require.Len(t, childData, 2, "seeded customer should have exactly 2 child accounts")

	childIDs := make(map[string]bool, len(childData))
	for _, item := range childData {
		if m, ok := item.(map[string]any); ok {
			childIDs[jsonField(m, "id")] = true
			assert.Equal(t, "child", jsonField(m, "relationship_type"), "each seeded child should itself report relationship_type=child")
		}
	}
	assert.True(t, childIDs["ac_01seedcustchild00001"], "expected seeded child 1 in child_accounts")
	assert.True(t, childIDs["ac_01seedcustchild00002"], "expected seeded child 2 in child_accounts")
}

// TestCovSalesCustomers_FreightPreferencesServiceLevelPopulated closes prodBugSuspect
// #2: SeedCustomerAccountID has default_carrier_option_id set to a real service level
// seed row; this asserts freight_preferences.service_level is populated with real
// field data (not just key-presence) when explicitly included.
func TestCovSalesCustomers_FreightPreferencesServiceLevelPopulated(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"freight_preferences,freight_preferences.service_level"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	fp := jsonObject(got, "freight_preferences")
	require.NotNil(t, fp, "freight_preferences should be present with ?include=freight_preferences")

	sl := jsonObject(fp, "service_level")
	require.NotNil(t, sl, "freight_preferences.service_level should be populated with real data for the seeded fixture")
	assert.Equal(t, SeedServiceLevelID, jsonField(sl, "id"))
	assert.Equal(t, "service_level", jsonField(sl, "object"))
	assert.NotEmpty(t, jsonField(sl, "name"), "service_level.name should not be empty")
}

// TestCovSalesCustomers_CreateAndClearDefaultServiceLevel exercises
// default_service_level_id end-to-end on create, on update (set), and on update
// (clear via null) — this field was never sent by any prior test.
func TestCovSalesCustomers_CreateAndClearDefaultServiceLevel(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covcust-svclvl")
	createResp, err := apiClient.PostFull(customersPath+"?include=freight_preferences.service_level", map[string]any{
		"name":                     name,
		"default_carrier_id":       SeedCarrierID,
		"default_payment_term_id":  SeedPaymentTermID,
		"default_shipping_term_id": SeedShippingTermID,
		"customer_type_group_id":   SeedCustomerGroupID,
		"default_service_level_id": SeedServiceLevelID,
		"bill_to_address": map[string]any{
			"name":    name + " Billing",
			"country": "US",
		},
		"ship_to_address": map[string]any{
			"name":    name + " Shipping",
			"country": "US",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(customersPath + "/" + id)

	fp := jsonObject(created, "freight_preferences")
	require.NotNil(t, fp, "freight_preferences should be present with ?include on create")
	sl := jsonObject(fp, "service_level")
	require.NotNil(t, sl, "freight_preferences.service_level should be set from default_service_level_id on create")
	assert.Equal(t, SeedServiceLevelID, jsonField(sl, "id"))

	// Clear via null.
	clearStatus, clearBody, err := apiClient.Patch(customersPath+"/"+id+"?include=freight_preferences.service_level", map[string]any{
		"default_service_level_id": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)

	cleared := parseJSON(clearBody)
	clearedFP := jsonObject(cleared, "freight_preferences")
	require.NotNil(t, clearedFP, "freight_preferences should still be present")
	assertNilField(t, clearedFP, "service_level")
}

// ──────────────────────────────────────────────
// DELETE Conflict (409) — customer referenced by sales orders
// ──────────────────────────────────────────────

// TestCovSalesCustomers_DeleteConflictWhenReferencedBySalesOrders verifies the
// documented 409 conflict for deleting a customer still referenced by sales
// orders. Uses SeedCustomerAccountID (referenced by seeded sales orders) — this
// is safe specifically because the endpoint is expected to reject the delete
// and leave the row untouched; the test additionally asserts the customer still
// exists afterward to guard against a false-positive 409 that deletes anyway.
func TestCovSalesCustomers_DeleteConflictWhenReferencedBySalesOrders(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(customersPath + "/" + SeedCustomerAccountID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")

	// Confirm the customer was NOT deleted.
	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, SeedCustomerAccountID, jsonField(parseJSON(getBody), "id"))
}
