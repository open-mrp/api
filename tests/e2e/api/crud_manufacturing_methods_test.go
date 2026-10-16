//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

const manufacturingMethodsPath = "/v1/operations/manufacturing-methods"

func methodOperationsPath(methodID string) string {
	return manufacturingMethodsPath + "/" + methodID + "/operations"
}

func methodMaterialsPath(methodID string) string {
	return manufacturingMethodsPath + "/" + methodID + "/materials"
}

// createDraftMethod creates a draft method for a fresh part and deletes it when the test ends, if it is still a draft.
func createDraftMethod(t *testing.T, body map[string]any) (method map[string]any, itemID string) {
	t.Helper()
	if _, ok := body["item_id"]; !ok {
		body["item_id"] = createPartItem(t, "e2e-mm-part")
	}
	status, raw, err := apiClient.Post(manufacturingMethodsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	method = parseJSON(raw)
	methodID := jsonField(method, "id")
	t.Cleanup(func() { apiClient.Delete(manufacturingMethodsPath + "/" + methodID) })
	return method, body["item_id"].(string)
}

// addOperation adds an operation with the least a caller must send, plus extra.
func addOperation(t *testing.T, methodID string, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{
		"work_center_id": SeedDepartmentID,
		"labor_time":     map[string]any{"value": "0.02", "unit_id": unitHour, "per_unit_id": unitEach},
		"labor_rate":     map[string]any{"amount": "20", "currency": "usd", "per_unit_id": unitHour},
	}
	for k, v := range extra {
		body[k] = v
	}
	status, raw, err := apiClient.Post(methodOperationsPath(methodID), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	return parseJSON(raw)
}

func addMaterial(t *testing.T, methodID, itemID string, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"item_id": itemID, "quantity": map[string]any{"value": "2", "unit_id": unitEach}}
	for k, v := range extra {
		body[k] = v
	}
	status, raw, err := apiClient.Post(methodMaterialsPath(methodID), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	return parseJSON(raw)
}

func methodAction(t *testing.T, methodID, verb string, wantStatus int) map[string]any {
	t.Helper()
	status, raw, err := apiClient.Post(manufacturingMethodsPath+"/"+methodID+"/actions/"+verb, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, wantStatus, status, raw)
	return parseJSON(raw)
}

// methodRows decodes a list response's rows.
func methodRows(list *ListResponse) []map[string]any {
	rows := make([]map[string]any, len(list.Data))
	for i, raw := range list.Data {
		rows[i] = parseJSON(raw)
	}
	return rows
}

func getMethod(t *testing.T, methodID string, include ...string) map[string]any {
	t.Helper()
	var q url.Values
	if len(include) > 0 {
		q = url.Values{"include": include}
	}
	status, raw, err := apiClient.GetListRaw(manufacturingMethodsPath+"/"+methodID, q)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	return parseJSON(raw)
}

func assertMethodQuantity(t *testing.T, q map[string]any, value, unitID, where string) {
	t.Helper()
	require.NotNil(t, q, "%s is set", where)
	assert.True(t, decimalClose(value, jsonField(q, "value")), "%s.value: want %s, got %s", where, value, jsonField(q, "value"))
	assert.Equal(t, unitID, jsonField(jsonObject(q, "unit"), "id"), "%s.unit", where)
}

func assertMethodRate(t *testing.T, r map[string]any, value, unitID, perUnitID, where string) {
	t.Helper()
	require.NotNil(t, r, "%s is set", where)
	assert.True(t, decimalClose(value, jsonField(r, "value")), "%s.value: want %s, got %s", where, value, jsonField(r, "value"))
	assert.Equal(t, unitID, jsonField(jsonObject(r, "unit"), "id"), "%s.unit", where)
	assert.Equal(t, perUnitID, jsonField(jsonObject(r, "per_unit"), "id"), "%s.per_unit", where)
}

func assertMethodUnitPrice(t *testing.T, p map[string]any, amount, perUnitID, where string) {
	t.Helper()
	require.NotNil(t, p, "%s is set", where)
	assert.True(t, decimalClose(amount, jsonField(p, "amount")), "%s.amount: want %s, got %s", where, amount, jsonField(p, "amount"))
	assert.Equal(t, "usd", jsonField(p, "currency"), "%s.currency", where)
	assert.Equal(t, perUnitID, jsonField(jsonObject(p, "per_unit"), "id"), "%s.per_unit", where)
}

// --- CRUD ---

func TestManufacturingMethods_CRUD(t *testing.T) {
	t.Parallel()
	method, itemID := createDraftMethod(t, map[string]any{})
	methodID := jsonField(method, "id")

	assertIDFormat(t, methodID, id.ManufacturingMethodIDPrefix)
	assertObjectField(t, method, "manufacturing_method")
	assert.Equal(t, "draft", jsonField(method, "status"))
	assert.Equal(t, "1", jsonField(method, "version"))
	assertMethodQuantity(t, jsonObject(method, "output_quantity"), "1", unitEach, "default output_quantity (one stocking unit)")
	assertValidTimestamp(t, jsonField(method, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(method, "updated_at"), "updated_at")

	got := getMethod(t, methodID, "item")
	assert.Equal(t, itemID, jsonField(jsonObject(got, "item"), "id"))

	status, raw, err := apiClient.Patch(manufacturingMethodsPath+"/"+methodID, map[string]any{
		"output_quantity": map[string]any{"value": "12", "unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertMethodQuantity(t, jsonObject(parseJSON(raw), "output_quantity"), "12", unitEach, "updated output_quantity")

	status, raw, err = apiClient.Delete(manufacturingMethodsPath + "/" + methodID)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertDeletedStub(t, raw, methodID, "manufacturing_method")

	status, raw, err = apiClient.GetListRaw(manufacturingMethodsPath+"/"+methodID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, raw)

	status, raw, err = apiClient.Delete(manufacturingMethodsPath + "/" + methodID)
	require.NoError(t, err)
	requireStatus(t, 410, status, raw)
}

func TestManufacturingMethods_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	got := getMethod(t, SeedSewLargeMethodID)
	for _, field := range []string{"item", "sales_order_line", "operations", "materials"} {
		assertNilField(t, got, field)
	}
	assert.Equal(t, "active", jsonField(got, "status"))

	got = getMethod(t, SeedSewLargeMethodID, "item", "operations", "materials")
	assert.Equal(t, SeedLsnItemID, jsonField(jsonObject(got, "item"), "id"))
	ops := jsonArray(jsonObject(got, "operations"), "data")
	require.Len(t, ops, 1, "the seeded step is the method's one operation")
	assert.Equal(t, SeedSewLargeProductionStepID, jsonField(ops[0].(map[string]any), "id"), "an operation keeps its step's id")
	assert.NotEmpty(t, jsonArray(jsonObject(got, "materials"), "data"), "the step's consumptions are the method's materials")
	assertNilField(t, got, "sales_order_line")
}

func TestManufacturingMethods_CreateValidation(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.Post(manufacturingMethodsPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{400, 422}, status, string(raw))

	status, raw, err = apiClient.Post(manufacturingMethodsPath, map[string]any{"item_id": "it_doesnotexist000"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "invalid_format", "invalid_request_error"), "item_id")

	status, raw, err = apiClient.Post(manufacturingMethodsPath, map[string]any{
		"item_id":         createPartItem(t, "e2e-mm-bad-qty"),
		"output_quantity": map[string]any{"value": "0", "unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "invalid_format", "invalid_request_error"), "output_quantity.value")
}

func TestManufacturingMethods_Idempotency(t *testing.T) {
	t.Parallel()
	itemID := createPartItem(t, "e2e-mm-idem")
	key := newIdempotencyKey()
	body := map[string]any{"item_id": itemID}
	status, raw, err := apiClient.Post(manufacturingMethodsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	first := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { apiClient.Delete(manufacturingMethodsPath + "/" + first) })

	status, raw, err = apiClient.Post(manufacturingMethodsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	assert.Equal(t, first, jsonField(parseJSON(raw), "id"), "a retry with the same key returns the same method")
}

// --- Operations ---

func TestMethodOperations_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	method, _ := createDraftMethod(t, map[string]any{})
	methodID := jsonField(method, "id")
	machineID := jsonField(createMachineAt(t, SeedDepartmentID, nil), "id")

	op := addOperation(t, methodID, map[string]any{
		"sequence":            "after_previous",
		"machine_ids":         []string{machineID},
		"scanning_station_id": seedKnittingStationID,
		"setup_time":          map[string]any{"value": "0.5", "unit_id": unitHour},
		"labor_time":          map[string]any{"value": "30", "unit_id": unitSecond, "per_unit_id": unitEach},
		"machine_time":        map[string]any{"value": "10", "unit_id": unitSecond, "per_unit_id": unitEach},
		"labor_rate":          map[string]any{"amount": "22.5", "currency": "usd", "per_unit_id": unitHour},
		"overhead_rate":       map[string]any{"amount": "5", "currency": "usd", "per_unit_id": unitHour},
		"leveling_percent":    "15",
		"allowance_percent":   "10",
		"work_instructions":   "Trim the threads.",
	})
	opID := jsonField(op, "id")
	assertIDFormat(t, opID, id.MethodOperationIDPrefix)
	assertObjectField(t, op, "method_operation")
	assert.Equal(t, "1", jsonField(op, "position"))
	assert.Equal(t, "after_previous", jsonField(op, "sequence"))
	assert.Equal(t, "inside", jsonField(op, "type"))
	assertMethodQuantity(t, jsonObject(op, "setup_time"), "0.5", unitHour, "setup_time")
	assertMethodRate(t, jsonObject(op, "labor_time"), "30", unitSecond, unitEach, "labor_time")
	assertMethodRate(t, jsonObject(op, "machine_time"), "10", unitSecond, unitEach, "machine_time")
	assertMethodUnitPrice(t, jsonObject(op, "labor_rate"), "22.5", unitHour, "labor_rate")
	assertMethodUnitPrice(t, jsonObject(op, "overhead_rate"), "5", unitHour, "overhead_rate")
	assert.Equal(t, "15", jsonField(op, "leveling_percent"))
	assert.Equal(t, "10", jsonField(op, "allowance_percent"))
	assert.Equal(t, "Trim the threads.", jsonField(op, "work_instructions"))
	for _, field := range []string{"work_center", "machines", "scanning_station"} {
		assertNilField(t, op, field)
	}
	assertValidTimestamp(t, jsonField(op, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(op, "updated_at"), "updated_at")

	status, raw, err := apiClient.GetListRaw(methodOperationsPath(methodID)+"/"+opID, url.Values{"include": {"work_center", "machines", "scanning_station"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	got := parseJSON(raw)
	assert.Equal(t, SeedDepartmentID, jsonField(jsonObject(got, "work_center"), "id"))
	assert.Equal(t, seedKnittingStationID, jsonField(jsonObject(got, "scanning_station"), "id"))
	machines := jsonArray(jsonObject(got, "machines"), "data")
	require.Len(t, machines, 1)
	machine := machines[0].(map[string]any)
	assert.Equal(t, machineID, jsonField(machine, "id"))
	assert.Equal(t, "active", jsonField(machine, "status"), "machines are loaded whole, status included")

	status, raw, err = apiClient.Patch(methodOperationsPath(methodID)+"/"+opID, map[string]any{
		"scanning_station_id": nil,
		"setup_time":          nil,
		"machine_time":        nil,
		"machine_ids":         []string{},
		"work_instructions":   nil,
		"leveling_percent":    "12.5",
		"sequence":            "with_previous",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	updated := parseJSON(raw)
	assertNilField(t, updated, "setup_time")
	assertNilField(t, updated, "machine_time")
	assertNilField(t, updated, "work_instructions")
	assert.Equal(t, "12.5", jsonField(updated, "leveling_percent"))
	assert.Equal(t, "10", jsonField(updated, "allowance_percent"), "unsent fields keep their values")
	assert.Equal(t, "with_previous", jsonField(updated, "sequence"))
	assertMethodRate(t, jsonObject(updated, "labor_time"), "30", unitSecond, unitEach, "labor_time kept")

	got = getWith(t, methodOperationsPath(methodID)+"/"+opID, "machines", "scanning_station")
	assertNilField(t, got, "scanning_station")
	assert.Empty(t, jsonArray(jsonObject(got, "machines"), "data"), "an empty list removes the machines")
}

func TestMethodOperations_DefaultsAndPositions(t *testing.T) {
	t.Parallel()
	wc := createWorkCenter(t, map[string]any{"labor_rate": map[string]any{"amount": "31", "currency": "usd", "per_unit_id": unitHour}})
	method, _ := createDraftMethod(t, map[string]any{})
	methodID := jsonField(method, "id")

	status, raw, err := apiClient.Post(methodOperationsPath(methodID), map[string]any{
		"work_center_id": jsonField(wc, "id"),
		"labor_time":     map[string]any{"value": "1", "unit_id": unitSecond, "per_unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	first := parseJSON(raw)
	assertMethodUnitPrice(t, jsonObject(first, "labor_rate"), "31", unitHour, "labor_rate defaults to the work center's")
	assertMethodUnitPrice(t, jsonObject(first, "overhead_rate"), "0", unitHour, "overhead_rate defaults to zero")
	assert.Equal(t, "0", jsonField(first, "leveling_percent"))
	assert.Equal(t, "0", jsonField(first, "allowance_percent"))
	assert.Equal(t, "after_previous", jsonField(first, "sequence"))
	for _, field := range []string{"setup_time", "machine_time", "work_instructions"} {
		assertNilField(t, first, field)
	}

	second := addOperation(t, methodID, nil)
	assert.Equal(t, "2", jsonField(second, "position"), "appended after the last")
	third := addOperation(t, methodID, map[string]any{"position": 1})
	assert.Equal(t, "1", jsonField(third, "position"), "inserted first")

	list, _, err := apiClient.GetList(methodOperationsPath(methodID), nil)
	require.NoError(t, err)
	var order []string
	for _, row := range methodRows(list) {
		order = append(order, jsonField(row, "id"))
	}
	assert.Equal(t, []string{jsonField(third, "id"), jsonField(first, "id"), jsonField(second, "id")}, order, "listed by position")

	// Without a labor rate of its own or its work center's, the caller must send one.
	status, raw, err = apiClient.Post(methodOperationsPath(methodID), map[string]any{
		"work_center_id": jsonField(createWorkCenter(t, map[string]any{}), "id"),
		"labor_time":     map[string]any{"value": "1", "unit_id": unitSecond, "per_unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "missing_field", "invalid_request_error"), "labor_rate")

	// Labor time is time per unit of output.
	status, raw, err = apiClient.Post(methodOperationsPath(methodID), map[string]any{
		"work_center_id": SeedDepartmentID,
		"labor_time":     map[string]any{"value": "1", "unit_id": unitEach, "per_unit_id": unitEach},
		"labor_rate":     map[string]any{"amount": "20", "currency": "usd", "per_unit_id": unitHour},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "invalid_format", "invalid_request_error"), "labor_time.unit_id")
}

func TestMethodOperations_ArchivedMachineRejected(t *testing.T) {
	t.Parallel()
	archived := jsonField(createMachineAt(t, SeedDepartmentID, map[string]any{"status": "archived"}), "id")
	method, _ := createDraftMethod(t, map[string]any{})
	status, raw, err := apiClient.Post(methodOperationsPath(jsonField(method, "id")), map[string]any{
		"work_center_id": SeedDepartmentID,
		"machine_ids":    []string{archived},
		"labor_time":     map[string]any{"value": "1", "unit_id": unitSecond, "per_unit_id": unitEach},
		"labor_rate":     map[string]any{"amount": "20", "currency": "usd", "per_unit_id": unitHour},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "resource_archived", "invalid_request_error"), "machine_ids")
}

// A machine an operation runs on can't be deleted out from under it.
func TestMethodOperations_MachineInUse(t *testing.T) {
	t.Parallel()
	machineID := jsonField(createMachineAt(t, SeedDepartmentID, nil), "id")
	method, _ := createDraftMethod(t, map[string]any{})
	addOperation(t, jsonField(method, "id"), map[string]any{"machine_ids": []string{machineID}})

	status, raw, err := apiClient.Delete(machinesPath + "/" + machineID)
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	requireErrorResponse(t, raw, "resource_in_use", "invalid_request_error")
}

// --- Materials ---

func TestMethodMaterials_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	method, _ := createDraftMethod(t, map[string]any{})
	methodID := jsonField(method, "id")
	op1 := jsonField(addOperation(t, methodID, nil), "id")
	op2 := jsonField(addOperation(t, methodID, nil), "id")
	consumed := createPartItem(t, "e2e-mm-consumed")

	mat := addMaterial(t, methodID, consumed, map[string]any{
		"operation_id":      op2,
		"scrap_quantity":    map[string]any{"value": "0.1", "unit_id": unitEach},
		"supply":            "buy",
		"work_instructions": "Keep dry.",
	})
	matID := jsonField(mat, "id")
	assertIDFormat(t, matID, id.MethodMaterialIDPrefix)
	assertObjectField(t, mat, "method_material")
	assertMethodQuantity(t, jsonObject(mat, "quantity"), "2", unitEach, "quantity")
	assertMethodQuantity(t, jsonObject(mat, "scrap_quantity"), "0.1", unitEach, "scrap_quantity")
	assert.Equal(t, "buy", jsonField(mat, "supply"))
	assert.Equal(t, "Keep dry.", jsonField(mat, "work_instructions"))
	assertNilField(t, mat, "item")
	assertNilField(t, mat, "operation")
	assertValidTimestamp(t, jsonField(mat, "created_at"), "created_at")

	got := getWith(t, methodMaterialsPath(methodID)+"/"+matID, "item", "operation")
	assert.Equal(t, consumed, jsonField(jsonObject(got, "item"), "id"))
	assert.Equal(t, op2, jsonField(jsonObject(got, "operation"), "id"))

	status, raw, err := apiClient.Patch(methodMaterialsPath(methodID)+"/"+matID, map[string]any{
		"operation_id":      op1,
		"quantity":          map[string]any{"value": "3", "unit_id": unitEach},
		"work_instructions": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	updated := parseJSON(raw)
	assertMethodQuantity(t, jsonObject(updated, "quantity"), "3", unitEach, "updated quantity")
	assertMethodQuantity(t, jsonObject(updated, "scrap_quantity"), "0.1", unitEach, "scrap kept")
	assert.Equal(t, "buy", jsonField(updated, "supply"), "supply kept")
	assertNilField(t, updated, "work_instructions")

	// An operation that consumes something can't go until its materials move.
	status, raw, err = apiClient.Delete(methodOperationsPath(methodID) + "/" + op1)
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	requireErrorResponse(t, raw, "resource_in_use", "invalid_request_error")

	status, raw, err = apiClient.Delete(methodMaterialsPath(methodID) + "/" + matID)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertDeletedStub(t, raw, matID, "method_material")

	status, raw, err = apiClient.Delete(methodOperationsPath(methodID) + "/" + op1)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertDeletedStub(t, raw, op1, "method_operation")

	ops, _, err := apiClient.GetList(methodOperationsPath(methodID), nil)
	require.NoError(t, err)
	require.Len(t, ops.Data, 1)
	assert.Equal(t, "1", jsonField(methodRows(ops)[0], "position"), "positions close up after a delete")
}

func TestMethodMaterials_Defaults(t *testing.T) {
	t.Parallel()
	method, itemID := createDraftMethod(t, map[string]any{})
	methodID := jsonField(method, "id")

	status, raw, err := apiClient.Post(methodMaterialsPath(methodID), map[string]any{
		"item_id": createPartItem(t, "e2e-mm-noop"), "quantity": map[string]any{"value": "1", "unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "missing_field", "invalid_request_error"), "operation_id")

	firstOp := jsonField(addOperation(t, methodID, nil), "id")
	addOperation(t, methodID, nil)
	mat := addMaterial(t, methodID, createPartItem(t, "e2e-mm-stock"), nil)
	assertMethodQuantity(t, jsonObject(mat, "scrap_quantity"), "0", unitEach, "scrap defaults to zero")
	assert.Equal(t, "stock", jsonField(mat, "supply"), "an item nothing makes is stocked")
	assertNilField(t, mat, "work_instructions")
	got := getWith(t, methodMaterialsPath(methodID)+"/"+jsonField(mat, "id"), "operation")
	assert.Equal(t, firstOp, jsonField(jsonObject(got, "operation"), "id"), "consumed at the first operation by default")

	made := addMaterial(t, methodID, SeedPartItemID, nil)
	assert.Equal(t, "make", jsonField(made, "supply"), "an item with an active method is made")

	status, raw, err = apiClient.Post(methodMaterialsPath(methodID), map[string]any{
		"item_id": itemID, "quantity": map[string]any{"value": "1", "unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "invalid_format", "invalid_request_error"), "item_id")
}

// --- Versions ---

// A draft is activated into the item's method; editing it means a new version, whose activation archives the old.
// Only the active version reaches execution: the preview.9 production step routes see its operation, not the old one.
func TestManufacturingMethods_VersionLifecycle(t *testing.T) {
	t.Parallel()
	method, itemID := createDraftMethod(t, map[string]any{})
	v1 := jsonField(method, "id")
	consumed := createPartItem(t, "e2e-mm-lifecycle-in")

	methodAction(t, v1, "activate", 409)

	op := addOperation(t, v1, nil)
	addMaterial(t, v1, consumed, nil)
	activated := methodAction(t, v1, "activate", 200)
	assert.Equal(t, "active", jsonField(activated, "status"))
	methodAction(t, v1, "activate", 409)

	status, raw, err := apiClient.Patch(manufacturingMethodsPath+"/"+v1, map[string]any{
		"output_quantity": map[string]any{"value": "2", "unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	requireErrorResponse(t, raw, "invalid_state_transition", "invalid_request_error")

	status, raw, err = apiClient.Post(methodOperationsPath(v1), map[string]any{
		"work_center_id": SeedDepartmentID,
		"labor_time":     map[string]any{"value": "1", "unit_id": unitSecond, "per_unit_id": unitEach},
		"labor_rate":     map[string]any{"amount": "20", "currency": "usd", "per_unit_id": unitHour},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)

	status, raw, err = apiClient.Delete(manufacturingMethodsPath + "/" + v1)
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)

	item := getItem(t, itemID, "default_manufacturing_method")
	assert.Equal(t, v1, jsonField(jsonObject(item, "default_manufacturing_method"), "id"))

	stepStatus, stepRaw, err := apiClient.GetListRaw(productionStepsPath+"/"+jsonField(op, "id"), nil)
	require.NoError(t, err)
	requireStatus(t, 200, stepStatus, stepRaw)

	draft := methodAction(t, v1, "new-version", 200)
	v2 := jsonField(draft, "id")
	t.Cleanup(func() { apiClient.Delete(manufacturingMethodsPath + "/" + v2) })
	assert.Equal(t, "draft", jsonField(draft, "status"))
	assert.Equal(t, "2", jsonField(draft, "version"))
	copied := getMethod(t, v2, "operations", "materials")
	ops := jsonArray(jsonObject(copied, "operations"), "data")
	require.Len(t, ops, 1, "the new version copies the operations")
	assert.NotEqual(t, jsonField(op, "id"), jsonField(ops[0].(map[string]any), "id"))
	require.Len(t, jsonArray(jsonObject(copied, "materials"), "data"), 1, "and the materials")

	// A draft's operation isn't a production step yet.
	stepStatus, _, err = apiClient.GetListRaw(productionStepsPath+"/"+jsonField(ops[0].(map[string]any), "id"), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, stepStatus)

	methodAction(t, v2, "activate", 200)
	assert.Equal(t, "archived", jsonField(getMethod(t, v1), "status"))
	item = getItem(t, itemID, "default_manufacturing_method")
	assert.Equal(t, v2, jsonField(jsonObject(item, "default_manufacturing_method"), "id"))

	stepStatus, _, err = apiClient.GetListRaw(productionStepsPath+"/"+jsonField(ops[0].(map[string]any), "id"), nil)
	require.NoError(t, err)
	assert.Equal(t, 200, stepStatus, "the active version's operation is a production step")
	stepStatus, _, err = apiClient.GetListRaw(productionStepsPath+"/"+jsonField(op, "id"), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, stepStatus, "the archived version's operation no longer is")

	list, _, err := apiClient.GetList(manufacturingMethodsPath, url.Values{"item_ids[]": {itemID}})
	require.NoError(t, err)
	require.Len(t, list.Data, 2)
	assert.Equal(t, v2, jsonField(methodRows(list)[0], "id"), "newest first")
	list, _, err = apiClient.GetList(manufacturingMethodsPath, url.Values{"item_ids[]": {itemID}, "statuses[]": {"archived"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, v1, jsonField(methodRows(list)[0], "id"))
}

// --- Lists ---

func TestManufacturingMethods_ListFilters(t *testing.T) {
	t.Parallel()
	wc := jsonField(createWorkCenter(t, map[string]any{}), "id")
	method, itemID := createDraftMethod(t, map[string]any{})
	methodID := jsonField(method, "id")
	addOperation(t, methodID, map[string]any{"work_center_id": wc})

	list, _, err := apiClient.GetList(manufacturingMethodsPath, url.Values{"work_center_ids[]": {wc}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, methodID, jsonField(methodRows(list)[0], "id"))

	list, _, err = apiClient.GetList(manufacturingMethodsPath, url.Values{"item_ids[]": {itemID}, "statuses[]": {"active"}})
	require.NoError(t, err)
	assert.Empty(t, list.Data, "a draft isn't active")

	status, raw, err := apiClient.GetListRaw(manufacturingMethodsPath, url.Values{"statuses[]": {"retired"}})
	require.NoError(t, err)
	assert.Contains(t, []int{400, 422}, status, string(raw))

	list, _, err = apiClient.GetList(manufacturingMethodsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	require.NotNil(t, list.PageInfo.NextPageURL, "another page exists")
}

// --- Production flow and adoption ---

// An item's flow is its active method, then the methods of what it makes from, down to stocked materials.
func TestManufacturingMethods_ProductionFlow(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.GetListRaw(itemsPath+"/"+SeedLsnItemID+"/production-flow", url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	flow := jsonArray(parseJSON(raw), "data")
	require.GreaterOrEqual(t, len(flow), 2, "the sewn sock's method, then the knitted sock's: %s", raw)
	first := flow[0].(map[string]any)
	assert.Equal(t, SeedSewLargeMethodID, jsonField(first, "id"))
	assert.Equal(t, SeedLsnItemID, jsonField(jsonObject(first, "item"), "id"))
	assert.Equal(t, SeedPartItemID, jsonField(jsonObject(flow[1].(map[string]any), "item"), "id"))

	status, raw, err = apiClient.GetListRaw(itemsPath+"/"+createPartItem(t, "e2e-mm-noflow")+"/production-flow", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Empty(t, jsonArray(parseJSON(raw), "data"), "an item with no method has no flow")
}

// A step written through the preview.9 routes is the single operation of an active method for what it makes.
func TestManufacturingMethods_PreviewStepIsAMethod(t *testing.T) {
	t.Parallel()
	produced := createPartItem(t, "e2e-mm-step-out")
	consumed := createPartItem(t, "e2e-mm-step-in")
	stepID := createLinkedStep(t, "e2e-mm-step", produced, consumed)

	list, _, err := apiClient.GetList(manufacturingMethodsPath, url.Values{"item_ids[]": {produced}, "include": {"operations", "materials"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1, "the step's method")
	method := methodRows(list)[0]
	assert.Equal(t, "active", jsonField(method, "status"))
	ops := jsonArray(jsonObject(method, "operations"), "data")
	require.Len(t, ops, 1)
	assert.Equal(t, stepID, jsonField(ops[0].(map[string]any), "id"))
	mats := jsonArray(jsonObject(method, "materials"), "data")
	require.Len(t, mats, 1)
	assert.Equal(t, "stock", jsonField(mats[0].(map[string]any), "supply"))

	status, raw, err := apiClient.Delete(productionStepsPath + "/" + stepID)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	list, _, err = apiClient.GetList(manufacturingMethodsPath, url.Values{"item_ids[]": {produced}})
	require.NoError(t, err)
	assert.Empty(t, list.Data, "deleting the step takes its method with it")
}
