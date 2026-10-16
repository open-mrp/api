package versiontransforms

import (
	"net/http"

	"github.com/open-mrp/api/shared/apiversion"
)

// preview9AnalyticsRoutes are the analytics reports preview.9 queried with PUT; forge queries them with
// POST …/actions/query.
var preview9AnalyticsRoutes = map[string]struct{}{
	"/v1/core/analytics/customer-pricing":      {},
	"/v1/core/analytics/deliveries":            {},
	"/v1/core/analytics/delivery-performance":  {},
	"/v1/core/analytics/demand-forecast":       {},
	"/v1/core/analytics/inventory-receipts":    {},
	"/v1/core/analytics/manufacturing":         {},
	"/v1/core/analytics/manufacturing-batch":   {},
	"/v1/core/analytics/materials":             {},
	"/v1/core/analytics/new-customers":         {},
	"/v1/core/analytics/new-customers-table":   {},
	"/v1/core/analytics/oee":                   {},
	"/v1/core/analytics/oee-trend":             {},
	"/v1/core/analytics/open-batches":          {},
	"/v1/core/analytics/open-orders":           {},
	"/v1/core/analytics/open-orders/breakdown": {},
	"/v1/core/analytics/open-orders/summary":   {},
	"/v1/core/analytics/orders":                {},
	"/v1/core/analytics/production-costs":      {},
	"/v1/core/analytics/quarterly-orders":      {},
	"/v1/core/analytics/realized-margins":      {},
	"/v1/core/analytics/sales":                 {},
	"/v1/core/analytics/sales-breakdown":       {},
	"/v1/core/analytics/sales-invoices":        {},
	"/v1/core/analytics/sales-lines":           {},
	"/v1/core/analytics/sales-summary":         {},
	"/v1/core/analytics/schedule-attainment":   {},
}

// preview9AnalyticsLimits are preview.9's page-size defaults on the analytics tables whose forge default is 25.
var preview9AnalyticsLimits = map[string]string{
	"/v1/core/analytics/new-customers-table": "100",
	"/v1/core/analytics/sales-lines":         "50",
}

// RoutePreview9Analytics serves preview.9's PUT analytics reports from their forge POST …/actions/query routes,
// keeping preview.9's page-size defaults. Only a request naming preview.9 is rerouted. It wraps the router, since a
// route is matched before any endpoint's transformers run.
func RoutePreview9Analytics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, moved := preview9AnalyticsRoutes[r.URL.Path]; moved && r.Method == http.MethodPut &&
			r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() {
			path := r.URL.Path
			r = r.Clone(r.Context())
			r.Method = http.MethodPost
			r.URL.Path = path + "/actions/query"
			r.URL.RawPath = ""
			if limit, ok := preview9AnalyticsLimits[path]; ok {
				q := r.URL.Query()
				if q.Get("limit") == "" {
					q.Set("limit", limit)
					r.URL.RawQuery = q.Encode()
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
