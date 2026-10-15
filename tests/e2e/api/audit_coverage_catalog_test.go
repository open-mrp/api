//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Audit coverage for writes that used to persist without an audit event, or with an event
// whose diff came out empty and was dropped.

// requireAuditChange returns the named field change from the first update event on the resource.
func requireAuditChange(t *testing.T, resourceID, resourceType, field string) map[string]any {
	t.Helper()
	var change map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		for _, event := range auditEventsFor(t, resourceID, resourceType, "update") {
			if c, ok := changeForField(jsonListData(event, "changes"), field); ok {
				change = c
				return nil
			}
		}
		return fmt.Errorf("no update on %s %s changes %s yet", resourceType, resourceID, field)
	})
	return change
}

// requireAuditMetadata returns the first event on the resource whose metadata[key] equals value.
func requireAuditMetadata(t *testing.T, resourceID, resourceType, action, key, value string) map[string]any {
	t.Helper()
	var matched map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		for _, event := range auditEventsFor(t, resourceID, resourceType, action) {
			if metadata, _ := event["metadata"].(map[string]any); metadata[key] == value {
				matched = event
				return nil
			}
		}
		return fmt.Errorf("no %s on %s %s with metadata %s=%s yet", action, resourceType, resourceID, key, value)
	})
	return matched
}

func auditEventsFor(t *testing.T, resourceID, resourceType, action string) []map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
		"resource_ids":   {resourceID},
		"resource_types": {resourceType},
		"actions":        {action},
		"include":        {"changes,metadata"},
		"limit":          {"100"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var events []map[string]any
	for _, raw := range jsonArray(parseJSON(body), "data") {
		if event, ok := raw.(map[string]any); ok {
			events = append(events, event)
		}
	}
	return events
}

// requireRateChange asserts an audited price. The audit diff keeps the stored rate's shape: a value
// over a currency unit and a per unit.
func requireRateChange(t *testing.T, change map[string]any, wantValue string) {
	t.Helper()
	newValue, ok := change["new_value"].(map[string]any)
	require.True(t, ok, "a rate change carries the rate as an object: %v", change)
	assert.Equal(t, wantValue, jsonField(newValue, "value"))
	assert.Equal(t, currencyUnitID, jsonField(newValue, "numerator_unit_id"))
	assert.Equal(t, nonCurrencyUnitID, jsonField(newValue, "denominator_unit_id"))
}

// --- Rates on single updates ---

