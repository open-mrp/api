//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// embeddedListOverflow is how many members these tests create: more than an embedded list holds.
const embeddedListOverflow = 12

// pageThroughEmbedded returns the rows of an embedded list followed by every page its next_page_url leads to, each
// fetched with client.
func pageThroughEmbedded(t *testing.T, client *Client, embedded map[string]any) []map[string]any {
	t.Helper()
	rows := listRows(t, embedded)
	require.Len(t, rows, 10, "an embedded list holds the first 10")
	next := jsonField(jsonObject(embedded, "page_info"), "next_page_url")
	require.NotEmpty(t, next, "a longer embedded list links to the rest: %v", embedded)
	for pages := 0; next != ""; pages++ {
		require.Less(t, pages, 10, "next_page_url keeps paging")
		status, raw, err := client.GetListRawFromPageURL(&next)
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		page := parseJSON(raw)
		rows = append(rows, listRows(t, page)...)
		next = jsonField(jsonObject(page, "page_info"), "next_page_url")
	}
	return rows
}

// listedIDs is one page of path's list holding everything params selects, in its newest-first order.
func listedIDs(t *testing.T, path string, params url.Values) []string {
	t.Helper()
	params.Set("limit", "100")
	list, _, err := apiClient.GetList(path, params)
	require.NoError(t, err)
	ids := make([]string, len(list.Data))
	for i, item := range list.Data {
		ids[i] = DataItemField(item, "id")
	}
	return ids
}

// workCenterWithManyMembers creates a work center holding embeddedListOverflow machines and scanning stations.
func workCenterWithManyMembers(t *testing.T) string {
	t.Helper()
	wcID := jsonField(createWorkCenter(t, map[string]any{}), "id")
	for range embeddedListOverflow {
		createMachineAt(t, wcID, nil)
		createStationAt(t, wcID)
	}
	return wcID
}

func TestForgeWorkCenters_EmbeddedMembersPageThroughNextPageURL(t *testing.T) {
	t.Parallel()
	wcID := workCenterWithManyMembers(t)
	got := getWith(t, departmentsPath+"/"+wcID, "machines", "scanning_stations")

	for _, tc := range []struct{ key, path, object string }{
		{"machines", machinesPath, "machine"},
		{"scanning_stations", scanningStationsPath, "scanning_station"},
	} {
		rows := pageThroughEmbedded(t, apiClient, jsonObject(got, tc.key))
		want := listedIDs(t, tc.path, url.Values{"work_center_ids[]": {wcID}})
		require.Len(t, want, embeddedListOverflow)
		assert.Equal(t, want, rowIDs(rows), "%s: the embedded list and its pages hold every member once, newest first", tc.key)
		for _, row := range rows {
			assert.Equal(t, tc.object, jsonField(row, "object"))
			assert.NotContains(t, row, "department")
		}
	}
}

func TestVersionCompatWorkCenters_Preview9DepartmentMembersPageThroughNextPageURL(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	wcID := workCenterWithManyMembers(t)

	status, raw, err := preview.GetListRaw(preview9DepartmentsPath+"/"+wcID, url.Values{"include": {"machines,scanning_stations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	got := parseJSON(raw)
	assert.Equal(t, "department", jsonField(got, "object"))

	for _, tc := range []struct{ key, path, object string }{
		{"machines", machinesPath, "machine"},
		{"scanning_stations", scanningStationsPath, "scanning_station"},
	} {
		rows := pageThroughEmbedded(t, preview, jsonObject(got, tc.key))
		want := listedIDs(t, tc.path, url.Values{"work_center_ids[]": {wcID}})
		require.Len(t, want, embeddedListOverflow)
		assert.Equal(t, want, rowIDs(rows), "%s: preview.9 pages through every member once, newest first", tc.key)
		for _, row := range rows {
			assert.Equal(t, tc.object, jsonField(row, "object"))
			assert.Contains(t, row, "department", "preview.9 pages are preview.9-shaped")
			assert.NotContains(t, row, "work_center")
			assert.Contains(t, row, "notes")
		}
	}
}

// locationWithManyChildren creates a location holding embeddedListOverflow child locations.
func locationWithManyChildren(t *testing.T) string {
	t.Helper()
	parentID := jsonField(createLocation(t, map[string]any{"name": uniqueName("e2e-loc-pages"), "type": "aisle"}), "id")
	for range embeddedListOverflow {
		createLocation(t, map[string]any{"name": uniqueName("e2e-loc-pages-c"), "type": "bin", "parent_id": parentID})
	}
	return parentID
}

func TestForgeLocations_EmbeddedChildrenPageThroughNextPageURL(t *testing.T) {
	t.Parallel()
	parentID := locationWithManyChildren(t)

	got := getWith(t, locationsPath+"/"+parentID, "children")
	rows := pageThroughEmbedded(t, apiClient, jsonObject(got, "children"))
	want := listedIDs(t, locationsPath, url.Values{"parent_ids[]": {parentID}})
	require.Len(t, want, embeddedListOverflow)
	assert.Equal(t, want, rowIDs(rows), "the embedded children and their pages hold every child once, newest first")
}

func TestVersionCompatLocations_Preview9ChildrenPageThroughNextPageURL(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	parentID := locationWithManyChildren(t)

	status, raw, err := preview.GetListRaw(locationsPath+"/"+parentID, url.Values{"include[]": {"children"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	rows := pageThroughEmbedded(t, preview, jsonObject(parseJSON(raw), "children"))
	want := listedIDs(t, locationsPath, url.Values{"parent_ids[]": {parentID}})
	require.Len(t, want, embeddedListOverflow)
	assert.Equal(t, want, rowIDs(rows), "preview.9 pages through every child once, newest first")
	for _, row := range rows {
		assert.Equal(t, "location", jsonField(row, "object"))
		assert.Equal(t, "bin", jsonField(row, "type"))
	}
}
