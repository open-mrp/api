package versiontransforms

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge renamed departments to work centers: the routes moved from /v1/operations/departments to
// /v1/operations/work-centers, the object became work_center, and machines and scanning stations name theirs in
// work_center (written as work_center_id, and as work_center in their bulk upsert rows) instead of department. Work
// centers, machines and scanning stations lost their notes, and machines gained a status.
//
// preview.9 callers see department objects with notes null, machines and stations with a department, and jobs and
// audit events about departments; their routes, bodies and include keys are upgraded, and the notes they send are
// dropped (notes are no longer stored on these records).
type workCenterForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(workCenterForgeToPreview9{})
}

const (
	workCentersRoute           = "/v1/operations/work-centers"
	workCenterRoute            = "/v1/operations/work-centers/{id}"
	workCentersBulkUpsertRoute = "/v1/operations/work-centers/actions/bulk-upsert"
	machinesRoute              = "/v1/operations/machines"
	machineRoute               = "/v1/operations/machines/{id}"
	machinesBulkUpsertRoute    = "/v1/operations/machines/actions/bulk-upsert"
	stationsRoute              = "/v1/operations/scanning-stations"
	stationRoute               = "/v1/operations/scanning-stations/{id}"
	stationsBulkUpsertRoute    = "/v1/operations/scanning-stations/actions/bulk-upsert"
	preview9DepartmentsPrefix  = "/v1/operations/departments"
)

func (workCenterForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		switch obj["object"] {
		case "work_center":
			obj["object"] = "department"
			setNullIfAbsent(obj, "notes")
		case "machine", "scanning_station":
			renameKey(obj, "work_center", "department")
			setNullIfAbsent(obj, "notes")
		case "async_job", "job", "audit_event":
			if obj["resource_type"] == "work_center" {
				obj["resource_type"] = "department"
			}
		}
	})
	return data
}

func setNullIfAbsent(obj map[string]any, key string) {
	if _, ok := obj[key]; !ok {
		obj[key] = nil
	}
}

func (workCenterForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch method + " " + route {
	case http.MethodPost + " " + workCentersRoute, http.MethodPatch + " " + workCenterRoute,
		http.MethodPatch + " " + machineRoute, http.MethodPatch + " " + stationRoute:
		delete(data, "notes")
	case http.MethodPost + " " + machinesRoute, http.MethodPost + " " + stationsRoute:
		delete(data, "notes")
		renameKey(data, "department_id", "work_center_id")
	case http.MethodPost + " " + workCentersBulkUpsertRoute:
		renameKey(data, "departments", "work_centers")
		forEachRow(data, "work_centers", func(row map[string]any) { delete(row, "notes") })
	case http.MethodPost + " " + machinesBulkUpsertRoute:
		forEachRow(data, "machines", upgradeMemberRow)
	case http.MethodPost + " " + stationsBulkUpsertRoute:
		forEachRow(data, "scanning_stations", upgradeMemberRow)
	}
	return data
}

// upgradeMemberRow renames a machine or station bulk row's department and drops its notes.
func upgradeMemberRow(row map[string]any) {
	delete(row, "notes")
	renameKey(row, "department", "work_center")
}

func forEachRow(data map[string]any, key string, fn func(map[string]any)) {
	rows, _ := data[key].([]any)
	for _, row := range rows {
		if m, ok := row.(map[string]any); ok {
			fn(m)
		}
	}
}

// TransformQuery renames a preview.9 include key's `department` segments to `work_center` when the route offers only
// the renamed key, as it does for a machine's or scanning station's work center.
func (workCenterForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	for _, param := range []string{"include", "include[]"} {
		values, ok := query[param]
		if !ok {
			continue
		}
		var keys []string
		for _, v := range values {
			for _, key := range strings.Split(v, ",") {
				if key = strings.TrimSpace(key); key != "" {
					keys = append(keys, preview9WorkCenterIncludeKey(route, key))
				}
			}
		}
		if param == "include" {
			query[param] = []string{strings.Join(keys, ",")}
		} else {
			query[param] = keys
		}
	}
	return query
}

func preview9WorkCenterIncludeKey(route, key string) string {
	if routeOffers(route, key) || !strings.Contains(key, "department") {
		return key
	}
	segs := strings.Split(key, ".")
	for i, seg := range segs {
		if seg == "department" {
			segs[i] = "work_center"
		}
	}
	if renamed := strings.Join(segs, "."); routeOffers(route, renamed) {
		return renamed
	}
	return key
}

// RoutePreview9WorkCenters serves preview.9's department routes from their forge paths: /v1/operations/departments[/…]
// reaches /v1/operations/work-centers[/…]. Only a request naming preview.9 is rerouted. It wraps the router, since a
// route is matched before any endpoint's transformers run.
func RoutePreview9WorkCenters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() {
			if rest, ok := strings.CutPrefix(r.URL.Path, preview9DepartmentsPrefix); ok && (rest == "" || strings.HasPrefix(rest, "/")) {
				r = r.Clone(r.Context())
				r.URL.Path = workCentersRoute + rest
				r.URL.RawPath = ""
			}
		}
		next.ServeHTTP(w, r)
	})
}
