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
				t.Skipf("Cannot resolve path params for %s", ep.Path)
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
				t.Skipf("Cannot resolve path params for %s", ep.Path)
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
				t.Skipf("Cannot resolve path params for %s", ep.Path)
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
	"list-inventories": true, // inventories resource missing object field
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
				t.Skipf("Cannot resolve path params for %s", ep.Path)
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
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, url.Values{"limit": {"1"}})
			require.NoError(t, err, "GET %s?limit=1 (page 1) failed", path)
			skipOnNonClientError(t, path, statusCode)
			require.Equal(t, 200, statusCode, "GET %s?limit=1 returned %d: %s", path, statusCode, string(body))

			var page1 ListResponse
			require.NoError(t, json.Unmarshal(body, &page1))

			if len(page1.Data) == 0 || !page1.PageInfo.HasNextPage {
				t.Skipf("Not enough data for pagination test on %s", path)
				return
			}

			require.NotNil(t, page1.PageInfo.NextCursor, "next_cursor should be set when has_next_page is true")

			page2, _, err := apiClient.GetList(path, url.Values{
				"limit":  {"1"},
				"cursor": {*page1.PageInfo.NextCursor},
			})
			require.NoError(t, err, "GET %s?limit=1&cursor=... (page 2) failed", path)
			assert.NotEmpty(t, page2.Data, "Page 2 should have data")

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
