//go:build e2e

package api_test

import (
	"bytes"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// Covers POST /v1/operations/inventory-change-logs/actions/export, the export that runs as a job: the file it produces and the window it is named for, the filters reaching the worker, and who may start one and read its job.
//
// Other tests write to the log concurrently, so row-for-row checks are made on an item the test creates and nobody else touches, and the account-wide export is checked by its head and containment.

const inventoryChangeLogExportSheet = "Inventory Change Logs"

var inventoryChangeLogExportHeader = []string{
	"Item", "Quantity Change", "Unit", "Action Type", "Responsible User", "Responsible Scanning Station", "Created At",
}

// changeLogExportItem creates a material and corrects its level twice, so the log holds its opening entry and two corrections for an item no other test touches.
func changeLogExportItem(t *testing.T) (sku, itemID string) {
	t.Helper()
	sku = uniqueName("e2e-icl-export")
	itemID = dashItemsMaterial(t, sku)
	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("1.5")})
	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("4")})
	return sku, itemID
}

// listedChangeLogCells reads the change-log list under params as the export writes it, newest first.
func listedChangeLogCells(t *testing.T, params url.Values) [][]string {
	t.Helper()
	params.Set("include", "item,responsible_user,responsible_scanning_station")
	whole := params.Get("limit") == ""
	if whole {
		params.Set("limit", "100")
	}
	list, status, err := apiClient.GetList(inventoryChangeLogsPath, params)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, nil)
	if whole {
		require.False(t, list.PageInfo.HasNextPage, "the comparison needs the whole list on one page")
	}

	cells := make([][]string, 0, len(list.Data))
	for _, raw := range list.Data {
		row := parseJSON(raw)
		quantity := jsonObject(row, "quantity")
		createdAt, err := time.Parse(time.RFC3339Nano, jsonField(row, "created_at"))
		require.NoError(t, err)
		cells = append(cells, []string{
			jsonField(jsonObject(row, "item"), "sku"),
			jsonField(quantity, "value"),
			jsonField(jsonObject(quantity, "unit"), "abbreviation"),
			jsonField(row, "action_type"),
			jsonField(jsonObject(row, "responsible_user"), "name"),
			jsonField(jsonObject(row, "responsible_scanning_station"), "name"),
			createdAt.UTC().Format(time.RFC3339),
		})
	}
	return cells
}

// exportedChangeLogRows downloads a completed export and returns the name it saves under and its data rows, having checked it is a workbook with the export's one sheet and header.
func exportedChangeLogRows(t *testing.T, job map[string]any) (filename string, rows [][]string) {
	t.Helper()
	filename, body := downloadExportFile(t, job)
	book, err := excelize.OpenReader(bytes.NewReader(body))
	require.NoError(t, err, "the export is an xlsx workbook")
	t.Cleanup(func() { _ = book.Close() })

	require.Equal(t, []string{inventoryChangeLogExportSheet}, book.GetSheetList())
	sheet, err := book.GetRows(inventoryChangeLogExportSheet)
	require.NoError(t, err)
	require.NotEmpty(t, sheet, "the file has at least its header")
	assert.Equal(t, inventoryChangeLogExportHeader, sheet[0])
	return filename, sheet[1:]
}

// assertChangeLogRows compares file rows with listed ones, reading the amounts as numbers: the list carries the stored scale and the file does not.
func assertChangeLogRows(t *testing.T, want, got [][]string) {
	t.Helper()
	require.Len(t, got, len(want), "one row per change log: %v", got)
	for i := range want {
		require.Len(t, got[i], len(want[i]), "row %d: %v", i, got[i])
		assertDecimalEqual(t, want[i][1], got[i][1], "row %d's amount", i)
		w, g := slices.Clone(want[i]), slices.Clone(got[i])
		w[1], g[1] = "", ""
		assert.Equal(t, w, g, "row %d", i)
	}
}

// rowsForSKU keeps the file rows recording changes to one item, in file order.
func rowsForSKU(rows [][]string, sku string) [][]string {
	var kept [][]string
	for _, row := range rows {
		if len(row) > 0 && row[0] == sku {
			kept = append(kept, row)
		}
	}
	return kept
}

// --- Render ---

