package versiontransforms

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/open-mrp/api/shared/apiversion"
)

const preview9PackListRoute = "/v1/core/records/actions/generate-pack-list"

// RoutePreview9PackList serves preview.9's POST /v1/core/records/actions/generate-pack-list { shipment_id } from
// POST /v1/operations/shipments/{id}/actions/generate-pack-list. A body without a shipment_id is passed on unchanged,
// so it fails as it did on preview.9. Only a request naming preview.9 is rerouted.
func RoutePreview9PackList(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != preview9PackListRoute ||
			r.Header.Get(apiversion.Header) != apiversion.V1_0_Forge_Preview9.String() || r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}
		raw, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			r.Body = io.NopCloser(bytes.NewReader(raw))
			next.ServeHTTP(w, r)
			return
		}
		var body struct {
			ShipmentID string `json:"shipment_id"`
		}
		if json.Unmarshal(raw, &body) != nil || body.ShipmentID == "" || strings.Contains(body.ShipmentID, "/") {
			r.Body = io.NopCloser(bytes.NewReader(raw))
			next.ServeHTTP(w, r)
			return
		}
		r = r.Clone(r.Context())
		r.URL.Path = "/v1/operations/shipments/" + body.ShipmentID + "/actions/generate-pack-list"
		r.URL.RawPath = "/v1/operations/shipments/" + url.PathEscape(body.ShipmentID) + "/actions/generate-pack-list"
		empty := []byte("{}")
		r.Body = io.NopCloser(bytes.NewReader(empty))
		r.ContentLength = int64(len(empty))
		r.Header.Set("Content-Length", "2")
		next.ServeHTTP(w, r)
	})
}
