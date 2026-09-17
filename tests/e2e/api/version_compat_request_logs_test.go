//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the client to 1.0.forge-preview.4, which searched a request log list's whole history when
// starts_at was omitted. preview.5 defaults that window to the last day; a pinned client keeps the old reach.

const preview4APIVersion = "1.0.forge-preview.4"

func TestVersionCompat_RequestLogs_OmittedStartSearchesEverything(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.WithAPIVersion(preview4APIVersion).GetList(requestLogsPath, url.Values{
		"normalized_routes": {SeedReqLogFilterDatesRoute},
		"limit":             {"50"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Len(t, list.Data, 3, "preview.4 still reaches logs years old without a start")
	assertRequestLogMembership(t, list.Data,
		[]string{SeedReqLogFilterDateOld, SeedReqLogFilterDateMid, SeedReqLogFilterDateNew}, nil)
}

// An end alone was never a window in preview.4: it meant everything up to that instant.
func TestVersionCompat_RequestLogs_EndDateAloneReachesBackToTheStart(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.WithAPIVersion(preview4APIVersion).GetList(requestLogsPath, url.Values{
		"normalized_routes": {SeedReqLogFilterDatesRoute},
		"ends_at":           {"2023-09-01T00:00:00Z"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertRequestLogMembership(t, list.Data,
		[]string{SeedReqLogFilterDateOld, SeedReqLogFilterDateMid},
		[]string{SeedReqLogFilterDateNew})
}

// Pages keep the old reach: following next_page_url on preview.4 must not narrow to the last day.
func TestVersionCompat_RequestLogs_PagesKeepTheWholeHistory(t *testing.T) {
	t.Parallel()

	client := apiClient.WithAPIVersion(preview4APIVersion)
	list, status, err := client.GetList(requestLogsPath, url.Values{
		"normalized_routes": {SeedReqLogFilterDatesRoute},
		"limit":             {"1"},
	})
	seen := 0
	for page := 0; ; page++ {
		require.NoError(t, err)
		require.Equal(t, 200, status)
		seen += len(list.Data)
		if !list.PageInfo.HasNextPage {
			break
		}
		require.Less(t, page, 10)
		list, status, err = client.GetListFromPageURL(list.PageInfo.NextPageURL)
	}
	assert.Equal(t, 3, seen, "every page of the pinned walk stays unbounded")
}

// The retrieve takes no window; the preview.4 upgrade must not hand it one.
func TestVersionCompat_RequestLogs_RetrieveIsUnaffected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.WithAPIVersion(preview4APIVersion).GetListRaw(requestLogsPath+"/"+SeedReqLogFilterDateOld, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, SeedReqLogFilterDateOld, jsonField(parseJSON(body), "id"))
}
