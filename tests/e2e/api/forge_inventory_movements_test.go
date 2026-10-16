//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Inventory movements, levels and lot-size defaults on 1.0.forge. Every test moves stock of a material it created, so
// no seeded figure another test reads changes.

const (
	movementsPath       = "/v1/operations/inventory-movements"
	inventoryLevelsPath = "/v1/operations/inventory-levels"
)

// newStockedMaterial creates a material stocked in pounds and returns its item id.
func newStockedMaterial(t *testing.T) string {
	t.Helper()
	return jsonField(createForgeItem(t, itemsPath, forgeItemBody("material", uniqueName("e2e-mv"))), "id")
}

func postMovement(t *testing.T, body map[string]any, key string) (int, map[string]any, []byte) {
	t.Helper()
	status, raw, err := apiClient.Post(movementsPath, body, key)
	require.NoError(t, err)
	require.Less(t, status, 500, "movement create must not 5xx: %s", string(raw))
	return status, parseJSON(raw), raw
}

func adjustBody(itemID, value string) map[string]any {
	return map[string]any{
		"item_id":   itemID,
		"operation": "adjust",
		"quantity":  map[string]any{"value": value, "unit_id": SeedMaterialUnitID},
	}
}

// itemLevel reads an item's inventory level on forge.
func itemLevel(t *testing.T, itemID string, params url.Values) map[string]any {
	t.Helper()
	status, raw, err := apiClient.Exact().GetListRaw(itemsPath+"/"+itemID+"/inventory", params)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	return parseJSON(raw)
}

// eventuallyOnHand waits for an item's on-hand level to settle at want: stock taken out is an issue that allocation
// draws against the item's receipts just after the movement commits.
func eventuallyOnHand(t *testing.T, itemID, want string) map[string]any {
	t.Helper()
	var level map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		level = itemLevel(t, itemID, nil)
		if got := jsonField(jsonObject(level, "on_hand"), "value"); got != want {
			return fmt.Errorf("on_hand %s, want %s", got, want)
		}
		return nil
	})
	return level
}

// --- Create ---

