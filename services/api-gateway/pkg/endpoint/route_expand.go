package apiendpoint

import (
	"net/url"
	"strings"
)

// ExpandRoute substitutes {param} placeholders in an API route template (same
// syntax as Materialize Route fields and OpenAPI path templates) with values from
// pathParams. Keys must match placeholder names (e.g. "property_id" for
// "{property_id}"). Values are escaped for use as a single path segment.
func ExpandRoute(routeTemplate string, pathParams map[string]string) string {
	if routeTemplate == "" || len(pathParams) == 0 {
		return routeTemplate
	}
	out := routeTemplate
	for k, v := range pathParams {
		out = strings.ReplaceAll(out, "{"+k+"}", url.PathEscape(v))
	}
	return out
}
