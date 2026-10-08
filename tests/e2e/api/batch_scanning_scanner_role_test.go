//go:build e2e

package api_test

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Operators on the floor scan under the built-in scanner role. It reads no units, items, departments,
// steps or runs, so everything a scanning station shows has to reach it through the batches and
// stations it may read: each read here answers the operator exactly as it answers an admin.

// scannerRolePermissions are the built-in scanner role's permissions, as permission code and the
// create/read/update/delete flags it holds.
var scannerRolePermissions = map[string]string{
	"batches":               "crud",
	"inventory":             "ru",
	"inventory_change_logs": "r",
	"inventory_logs":        "r",
	"jobs":                  "r",
	"messaging":             "cru",
	"scanners":              "r",
	"self":                  "r",
}

// stationBatchIncludes are the expansions the scanning station asks for on every batch it shows.
const stationBatchIncludes = "quantity.unit,seconds.unit,waste.unit"

// scannerOperator is a person signed in to the seed account under a role holding exactly the scanner
// role's permissions. Role assignment has no API: an operator grants it in SQL, as this does.
func scannerOperator(t *testing.T) *Client {
	t.Helper()
	db := authDB(t)
	suffix := uuid.New().String()[:12]
	roleID := "rl_e2escanner_" + suffix
	_, err := db.Exec(`INSERT INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES (?, ?, 'user', ?, NOW(3), NOW(3))`,
		roleID, "E2E scanner "+suffix, SeedAccountID)
	require.NoError(t, err)
	i := 0
	for code, flags := range scannerRolePermissions {
		_, err = db.Exec("INSERT INTO role_permission (id, `create`, `read`, `update`, `delete`, role_id, permission_code, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))",
			fmt.Sprintf("rlpm_e2escanner_%s_%d", suffix, i),
			strings.Contains(flags, "c"), strings.Contains(flags, "r"), strings.Contains(flags, "u"), strings.Contains(flags, "d"),
			roleID, code)
		require.NoError(t, err)
		i++
	}

	_, email := covAuthPasswordsRegisterUser(t, "e2e-scanner")
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{"email": email, "role_id": roleID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)
	accountUserID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, accountUserID)
	t.Cleanup(func() {
		removeAccountUser(accountUserID)
		_, _ = db.Exec("DELETE FROM role_permission WHERE role_id = ?", roleID)
		_, _ = db.Exec("DELETE FROM role WHERE id = ?", roleID)
	})
	return loginAsUser(t, email, covAuthUsersPassword, SeedAccountID)
}

func readAs(t *testing.T, client *Client, path string, params url.Values) map[string]any {
	t.Helper()
	status, raw, err := client.GetListRaw(path, params)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	return parseJSON(raw)
}

func postAs(t *testing.T, client *Client, path string, body map[string]any, wantStatus int) map[string]any {
	t.Helper()
	status, raw, err := client.Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, wantStatus, status, raw)
	return parseJSON(raw)
}

// requireSameForBoth asserts the operator reads what an admin reads at path.
func requireSameForBoth(t *testing.T, operator *Client, path string, params url.Values) {
	t.Helper()
	requireReadsAgree(t, operator, path, func(client *Client) map[string]any { return readAs(t, client, path, params) })
}

// requireSamePostForBoth asserts a read-only POST answers the operator as it answers an admin.
func requireSamePostForBoth(t *testing.T, operator *Client, path string, body map[string]any) {
	t.Helper()
	requireReadsAgree(t, operator, path, func(client *Client) map[string]any { return postAs(t, client, path, body, http.StatusOK) })
}

// requireReadsAgree reads as an admin and as the operator until both agree. Stock a scan moves settles
// a moment after the scan, so two reads in a row can straddle it; what a role may not see never agrees.
func requireReadsAgree(t *testing.T, operator *Client, what string, read func(client *Client) map[string]any) {
	t.Helper()
	eventually(t, 15*time.Second, 250*time.Millisecond, func() error {
		if admin, asOperator := read(apiClient), read(operator); !reflect.DeepEqual(admin, asOperator) {
			return fmt.Errorf("the operator reads %s differently from an admin:\nadmin:    %v\noperator: %v", what, admin, asOperator)
		}
		return nil
	})
}