func TestInventoryChangeLogExport_WithoutAWindowExportsAnItemsWholeHistory(t *testing.T) {
	t.Parallel()
	_, itemID := changeLogExportItem(t)

	resp, err := apiClient.DoFull(http.MethodPost, inventoryChangeLogsExportPath, map[string]any{"item_ids": []string{itemID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, resp.StatusCode, resp.Body)
	accepted := parseJSON(resp.Body)
	jobID := jsonField(accepted, "id")
	assert.Equal(t, "job", jsonField(accepted, "object"))
	assert.Equal(t, "export", jsonField(accepted, "type"))
	assert.Equal(t, "inventory_change_log", jsonField(accepted, "resource_type"), "the job says what the export lists")
	assert.Equal(t, jobsPath+"/"+jobID, resp.Header.Get("Location"), "202 points at the job to poll")

	job := awaitExportJob(t, apiClient, jobID)
	assert.Equal(t, "job_export", jsonField(jsonObject(job, "export"), "object"))
	filename, rows := exportedChangeLogRows(t, job)
	assert.Equal(t, "inventory-change-logs-all-all.xlsx", filename, "an open window reads as `all` at both ends")

	want := listedChangeLogCells(t, url.Values{"item_ids": {itemID}})
	require.Len(t, want, 3, "the opening entry and two corrections")
	assertChangeLogRows(t, want, rows)
}

// The file is named for the window it was asked for, so two ranges downloaded side by side do not overwrite each other, and holds only the changes inside it.
func TestInventoryChangeLogExport_NamesTheFileForItsWindow(t *testing.T) {
	t.Parallel()
	_, itemID := changeLogExportItem(t)
	startsAt := time.Now().UTC().Add(-time.Hour)
	endsAt := time.Now().UTC().Add(time.Hour)
	day := func(at time.Time) string { return at.Format(time.DateOnly) }

	for name, tc := range map[string]struct {
		filters  map[string]any
		listed   url.Values
		filename string
		rows     int
	}{
		"both bounds": {
			map[string]any{"item_ids": []string{itemID}, "starts_at": rfc3339(startsAt), "ends_at": rfc3339(endsAt)},
			url.Values{"item_ids": {itemID}, "starts_at": {rfc3339(startsAt)}, "ends_at": {rfc3339(endsAt)}},
			"inventory-change-logs-" + day(startsAt) + "-" + day(endsAt) + ".xlsx", 3,
		},
		"open end": {
			map[string]any{"item_ids": []string{itemID}, "starts_at": rfc3339(startsAt)},
			url.Values{"item_ids": {itemID}, "starts_at": {rfc3339(startsAt)}},
			"inventory-change-logs-" + day(startsAt) + "-all.xlsx", 3,
		},
		"a window that closed before the item existed": {
			map[string]any{"item_ids": []string{itemID}, "starts_at": "2000-01-01T00:00:00Z", "ends_at": "2000-01-02T00:00:00Z"},
			url.Values{"item_ids": {itemID}, "starts_at": {"2000-01-01T00:00:00Z"}, "ends_at": {"2000-01-02T00:00:00Z"}},
			"inventory-change-logs-2000-01-01-2000-01-02.xlsx", 0,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			filename, rows := exportedChangeLogRows(t, completedExportJob(t, inventoryChangeLogsExportPath, tc.filters))
			assert.Equal(t, tc.filename, filename)
			want := listedChangeLogCells(t, tc.listed)
			require.Len(t, want, tc.rows)
			assertChangeLogRows(t, want, rows)
		})
	}
}

// With neither a window nor a filter the export walks the account's whole log, newest first: the 2099-dated fixtures head it, and an item created moments ago is in it.
func TestInventoryChangeLogExport_WithNoFiltersCoversTheWholeAccount(t *testing.T) {
	t.Parallel()
	sku, itemID := changeLogExportItem(t)

	filename, rows := exportedChangeLogRows(t, completedExportJob(t, inventoryChangeLogsExportPath, nil))
	assert.Equal(t, "inventory-change-logs-all-all.xlsx", filename)

	head := listedChangeLogCells(t, url.Values{"limit": {"2"}})
	require.Len(t, head, 2)
	require.GreaterOrEqual(t, len(rows), 2)
	assertChangeLogRows(t, head, rows[:2])
	assertChangeLogRows(t, listedChangeLogCells(t, url.Values{"item_ids": {itemID}}), rowsForSKU(rows, sku))
}

// --- Filters ---

// Each of the list's filters reaches the worker; one it dropped would hand back rows the caller excluded.
func TestInventoryChangeLogExport_AppliesEachFilter(t *testing.T) {
	t.Parallel()
	_, itemID := changeLogExportItem(t)

	for name, tc := range map[string]struct {
		filters map[string]any
		listed  url.Values
	}{
		"action type": {
			map[string]any{"item_ids": []string{itemID}, "action_types": []string{"user_correction"}},
			url.Values{"item_ids": {itemID}, "action_types": {"user_correction"}},
		},
		"every filter on the seeded entries": {
			map[string]any{
				"item_ids":            []string{SeedInventoryChangeLogItemID, SeedInventoryChangeLog2ItemID},
				"action_types":        []string{"user_action", "system_action"},
				"changed_by_user_ids": []string{SeedUserID},
				"starts_at":           "2099-01-01T00:00:00Z",
			},
			url.Values{
				"item_ids":            {SeedInventoryChangeLogItemID, SeedInventoryChangeLog2ItemID},
				"action_types":        {"user_action", "system_action"},
				"changed_by_user_ids": {SeedUserID},
				"starts_at":           {"2099-01-01T00:00:00Z"},
			},
		},
		"an unknown user narrows to nothing": {
			map[string]any{"item_ids": []string{itemID}, "changed_by_user_ids": []string{"us_nosuchuser00"}},
			url.Values{"item_ids": {itemID}, "changed_by_user_ids": {"us_nosuchuser00"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, rows := exportedChangeLogRows(t, completedExportJob(t, inventoryChangeLogsExportPath, tc.filters))
			assertChangeLogRows(t, listedChangeLogCells(t, tc.listed), rows)
		})
	}
}

func TestInventoryChangeLogExport_RejectsInvalidFilters(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		body  map[string]any
		param string
	}{
		"unknown action type": {map[string]any{"action_types": []string{"bogus_e2e_action"}}, "action_types"},
		"start is not a date": {map[string]any{"starts_at": "yesterday"}, "starts_at"},
		"null start":          {map[string]any{"starts_at": nil}, "starts_at"},
		"null end":            {map[string]any{"ends_at": nil}, "ends_at"},
		"ids are not a list":  {map[string]any{"item_ids": SeedInventoryChangeLogItemID}, "item_ids"},
		"unknown field":       {map[string]any{bogusE2EJSONField: "x"}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(inventoryChangeLogsExportPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, http.StatusBadRequest, status, body)
			errObj := requireErrorResponse(t, body, "", "invalid_request_error")
			if tc.param != "" {
				assert.Contains(t, errObj["param"], tc.param, "the error names the offending filter: %s", string(body))
			}
		})
	}
}