func TestInventoryMovements_CreateAllFields(t *testing.T) {
	t.Parallel()
	itemID := newStockedMaterial(t)
	location := createLocation(t, map[string]any{"name": uniqueName("e2e-mv-bin"), "type": "bin"})
	lotNumber := uniqueName("LOT")

	body := adjustBody(itemID, "12.5")
	body["location_id"] = jsonField(location, "id")
	body["lot_number"] = lotNumber
	body["reason"] = "found"
	status, got, raw := postMovement(t, body, newIdempotencyKey())
	requireStatus(t, 201, status, raw)

	assert.True(t, strings.HasPrefix(jsonField(got, "id"), "inmv_"), "movement id prefix: %s", jsonField(got, "id"))
	assertObjectField(t, got, "inventory_movement")
	assert.Equal(t, "adjustment", jsonField(got, "type"))
	quantity := jsonObject(got, "quantity")
	require.NotNil(t, quantity)
	assert.Equal(t, "12.5", jsonField(quantity, "value"))
	assert.Equal(t, SeedMaterialUnitID, jsonField(jsonObject(quantity, "unit"), "id"))
	assert.Equal(t, "found", jsonField(got, "reason"))
	actor := jsonObject(got, "actor")
	require.NotNil(t, actor)
	assert.Equal(t, "api_key", jsonField(actor, "type"))
	assert.NotEmpty(t, jsonField(actor, "id"))
	assert.Contains(t, got, "unit_cost")
	assertValidTimestamp(t, jsonField(got, "occurred_at"), "occurred_at")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	for _, f := range []string{"item", "location", "lot", "related"} {
		assertNilField(t, got, f)
	}

	t.Run("includes", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(movementsPath+"/"+jsonField(got, "id"), url.Values{"include": {"item,location,lot,related"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		m := parseJSON(raw)
		assert.Equal(t, itemID, jsonField(jsonObject(m, "item"), "id"))
		assert.Equal(t, jsonField(location, "id"), jsonField(jsonObject(m, "location"), "id"))
		lot := jsonObject(m, "lot")
		require.NotNil(t, lot, "lot_number creates the lot: %s", string(raw))
		assert.Equal(t, lotNumber, jsonField(lot, "lot_number"))
		assertNilField(t, m, "related")

		// The lot now exists, so a movement can name it by id.
		byID := adjustBody(itemID, "-2.5")
		byID["lot_id"] = jsonField(lot, "id")
		status, again, raw := postMovement(t, byID, newIdempotencyKey())
		requireStatus(t, 201, status, raw)
		assert.Equal(t, "-2.5", jsonField(jsonObject(again, "quantity"), "value"))
		assertNilField(t, again, "reason")
	})

	t.Run("level follows", func(t *testing.T) {
		level := eventuallyOnHand(t, itemID, "10")
		assertObjectField(t, level, "inventory_level")
		assert.Equal(t, "0", jsonField(jsonObject(level, "reserved"), "value"))
		assert.Equal(t, "10", jsonField(jsonObject(level, "available_to_promise"), "value"))
		assert.Equal(t, "0", jsonField(jsonObject(level, "short"), "value"))
		assertNilField(t, level, "item")
		assertNilField(t, level, "location")
	})
}

func TestInventoryMovements_Reconcile(t *testing.T) {
	t.Parallel()
	itemID := newStockedMaterial(t)

	status, _, raw := postMovement(t, adjustBody(itemID, "30"), newIdempotencyKey())
	requireStatus(t, 201, status, raw)

	count := map[string]any{"item_id": itemID, "operation": "reconcile", "quantity": map[string]any{"value": "24", "unit_id": SeedMaterialUnitID}, "reason": "count_correction"}
	status, got, raw := postMovement(t, count, newIdempotencyKey())
	requireStatus(t, 201, status, raw)
	assert.Equal(t, "reconciliation", jsonField(got, "type"))
	assert.Equal(t, "-6", jsonField(jsonObject(got, "quantity"), "value"), "a count records the difference it made")
	eventuallyOnHand(t, itemID, "24")

	// Counting what's already there still records the count, as a movement of nothing.
	status, same, raw := postMovement(t, count, newIdempotencyKey())
	requireStatus(t, 201, status, raw)
	assert.Equal(t, "0", jsonField(jsonObject(same, "quantity"), "value"))
	assert.Equal(t, "24", jsonField(jsonObject(itemLevel(t, itemID, nil), "on_hand"), "value"))
}

func TestInventoryMovements_Idempotency(t *testing.T) {
	t.Parallel()
	itemID := newStockedMaterial(t)
	key := newIdempotencyKey()
	status, first, raw := postMovement(t, adjustBody(itemID, "5"), key)
	requireStatus(t, 201, status, raw)
	status, second, raw := postMovement(t, adjustBody(itemID, "5"), key)
	requireStatus(t, 201, status, raw)
	assert.Equal(t, jsonField(first, "id"), jsonField(second, "id"))
	assert.Equal(t, "5", jsonField(jsonObject(itemLevel(t, itemID, nil), "on_hand"), "value"), "a replay moves nothing")
}

func TestInventoryMovements_Validation(t *testing.T) {
	t.Parallel()
	itemID := newStockedMaterial(t)

	t.Run("adjusting by zero", func(t *testing.T) {
		status, _, raw := postMovement(t, adjustBody(itemID, "0"), newIdempotencyKey())
		requireStatus(t, 422, status, raw)
	})
	t.Run("negative count", func(t *testing.T) {
		body := adjustBody(itemID, "-1")
		body["operation"] = "reconcile"
		status, _, raw := postMovement(t, body, newIdempotencyKey())
		requireStatus(t, 422, status, raw)
	})
	t.Run("lot id and number", func(t *testing.T) {
		body := adjustBody(itemID, "1")
		body["lot_id"] = "lot_nope"
		body["lot_number"] = "L1"
		status, _, raw := postMovement(t, body, newIdempotencyKey())
		requireStatus(t, 422, status, raw)
	})
	t.Run("unknown lot", func(t *testing.T) {
		body := adjustBody(itemID, "1")
		body["lot_id"] = "lot_doesnotexist01"
		status, _, raw := postMovement(t, body, newIdempotencyKey())
		requireStatus(t, 422, status, raw)
	})
	t.Run("archived location", func(t *testing.T) {
		archived := createLocation(t, map[string]any{"name": uniqueName("e2e-mv-arch"), "type": "rack", "status": "archived"})
		body := adjustBody(itemID, "1")
		body["location_id"] = jsonField(archived, "id")
		status, _, raw := postMovement(t, body, newIdempotencyKey())
		requireStatus(t, 422, status, raw)
		assertErrorParam(t, requireErrorResponse(t, raw, "resource_archived", "invalid_request_error"), "location_id")
	})
	t.Run("unit outside the item's units", func(t *testing.T) {
		body := adjustBody(itemID, "1")
		body["quantity"] = map[string]any{"value": "1", "unit_id": SeedUnitID}
		status, _, raw := postMovement(t, body, newIdempotencyKey())
		requireStatus(t, 422, status, raw)
	})
	t.Run("service item", func(t *testing.T) {
		service := createForgeItem(t, itemsPath, forgeItemBody("service", uniqueName("e2e-mv-svc")))
		status, _, raw := postMovement(t, adjustBody(jsonField(service, "id"), "1"), newIdempotencyKey())
		requireStatus(t, 422, status, raw)
	})
	t.Run("missing operation", func(t *testing.T) {
		body := adjustBody(itemID, "1")
		delete(body, "operation")
		status, _, raw := postMovement(t, body, newIdempotencyKey())
		requireClientRejection(t, status, raw)
	})
	t.Run("bad reason", func(t *testing.T) {
		body := adjustBody(itemID, "1")
		body["reason"] = "stolen"
		status, _, raw := postMovement(t, body, newIdempotencyKey())
		requireClientRejection(t, status, raw)
	})
}

// --- List ---

func TestInventoryMovements_ListFilters(t *testing.T) {
	t.Parallel()
	itemID := newStockedMaterial(t)
	location := createLocation(t, map[string]any{"name": uniqueName("e2e-mv-list"), "type": "bin"})
	in := adjustBody(itemID, "8")
	in["location_id"] = jsonField(location, "id")
	status, adjusted, raw := postMovement(t, in, newIdempotencyKey())
	requireStatus(t, 201, status, raw)
	count := adjustBody(itemID, "3")
	count["operation"] = "reconcile"
	status, counted, raw := postMovement(t, count, newIdempotencyKey())
	requireStatus(t, 201, status, raw)

	ids := func(params url.Values) []string {
		list, status, err := apiClient.GetList(movementsPath, params)
		require.NoError(t, err)
		requireStatus(t, 200, status, nil)
		var out []string
		for _, d := range list.Data {
			out = append(out, DataItemField(d, "id"))
		}
		return out
	}

	// The item's first movement is the zero opening balance OpenMRP records when an item is created.
	all := ids(url.Values{"item_ids[]": {itemID}})
	require.Len(t, all, 3)
	assert.Equal(t, []string{jsonField(counted, "id"), jsonField(adjusted, "id")}, all[:2], "newest first")
	assert.Equal(t, []string{all[2]}, ids(url.Values{"item_ids[]": {itemID}, "actor_types[]": {"system"}}))
	assert.Equal(t, []string{jsonField(counted, "id")}, ids(url.Values{"item_ids[]": {itemID}, "types[]": {"reconciliation"}}))
	assert.Equal(t, []string{jsonField(adjusted, "id")}, ids(url.Values{"location_ids[]": {jsonField(location, "id")}}))
	assert.Equal(t, all[:2], ids(url.Values{"item_ids[]": {itemID}, "actor_types[]": {"api_key"}}))
	assert.Empty(t, ids(url.Values{"item_ids[]": {itemID}, "actor_types[]": {"device"}}))
	actorID := jsonField(jsonObject(adjusted, "actor"), "id")
	assert.Equal(t, all[:2], ids(url.Values{"actor_ids[]": {actorID}, "item_ids[]": {itemID}}))

	t.Run("bad type is rejected", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(movementsPath, url.Values{"types[]": {"scan"}})
		require.NoError(t, err)
		requireClientRejection(t, status, raw)
	})
	t.Run("retrieve unknown", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(movementsPath+"/inmv_doesnotexist0", nil)
		require.NoError(t, err)
		requireStatus(t, 404, status, raw)
	})
}