func flowByBatch(t *testing.T, flow map[string]any) map[string]any {
	t.Helper()
	byID := map[string]any{}
	for _, node := range jsonArray(flow, "data") {
		byID[jsonField(jsonObject(node.(map[string]any), "batch"), "id")] = node
	}
	require.NotEmpty(t, byID)
	return byID
}

// scanAs makes a scan as client, the way the station sends it, and registers its cleanup.
func scanAs(t *testing.T, client *Client, action string, body map[string]any) map[string]any {
	t.Helper()
	markOutbox(t)
	got := postAs(t, client, batchesPath+"/actions/"+action+"?include="+stationBatchIncludes, body, http.StatusCreated)
	settleScanAtCleanup(t, action, jsonField(got, "id"))
	return got
}

// requireStationShowsScan asserts the batch a scan returned to the operator carries everything the
// station shows of it, as an admin reads the same batch in the station's list.
func requireStationShowsScan(t *testing.T, stationID string, scanned map[string]any) {
	t.Helper()
	rows := jsonArray(readAs(t, apiClient, scanningStationsPath+"/"+stationID+"/batches", url.Values{"limit": {"100"}, "include": {stationBatchIncludes}}), "data")
	var adminRow map[string]any
	for _, row := range rows {
		if jsonField(row.(map[string]any), "id") == jsonField(scanned, "id") {
			adminRow = row.(map[string]any)
		}
	}
	require.NotNil(t, adminRow, "the station lists the batch %s", jsonField(scanned, "id"))
	// The list carries no department; the station reads it off the scan.
	for _, field := range []string{"item", "production_step", "scanning_station", "quantity", "seconds", "waste", "scanned_at", "closed_at"} {
		assert.Equal(t, adminRow[field], scanned[field], "the operator's %s on the scanned batch", field)
	}
	require.NotNil(t, jsonObject(scanned, "item"), "the station prints the item")
	assert.NotEmpty(t, jsonField(jsonObject(scanned, "item"), "name"))
	require.NotNil(t, jsonObject(jsonObject(scanned, "quantity"), "unit"), "the station shows the quantity's unit")
	assert.NotEmpty(t, jsonField(jsonObject(jsonObject(scanned, "quantity"), "unit"), "abbreviation"))
	require.NotNil(t, jsonObject(scanned, "department"), "the station shows the scan's department")
	assert.NotEmpty(t, jsonField(jsonObject(scanned, "department"), "name"))
}

