package versiontransforms

import (
	"net/http"
	"strings"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge renamed background work from job to async_job: the object and its rows and export, the statuses
// (created, started, cancelled became queued, running, canceled) and cancelled_at. preview.9 callers see the old
// names wherever a job appears, including the 202 of every long-running action.
type asyncJobForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(asyncJobForgeToPreview9{})
}

var preview9JobStatuses = map[string]string{"queued": "created", "running": "started", "canceled": "cancelled"}

var preview9JobObjects = map[string]string{"async_job": "job", "async_job_result": "job_result", "async_job_export": "job_export"}

func (asyncJobForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		name, _ := obj["object"].(string)
		old, ok := preview9JobObjects[name]
		if !ok {
			return
		}
		obj["object"] = old
		if name != "async_job" {
			return
		}
		if status, ok := obj["status"].(string); ok {
			if s, renamed := preview9JobStatuses[status]; renamed {
				obj["status"] = s
			}
		}
		if at, ok := obj["canceled_at"]; ok {
			obj["cancelled_at"] = at
			delete(obj, "canceled_at")
		}
	})
	return data
}

const (
	preview9JobsPrefix   = "/v1/core/jobs/"
	asyncJobsPrefix      = "/v1/core/async-jobs/"
	preview9CancelSuffix = "/cancel"
)

// RoutePreview9Jobs serves preview.9's job routes from their forge paths: GET /v1/core/jobs/{id} and
// POST /v1/core/jobs/{id}/cancel reach /v1/core/async-jobs/{id} and /v1/core/async-jobs/{id}/actions/cancel.
// Only a request naming preview.9 is rerouted, so forge callers never see the old paths. It wraps the router,
// since a route is matched before any endpoint's transformers run.
func RoutePreview9Jobs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() && strings.HasPrefix(r.URL.Path, preview9JobsPrefix) {
			rest := strings.TrimPrefix(r.URL.Path, preview9JobsPrefix)
			if id, ok := strings.CutSuffix(rest, preview9CancelSuffix); ok && !strings.Contains(id, "/") {
				rest = id + "/actions/cancel"
			}
			r = r.Clone(r.Context())
			r.URL.Path = asyncJobsPrefix + rest
			r.URL.RawPath = ""
		}
		next.ServeHTTP(w, r)
	})
}