// A change log written before movements stored a type and actor reads them from what it did record.
func TestInventoryMovements_LegacyEntriesReadAsMovements(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.GetListRaw(movementsPath+"/"+SeedInventoryChangeLogID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	got := parseJSON(raw)
	assert.Equal(t, "adjustment", jsonField(got, "type"), "a user action is an adjustment")
	actor := jsonObject(got, "actor")
	require.NotNil(t, actor)
	assert.Equal(t, "device", jsonField(actor, "type"), "it was recorded at a scanning station")
	assert.Equal(t, "sgsn_01k0a8201zegarjfsjaw5n7yfv", jsonField(actor, "id"))

	status, raw, err = apiClient.GetListRaw(movementsPath+"/inmv_01seedmvfull0000", url.Values{"include": {"related,lot,location"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	full := parseJSON(raw)
	assert.Equal(t, "found", jsonField(full, "reason"))
	assert.Equal(t, "system", jsonField(jsonObject(full, "actor"), "type"))
	assertNilField(t, jsonObject(full, "actor"), "id")
	related := jsonObject(full, "related")
	require.NotNil(t, related)
	assert.Equal(t, "record", jsonField(related, "object"))
	assert.Equal(t, "shipment", jsonField(related, "type"))
	assert.Equal(t, SeedShipmentID, jsonField(related, "id"))
	assert.Equal(t, "LOT-DLV-001", jsonField(jsonObject(full, "lot"), "lot_number"))
	assert.Equal(t, SeedLocationID, jsonField(jsonObject(full, "location"), "id"))
}

// --- Bulk ---

func TestInventoryMovements_BulkCreate(t *testing.T) {
	t.Parallel()
	itemID := newStockedMaterial(t)
	status, item, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, item)
	sku := jsonField(parseJSON(item), "sku")

	status, raw, err := apiClient.Post(movementsPath+"/actions/bulk-create", map[string]any{
		"operation": "reconcile",
		"movements": []map[string]any{
			{"sku": sku, "quantity": map[string]any{"value": "40", "unit_id": SeedMaterialUnitID}, "reason": "count_correction"},
			{"sku": uniqueName("e2e-no-such-sku"), "quantity": map[string]any{"value": "1", "unit_id": SeedMaterialUnitID}},
			{"item_id": itemID, "quantity": map[string]any{"value": "35", "unit_id": SeedMaterialUnitID}},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, raw)
	job := pollJobUntilTerminal(t, jsonField(parseJSON(raw), "id"))
	require.Equal(t, "completed", jsonField(job, "status"), "job: %v", job)

	results, _ := job["results"].(map[string]any)["data"].([]any)
	require.Len(t, results, 3, "every row is accounted for: %v", job)
	statuses := map[string]int{}
	for _, r := range results {
		statuses[r.(map[string]any)["status"].(string)]++
	}
	assert.Equal(t, map[string]int{"created": 2, "failed": 1}, statuses)
	eventuallyOnHand(t, itemID, "35")

	t.Run("rows name exactly one item", func(t *testing.T) {
		status, raw, err := apiClient.Post(movementsPath+"/actions/bulk-create", map[string]any{
			"operation": "adjust",
			"movements": []map[string]any{{"quantity": map[string]any{"value": "1", "unit_id": SeedMaterialUnitID}}},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireClientRejection(t, status, raw)
	})
}

func TestInventoryMovements_Export(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.Post(movementsPath+"/actions/export", map[string]any{"types": []string{"adjustment"}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, raw)
	job := pollJobUntilTerminal(t, jsonField(parseJSON(raw), "id"))
	assert.Equal(t, "completed", jsonField(job, "status"), "job: %v", job)
}

// --- Levels ---

func TestInventoryLevels_List(t *testing.T) {
	t.Parallel()
	itemID := newStockedMaterial(t)
	location := createLocation(t, map[string]any{"name": uniqueName("e2e-lvl-bin"), "type": "bin"})
	at := adjustBody(itemID, "7")
	at["location_id"] = jsonField(location, "id")
	status, _, raw := postMovement(t, at, newIdempotencyKey())
	requireStatus(t, 201, status, raw)
	status, _, raw = postMovement(t, adjustBody(itemID, "3"), newIdempotencyKey())
	requireStatus(t, 201, status, raw)

	list := func(params url.Values) map[string]any {
		status, raw, err := apiClient.GetListRaw(inventoryLevelsPath, params)
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		data := parseJSON(raw)["data"].([]any)
		require.Len(t, data, 1, "one level per item: %s", string(raw))
		return data[0].(map[string]any)
	}

	total := list(url.Values{"item_ids[]": {itemID}, "include": {"item"}})
	assertObjectField(t, total, "inventory_level")
	assert.Equal(t, itemID, jsonField(jsonObject(total, "item"), "id"))
	assertNilField(t, total, "location")
	assert.Equal(t, "10", jsonField(jsonObject(total, "on_hand"), "value"))

	atBin := list(url.Values{"item_ids[]": {itemID}, "location_ids[]": {jsonField(location, "id")}, "include": {"location"}})
	assert.Equal(t, "7", jsonField(jsonObject(atBin, "on_hand"), "value"))
	assert.Equal(t, jsonField(location, "id"), jsonField(jsonObject(atBin, "location"), "id"))

	past := list(url.Values{"item_ids[]": {itemID}, "as_of": {"2100-01-01T00:00:00Z"}})
	assert.Equal(t, "10", jsonField(jsonObject(past, "on_hand"), "value"))
	assertNilField(t, past, "reserved")
	assertNilField(t, past, "available_to_promise")
	assertNilField(t, past, "short")

	t.Run("as_of with locations is rejected", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(inventoryLevelsPath, url.Values{"as_of": {"2100-01-01T00:00:00Z"}, "location_ids[]": {jsonField(location, "id")}})
		require.NoError(t, err)
		requireStatus(t, 400, status, raw)
	})
	t.Run("services have no level", func(t *testing.T) {
		service := createForgeItem(t, itemsPath, forgeItemBody("service", uniqueName("e2e-lvl-svc")))
		status, raw, err := apiClient.Exact().GetListRaw(itemsPath+"/"+jsonField(service, "id")+"/inventory", nil)
		require.NoError(t, err)
		requireStatus(t, 404, status, raw)
	})
}

// --- Lot-size default ---

func TestLotSizeDefault_Retrieve(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.GetListRaw(itemsPath+"/"+SeedMaterialItemID+"/lot-size-default", url.Values{"include": {"item,item_category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	got := parseJSON(raw)
	assertObjectField(t, got, "lot_size_default")
	assert.Equal(t, SeedMaterialItemID, jsonField(jsonObject(got, "item"), "id"))
	assert.Contains(t, got, "lot_size")
	assert.Contains(t, got, "source")
	if jsonField(got, "source") == "item_category" {
		assert.NotNil(t, jsonObject(got, "item_category"))
	} else {
		assertNilField(t, got, "item_category")
	}

	status, raw, err = apiClient.GetListRaw(itemsPath+"/"+SeedMaterialItemID+"/lot-size-default", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertNilField(t, parseJSON(raw), "item")
}

// --- Version compatibility ---

func TestVersionCompatInventory_Preview9(t *testing.T) {
	t.Parallel()
	itemID := newStockedMaterial(t)
	status, _, raw := postMovement(t, adjustBody(itemID, "4"), newIdempotencyKey())
	requireStatus(t, 201, status, raw)
	preview9 := apiClient.WithAPIVersion(preview9Version)

	t.Run("item inventory keeps its preview.9 shape", func(t *testing.T) {
		status, raw, err := preview9.GetListRaw(itemsPath+"/"+itemID+"/inventory", url.Values{"include": {"on_hand"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		got := parseJSON(raw)
		assertObjectField(t, got, "item_inventory")
		assert.Equal(t, "4", jsonField(jsonObject(got, "on_hand"), "value"))
	})
	t.Run("change logs list the movement", func(t *testing.T) {
		list, status, err := preview9.GetList("/v1/operations/inventory-change-logs", url.Values{"item_ids": {itemID}})
		require.NoError(t, err)
		requireStatus(t, 200, status, nil)
		require.NotEmpty(t, list.Data)
		assert.Equal(t, "user_correction", DataItemField(list.Data[0], "action_type"))
	})
	t.Run("retired routes are gone on forge", func(t *testing.T) {
		for _, path := range []string{"/v1/operations/inventory-change-logs", "/v1/operations/inventories", itemsPath + "/" + itemID + "/lot-default"} {
			status, raw, err := apiClient.Exact().GetListRaw(path, nil)
			require.NoError(t, err)
			assert.Equal(t, 404, status, "%s: %s", path, string(raw))
		}
	})
}