func TestBatchScanningScanner_RunsEveryStation(t *testing.T) {
	t.Parallel()
	operator := scannerOperator(t)
	c := newScanChain(t)
	mergeStation := newScanStation(t, "merge_batch")
	merged := newScanPart(t, scanShippingCategory)
	mergeStep := newScanStep(t, mergeStation, scanQty{merged.id, "1", unitEach},
		scanConsumption{itemID: c.c.id, value: "1", unitID: unitEach})
	first := planScanBatch(t, c.a.id, "10", unitEach)
	second := planScanBatch(t, c.a.id, "4", unitEach)

	// The station page: the station, what was scanned there, and what a scan will use.
	for _, station := range []string{c.initStation, c.moveStation, c.splitStation, mergeStation} {
		requireSameForBoth(t, operator, scanningStationsPath+"/"+station, url.Values{"include": {"department"}})
	}
	requireSamePostForBoth(t, operator, scanningStationsPath+"/"+c.initStation+"/consumptions", map[string]any{"batch_ids": []string{first}})

	// Initialize.
	initialized := scanAs(t, operator, "initialize", map[string]any{"batch_id": first, "scanning_station_id": c.initStation})
	requireStationShowsScan(t, c.initStation, initialized)
	scanAs(t, operator, "initialize", map[string]any{"batch_id": second, "scanning_station_id": c.initStation})
	requireSameForBoth(t, operator, scanningStationsPath+"/"+c.initStation+"/batches", url.Values{"limit": {"25"}, "include": {stationBatchIncludes}})
	// The preview reads stock, which the scans' inventory moves once their messages are handled.
	settleScans(t, first, second)
	waitForOnHand(t, c.a.id, "14")

	// Move, after asking where the batch goes next.
	requireSamePostForBoth(t, operator, batchesPath+"/"+first+"/next-steps", map[string]any{"scanning_station_id": c.moveStation})
	requireSamePostForBoth(t, operator, scanningStationsPath+"/"+c.moveStation+"/consumptions", map[string]any{"batch_ids": []string{first}, "production_step_id": c.moveStep})
	moved := scanAs(t, operator, "move", map[string]any{"batch_ids": []string{first}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation})
	requireStationShowsScan(t, c.moveStation, moved)
	scanAs(t, operator, "move", map[string]any{"batch_ids": []string{second}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation})

	// Split, after reading what is left.
	remainingBody := map[string]any{"batch_ids": []string{first}, "production_step_id": c.splitStep}
	adminLeft := postAs(t, apiClient, batchesPath+"/remaining-quantities?include=unit", remainingBody, http.StatusOK)
	operatorLeft := postAs(t, operator, batchesPath+"/remaining-quantities?include=unit", remainingBody, http.StatusOK)
	delete(adminLeft, "id") // a fresh id each read
	delete(operatorLeft, "id")
	assert.Equal(t, adminLeft, operatorLeft, "the operator reads what is left as an admin does")
	split := scanAs(t, operator, "split", splitBody(c.splitStation, c.splitStep, []string{first}, "6", "1", "1", false))
	requireStationShowsScan(t, c.splitStation, split)
	rest := scanAs(t, operator, "split", splitBody(c.splitStation, c.splitStep, []string{second}, "4", "", "", true))

	// Merge.
	mergeBody := map[string]any{"batch_ids": []string{jsonField(split, "id"), jsonField(rest, "id")}, "production_step_id": mergeStep, "scanning_station_id": mergeStation}
	requireSamePostForBoth(t, operator, scanningStationsPath+"/"+mergeStation+"/consumptions", map[string]any{"batch_ids": mergeBody["batch_ids"], "production_step_id": mergeStep})
	mergedBatch := scanAs(t, operator, "merge", mergeBody)
	requireStationShowsScan(t, mergeStation, mergedBatch)

	// The batch's history, as the flow page and a printed traveler read it. Its nodes come in no fixed
	// order; the page lays them out by their links.
	flowParams := url.Values{"include": {"batch.quantity.unit,batch.seconds.unit,batch.waste.unit"}}
	assert.Equal(t, flowByBatch(t, readAs(t, apiClient, batchesPath+"/"+first+"/flow", flowParams)),
		flowByBatch(t, readAs(t, operator, batchesPath+"/"+first+"/flow", flowParams)), "the operator reads the flow as an admin does")

	// Undo, the way the station does: the latest scan first.
	postAs(t, operator, batchesPath+"/actions/bulk-delete", map[string]any{"batch_ids": []string{jsonField(mergedBatch, "id")}}, http.StatusOK)
	waitForMessagesSettled(t, "core.cmd.undo_batch_scan", "core.undo_batch_scan", jsonField(mergedBatch, "id"))
	status, raw, err := operator.Delete(batchesPath + "/" + jsonField(rest, "id") + "?include=" + stationBatchIncludes)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
}

// Turning a scan's materials off, or scanning as another kind of station, is a supervisor's correction:
// the scanner role is refused both, as it was on the dashboard.
func TestBatchScanningScanner_CannotCorrectAScan(t *testing.T) {
	t.Parallel()
	operator := scannerOperator(t)
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "3", unitEach)

	for name, extra := range map[string]map[string]any{
		"without materials": {"consume_materials": false},
		"as another type":   {"type_override": "move_batch"},
	} {
		body := map[string]any{"batch_id": planned, "scanning_station_id": c.initStation}
		for k, v := range extra {
			body[k] = v
		}
		status, raw, err := operator.Post(batchesPath+"/actions/initialize", body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, http.StatusForbidden, status, raw)
		assert.Contains(t, string(raw), "permission", name)
	}

	initializeScan(t, planned, c.initStation, nil)
}
