package versiontransforms

import (
	"net/url"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&requestLogForgePreview5To4{})
}

// requestLogForgePreview5To4 bridges the request-log list from 1.0.forge-preview.5 to 1.0.forge-preview.4.
//
// preview.5 bounds a list that omits `starts_at` to the 24 hours before `ends_at` (or now): a filter matching nothing
// otherwise reads every log the account has ever produced. preview.4 searched the whole history, so a preview.4 list
// that omits `starts_at` asks for the beginning of time explicitly. The resource itself did not change.
type requestLogForgePreview5To4 struct{}

// requestLogListRoute is the only request-log endpoint that takes a window; the retrieve rejects unknown parameters.
const requestLogListRoute = "/v1/core/request-logs"

func (t *requestLogForgePreview5To4) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *requestLogForgePreview5To4) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview4
}

func (t *requestLogForgePreview5To4) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeRequestLog}
}

func (t *requestLogForgePreview5To4) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *requestLogForgePreview5To4) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *requestLogForgePreview5To4) TransformQuery(_ constants.ObjectType, route string, query url.Values) url.Values {
	return searchWholeHistoryWithoutStart(route, requestLogListRoute, query)
}
