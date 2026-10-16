package versiontransforms

import (
	"net/http"
	"strings"

	"github.com/open-mrp/api/shared/apiversion"
)

// forge replaced inventory change logs with inventory movements (/v1/operations/inventory-movements), item inventory
// PATCHes and bulk reconciles with movement creates, the internal inventories list with inventory levels, and the
// item lot default with the lot-size default. The retired routes stay for preview.9 callers only, unchanged, so
// preview.9 needs no reshaping except one: GET /v1/catalog/items/{id}/inventory now returns an inventory level, so a
// preview.9 request for it is served by the retired item-inventory endpoint on its own path.
const (
	itemsPrefix                 = "/v1/catalog/items/"
	itemInventorySuffix         = "/inventory"
	preview9ItemInventorySuffix = "/preview-9-inventory"
)

// RoutePreview9Inventory serves preview.9's GET /v1/catalog/items/{id}/inventory from the retired item-inventory
// endpoint. Only a request naming preview.9 is rerouted.
func RoutePreview9Inventory(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() {
			if id, ok := itemInventoryPathID(r.URL.Path); ok {
				r = r.Clone(r.Context())
				r.URL.Path = itemsPrefix + id + preview9ItemInventorySuffix
				r.URL.RawPath = ""
			}
		}
		next.ServeHTTP(w, r)
	})
}

// itemInventoryPathID is the item ID of a /v1/catalog/items/{id}/inventory path.
func itemInventoryPathID(path string) (string, bool) {
	rest, ok := strings.CutPrefix(path, itemsPrefix)
	if !ok {
		return "", false
	}
	id, ok := strings.CutSuffix(rest, itemInventorySuffix)
	if !ok || id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}
