//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListEndpoints_ValidShape(t *testing.T) {
	t.Parallel()
	for _, ep := range listEndpoints {
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Fatalf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, nil)
			require.NoError(t, err, "GET %s failed", path)
			skipOnNonClientError(t, path, statusCode)
			require.Equal(t, 200, statusCode, "GET %s returned %d: %s", path, statusCode, string(body))

			var list ListResponse
			require.NoError(t, json.Unmarshal(body, &list), "GET %s: invalid JSON", path)
			assert.Equal(t, "list", list.Object, "GET %s: object field should be 'list'", path)
			assert.NotNil(t, list.Data, "GET %s: data should not be nil", path)
		})
	}
}

func TestListEndpoints_LimitParam(t *testing.T) {
	t.Parallel()
	for _, ep := range listEndpoints {
		if !ep.HasParam("limit") {
			continue
		}

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Fatalf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, url.Values{"limit": {"1"}})
			require.NoError(t, err, "GET %s?limit=1 failed", path)
			skipOnNonClientError(t, path, statusCode)
			require.Equal(t, 200, statusCode, "GET %s?limit=1 returned %d: %s", path, statusCode, string(body))

			var list ListResponse
			require.NoError(t, json.Unmarshal(body, &list))
			assert.LessOrEqual(t, len(list.Data), 1, "GET %s?limit=1 returned more than 1 item", path)
		})
	}
}

func TestListEndpoints_SearchNonsense(t *testing.T) {
	t.Parallel()
	for _, ep := range listEndpoints {
		if !ep.HasParam("q") {
			continue
		}

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Fatalf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, url.Values{"q": {"zzzznonexistent99999"}})
			require.NoError(t, err, "GET %s?q=nonsense failed", path)
			skipOnNonClientError(t, path, statusCode)
			require.Equal(t, 200, statusCode, "GET %s?q=nonsense returned %d: %s", path, statusCode, string(body))

			var list ListResponse
			require.NoError(t, json.Unmarshal(body, &list))
			assertEmptyListData(t, list.Data, fmt.Sprintf("GET %s?q=nonsense should return empty data", path))
		})
	}
}

// objectFieldExcludedPaths are endpoints whose items don't have an "object" field.
// These represent real production issues that should be tracked separately.
var objectFieldExcludedPaths = map[string]bool{
	"list-inventories":                true, // inventories resource missing object field
	"list-audit-event-resource-types": true, // returns plain enum string values, not resource objects
}

func TestListEndpoints_ItemObjectField(t *testing.T) {
	t.Parallel()
	for _, ep := range listEndpoints {
		if objectFieldExcludedPaths[ep.OperationID] {
			continue
		}

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Fatalf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, nil)
			require.NoError(t, err, "GET %s failed", path)
			skipOnNonClientError(t, path, statusCode)
			require.Equal(t, 200, statusCode, "GET %s returned %d: %s", path, statusCode, string(body))

			var list ListResponse
			require.NoError(t, json.Unmarshal(body, &list), "GET %s: invalid JSON", path)

			for i, item := range list.Data {
				objectVal := DataItemField(item, "object")
				assert.NotEmpty(t, objectVal,
					"GET %s: item[%d] should have a non-empty 'object' field", path, i)
			}
		})
	}
}

func TestListEndpoints_PaginationCursor(t *testing.T) {
	t.Parallel()
	for _, ep := range listEndpoints {
		if !ep.HasParam("cursor") || !ep.HasParam("limit") {
			continue
		}
		if isExcludedFromPagination(ep.Path, ep.OperationID) {
			continue
		}

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Fatalf("Cannot resolve path params for %s", ep.Path)
				return
			}

			// This sweep paginates shared global lists, so parallel tests can
			// delete the rows behind the cursor between the two fetches and
			// leave page 2 legitimately empty. Retry the whole sequence a few
			// times: transient interference passes on a later attempt, while a
			// real pagination bug fails every attempt.
			const pageFetchAttempts = 3
			var page1, page2 ListResponse
			for attempt := 1; ; attempt++ {
				statusCode, body, err := apiClient.GetListRaw(path, url.Values{"limit": {"1"}})
				require.NoError(t, err, "GET %s?limit=1 (page 1) failed", path)
				skipOnNonClientError(t, path, statusCode)
				require.Equal(t, 200, statusCode, "GET %s?limit=1 returned %d: %s", path, statusCode, string(body))

				page1 = ListResponse{}
				require.NoError(t, json.Unmarshal(body, &page1))

				if len(page1.Data) == 0 || !page1.PageInfo.HasNextPage {
					t.Fatalf("Not enough data for pagination test on %s", path)
					return
				}

				require.NotNil(t, page1.PageInfo.NextPageURL, "next_page_url should be set when has_next_page is true")

				p2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
				require.NoError(t, err, "GET page 2 via next_page_url failed for %s", path)
				page2 = *p2

				if len(page2.Data) > 0 || attempt >= pageFetchAttempts {
					break
				}
				t.Logf("page 2 of %s empty on attempt %d (likely parallel deletes); retrying", path, attempt)
			}
			assert.NotEmpty(t, page2.Data, "Page 2 should have data (after %d attempts)", pageFetchAttempts)

			if len(page1.Data) > 0 && len(page2.Data) > 0 {
				id1 := DataItemField(page1.Data[0], "id")
				id2 := DataItemField(page2.Data[0], "id")
				if id1 != "" && id2 != "" {
					assert.NotEqual(t, id1, id2, "Page 1 and page 2 should return different items")
				}
			}
		})
	}
}