func TestAuditCoverage_ProductPriceOnlyPatch(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(productsPath, validProductBody(uniqueName("e2e-audit-price")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	productID := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(productsPath + "/" + productID)

	status, body, err = apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"unit_price": bulkCurrencyRate("12.50"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	requireRateChange(t, requireAuditChange(t, productID, "product", "unit_price"), "12.50")
}

func TestAuditCoverage_MaterialCostOnlyPatch(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(materialsPath, validMaterialBody(uniqueName("e2e-audit-cost")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	materialID := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(materialsPath + "/" + materialID)

	status, body, err = apiClient.Patch(materialsPath+"/"+materialID, map[string]any{
		"unit_cost": bulkCurrencyRate("3.25"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	requireRateChange(t, requireAuditChange(t, materialID, "material", "unit_cost"), "3.25")
}

// --- Bulk upsert update rows ---

// The update branch writes rates and attributes after the row's own fields; the event has to include them.
func TestAuditCoverage_BulkUpsertUpdateIncludesRatesAndAttributes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		resourceType string
		path         string
		categoryID   string
		upsert       func(t *testing.T, rows ...map[string]any) ([]string, []string)
	}{
		{"product", "product", productsPath, SeedItemCategoryID, bulkUpsertProductIDs},
		{"material", "material", materialsPath, SeedMaterialCategoryID, bulkUpsertMaterialIDs},
		{"part", "part", partsPath, SeedItemCategoryID, bulkUpsertPartIDs},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sku := uniqueName("e2e-audit-bup-" + tc.name)
			created, _ := tc.upsert(t, map[string]any{"sku": sku, "category": map[string]any{"id": tc.categoryID}})
			require.Len(t, created, 1)
			resourceID := created[0]
			defer apiClient.Delete(tc.path + "/" + resourceID)

			_, updated := tc.upsert(t, map[string]any{
				"sku":        sku,
				"category":   map[string]any{"id": tc.categoryID},
				"unit_price": bulkCurrencyRate("7.75"),
				"unit_cost":  bulkCurrencyRate("2.50"),
				"properties": []any{bulkProperty(uniqueName("e2e-audit-prop"), uniqueName("v"))},
			})
			require.Equal(t, []string{resourceID}, updated)

			event := expectAuditEventWithChanges(t, resourceID, tc.resourceType, "update")
			changes := jsonListData(event, "changes")
			price, ok := changeForField(changes, "unit_price")
			require.True(t, ok, "unit_price change missing: %v", changes)
			requireRateChange(t, price, "7.75")
			cost, ok := changeForField(changes, "unit_cost")
			require.True(t, ok, "unit_cost change missing: %v", changes)
			requireRateChange(t, cost, "2.50")
			_, ok = changeForField(changes, "attributes")
			assert.True(t, ok, "attributes change missing: %v", changes)
		})
	}
}

// A property value the batch introduces creates an attribute, which is audited on its own.
func TestAuditCoverage_BulkUpsertCreatesAuditedAttribute(t *testing.T) {
	t.Parallel()

	value := uniqueName("e2e-audit-attr")
	created, _ := bulkUpsertProductIDs(t, map[string]any{
		"sku":        uniqueName("e2e-audit-attr-prod"),
		"category":   map[string]any{"id": SeedItemCategoryID},
		"properties": []any{bulkProperty(uniqueName("e2e-audit-attr-prop"), value)},
	})
	require.Len(t, created, 1)
	defer cleanupProductIDs(created)

	itemID := jsonField(catalogItem(t, productsPath, created[0]), "id")
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var attributeID string
	for _, raw := range jsonListData(parseJSON(body), "attributes") {
		if attr, ok := raw.(map[string]any); ok && jsonField(attr, "value") == value {
			attributeID = jsonField(attr, "id")
		}
	}
	require.NotEmpty(t, attributeID, "the new attribute is attached to the item")

	expectAuditEvent(t, attributeID, "attribute", "create")
}

// --- Bulk create items on an existing SKU ---

func TestAuditCoverage_BulkCreateItemsExistingSKU(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-audit-bci")
	status, body, err := apiClient.Post(productsPath, validProductBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	productID := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(productsPath + "/" + productID)
	itemID := jsonField(catalogItem(t, productsPath, productID), "id")

	description := uniqueName("e2e-audit-bci-desc")
	status, body, err = apiClient.Post(itemsPath+"/actions/bulk-create", map[string]any{
		"type": "product",
		"items": []any{map[string]any{
			"sku":              sku,
			"description":      description,
			"item_category_id": SeedItemCategoryID,
			"product_line_id":  SeedProductLineID,
		}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	desc := requireAuditChange(t, itemID, "item", "description")
	assert.Equal(t, description, desc["new_value"])
	line := requireAuditChange(t, productID, "product", "product_line_id")
	assert.Equal(t, SeedProductLineID, line["new_value"])
}

// --- Customer credit limit ---

func TestAuditCoverage_CustomerCreditLimitOnlyPatch(t *testing.T) {
	t.Parallel()

	body := validCustomerBody(uniqueName("e2e-audit-credit"))
	body["credit_limit"] = map[string]any{"amount": "1000", "currency": "usd"}
	status, raw, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	customerID := jsonField(parseJSON(raw), "id")
	defer apiClient.Delete(customersPath + "/" + customerID)

	status, raw, err = apiClient.Patch(customersPath+"/"+customerID, map[string]any{
		"credit_limit": map[string]any{"amount": "2500", "currency": "usd"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	change := requireAuditChange(t, customerID, "customer", "credit_limit_value")
	newValue, err := strconv.ParseFloat(fmt.Sprint(change["new_value"]), 64)
	require.NoError(t, err)
	assert.InDelta(t, 2500, newValue, 0.0001)
}

// --- Location and category links in bulk upserts ---

func TestAuditCoverage_BulkUpsertLocationParentLink(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(locationsPath, map[string]any{
		"name": uniqueName("e2e-audit-loc-parent"),
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	parentID := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(locationsPath + "/" + parentID)

	created, _ := bulkUpsertLocations(t, map[string]any{
		"name":   uniqueName("e2e-audit-loc-child"),
		"type":   "section",
		"parent": map[string]any{"id": parentID},
	})
	defer cleanupLocationIDs(created)
	require.Len(t, created, 1)

	change := requireAuditChange(t, created[0], "location", "parent_id")
	assert.Equal(t, parentID, change["new_value"])
}

func TestAuditCoverage_BulkUpsertCategoryPropertyLinks(t *testing.T) {
	t.Parallel()

	row := bulkUpsertICInput(uniqueName("e2e-audit-cat"), "product", SeedUnitGroupID)
	row["property_names"] = []any{uniqueName("e2e-audit-cat-prop")}
	created, _ := bulkUpsertItemCategoryIDs(t, row)
	defer cleanupItemCategoryIDs(created)
	require.Len(t, created, 1)

	change := requireAuditChange(t, created[0], "item_category", "properties")
	assert.NotNil(t, change["new_value"])
}

// --- Production runs ---

func TestAuditCoverage_ProductionRunBatchesAudited(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
	planned := runBatches(t, runID, "run")
	require.Len(t, planned, 1)
	expectAuditEvent(t, jsonField(planned[0], "id"), "batch", "create")

	status, body, err := apiClient.Post(productionRunBatchesPath(runID), map[string]any{
		"batches": []any{plannedBatch("2"), plannedBatch("3")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	for _, raw := range jsonArray(parseJSON(body), "data") {
		expectAuditEvent(t, jsonField(raw.(map[string]any), "id"), "batch", "create")
	}
}

// --- Production schedules ---

func TestAuditCoverage_ScheduleGenerateAndRegenerate(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-audit-sched"))
	scheduleID := jsonField(schedule, "id")
	expectAuditEvent(t, scheduleID, "production_schedule", "create")

	addLine(t, scheduleID, map[string]any{"week_index": 3, "quantity": 999})
	regenerate(t, scheduleID, map[string]any{"merge_mode": "replace_all"})

	event := requireAuditMetadata(t, scheduleID, "production_schedule", "update", "reason", "regenerate")
	metadata := event["metadata"].(map[string]any)
	assert.Equal(t, "replace_all", metadata["merge_mode"])
	assert.EqualValues(t, 1, metadata["manual_lines_discarded"], "the hand-added line is discarded by replace_all")
	assert.EqualValues(t, 0, metadata["manual_lines_kept"])
}

func TestAuditCoverage_PublishAuditsSupersededSchedule(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	firstID := jsonField(ownedSchedule(t, uniqueName("e2e-audit-supersede-1")), "id")
	status, body, err := apiClient.Put(schedulePath(firstID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	secondID := jsonField(ownedSchedule(t, uniqueName("e2e-audit-supersede-2")), "id")
	status, body, err = apiClient.Put(schedulePath(secondID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var superseded map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		for _, event := range auditEventsFor(t, firstID, "production_schedule", "update") {
			if c, ok := changeForField(jsonListData(event, "changes"), "status_code"); ok && c["new_value"] == "superseded" {
				superseded = event
				return nil
			}
		}
		return fmt.Errorf("no superseded status change on %s yet", firstID)
	})
	_, ok := changeForField(jsonListData(superseded, "changes"), "superseded_by_id")
	assert.True(t, ok, "the superseding schedule is recorded")
}

// --- Self-registration and password reset ---

func TestAuditCoverage_PortalRegistrationAuditsCustomer(t *testing.T) {
	t.Parallel()

	buyer := newPortalBuyerClient(t)
	sessionPath := portalRegSessionsPath + "/" + jsonField(startPortalRegistration(t, buyer), "id")
	name := "E2E Audit Registrant " + uuid.New().String()[:8]

	status, body, err := buyer.Patch(sessionPath, map[string]any{
		"step":                 "contact",
		"session_data":         fullNewCustomerData(name),
		"is_existing_customer": false,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = buyer.Post(sessionPath+"/actions/complete", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(customersPath, url.Values{"q": {name}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	customers := jsonArray(parseJSON(body), "data")
	require.Len(t, customers, 1, "the registration created one customer named %q", name)
	customerID := jsonField(customers[0].(map[string]any), "id")

	requireAuditMetadata(t, customerID, "customer", "create", "source", "registration")
}

func TestAuditCoverage_PasswordResetAudited(t *testing.T) {
	t.Parallel()

	_, userID, _, _ := covAuthPasswordsNewAccountMember(t, "e2e-audit-reset")
	since := time.Now().UTC().Add(-time.Second)

	status, body := covAuthPasswordsRawPost(t, covAuthPasswordsResetPath, map[string]any{
		"token":    covAuthPasswordsResetToken(t, userID),
		"password": "AuditReset123!",
	})
	requireStatus(t, 200, status, body)

	event := requireAuditMetadata(t, userID, "user", "update", "reason", "password_reset")
	occurredAt, err := time.Parse(time.RFC3339Nano, jsonField(event, "occurred_at"))
	require.NoError(t, err)
	assert.False(t, occurredAt.Before(since))
}
