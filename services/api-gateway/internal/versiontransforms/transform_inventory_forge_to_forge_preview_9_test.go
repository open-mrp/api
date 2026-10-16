package versiontransforms

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
)

func TestRoutePreview9Inventory(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, method, version, path, want string
	}{
		{"preview.9 item inventory is rerouted", http.MethodGet, apiversion.V1_0_Forge_Preview9.String(), "/v1/catalog/items/it_1/inventory", "/v1/catalog/items/it_1/preview-9-inventory"},
		{"forge keeps the level route", http.MethodGet, "1.0.forge", "/v1/catalog/items/it_1/inventory", "/v1/catalog/items/it_1/inventory"},
		{"preview.9 PATCH is left to its own route", http.MethodPatch, apiversion.V1_0_Forge_Preview9.String(), "/v1/catalog/items/it_1/inventory", "/v1/catalog/items/it_1/inventory"},
		{"other item routes pass", http.MethodGet, apiversion.V1_0_Forge_Preview9.String(), "/v1/catalog/items/it_1", "/v1/catalog/items/it_1"},
		{"nested paths pass", http.MethodGet, apiversion.V1_0_Forge_Preview9.String(), "/v1/catalog/items/it_1/x/inventory", "/v1/catalog/items/it_1/x/inventory"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got string
			h := RoutePreview9Inventory(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.URL.Path }))
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set(apiversion.Header, tc.version)
			h.ServeHTTP(httptest.NewRecorder(), req)
			assert.Equal(t, tc.want, got)
		})
	}
}