// --- Idempotency ---

func TestInventoryChangeLogExport_SameKeyReplaysTheSameJob(t *testing.T) {
	t.Parallel()
	key := newIdempotencyKey()
	body := map[string]any{"item_ids": []string{SeedInventoryChangeLogItemID}}

	status, first, err := apiClient.Post(inventoryChangeLogsExportPath, body, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, first)
	status, second, err := apiClient.Post(inventoryChangeLogsExportPath, body, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, second)

	assert.Equal(t, jsonField(parseJSON(first), "id"), jsonField(parseJSON(second), "id"), "a retried accept must not start a second export")
}

// --- Auth ---

func TestInventoryChangeLogExport_RequiresAuthentication(t *testing.T) {
	t.Parallel()
	anon := apiClient.WithBearerToken("", SeedAccountID)
	status, body, err := anon.Post(inventoryChangeLogsExportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, body)
}

// The export asks for what the synchronous one does, inventory_logs:read; reading the job it returns takes jobs:read.
func TestInventoryChangeLogExport_RefusesARoleWithoutInventoryLogsRead(t *testing.T) {
	t.Parallel()

	refused := customRoleClient(t, "items:read", "jobs:read")
	status, body, err := refused.Post(inventoryChangeLogsExportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, body)
	errObj := requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	assert.Contains(t, errObj["message"], "inventory_logs:read")

	_, itemID := changeLogExportItem(t)
	holder := customRoleClient(t, "inventory_logs:read", "jobs:read")
	job := completedExportJobAs(t, holder, inventoryChangeLogsExportPath, map[string]any{"item_ids": []string{itemID}})
	_, rows := exportedChangeLogRows(t, job)
	assertChangeLogRows(t, listedChangeLogCells(t, url.Values{"item_ids": {itemID}}), rows)
}

// --- Tenancy ---

// Each tenant's export reads its own account's log whatever IDs it filters by, and neither can read the other's job.
func TestInventoryChangeLogExport_TenantsExportAndReadOnlyTheirOwn(t *testing.T) {
	t.Parallel()
	_, itemID := changeLogExportItem(t)
	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	jobB := completedExportJobAs(t, tenantB, inventoryChangeLogsExportPath, map[string]any{"item_ids": []string{itemID}})
	var accountID string
	require.NoError(t, authDB(t).QueryRow("SELECT account_id FROM job WHERE job_id = ?", jsonField(jobB, "id")).Scan(&accountID))
	assert.Equal(t, SeedTenantBAccountID, accountID, "tenant B's export is recorded against tenant B")
	_, rows := exportedChangeLogRows(t, jobB)
	assert.Empty(t, rows, "tenant A's item has no history in tenant B's log")

	jobA := completedExportJob(t, inventoryChangeLogsExportPath, map[string]any{"item_ids": []string{itemID}})
	_, rows = exportedChangeLogRows(t, jobA)
	assert.Len(t, rows, 3, "tenant A's own export holds the item's history")

	resp, err := tenantB.GetFull(jobsPath+"/"+jsonField(jobA, "id"), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "another tenant's export job must not resolve: %s", string(resp.Body))

	resp, err = apiClient.GetFull(jobsPath+"/"+jsonField(jobB, "id"), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "another tenant's export job must not resolve: %s", string(resp.Body))
}
