package versiontransforms

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped integrations and portal domains:
//   - integration: credentials became a typed object instead of a JSON string; status inactive became disabled; a
//     read-only connection_status was added; PUT became PATCH; creating a connected provider again is a 409 (preview.9
//     callers keep the upsert, see the endpoint service).
//   - portal domain: dns_records became a plain array of {type, name, value} instead of a list of dns_record objects
//     with a reason.
//
// preview.9 callers send and read the old shapes. A record's reason is rebuilt from its type: TXT records prove
// ownership, the rest route traffic.
type settingsForgeToPreview9 struct {
	forgeToPreview9
}

func init() {
	version.RegisterTransformer(settingsForgeToPreview9{})
}

func (settingsForgeToPreview9) ObjectTypes() []object.Type {
	return []object.Type{constants.ObjectTypeAccountIntegration, constants.ObjectTypePortalDomain}
}

func (settingsForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		switch obj["object"] {
		case string(constants.ObjectTypeAccountIntegration):
			if obj["status"] == string(constants.PauseStatusDisabled) {
				obj["status"] = preview9IntegrationInactive
			}
			delete(obj, "connection_status")
		case string(constants.ObjectTypePortalDomain):
			if records, ok := obj["dns_records"].([]any); ok {
				obj["dns_records"] = preview9DNSRecordList(records)
			}
		}
	})
	return data
}

const preview9IntegrationInactive = "inactive"

// preview9DNSRecordList wraps portal domain records in the single-page list preview.9 returned, each a dns_record with
// its reason.
func preview9DNSRecordList(records []any) map[string]any {
	data := make([]any, 0, len(records))
	for _, r := range records {
		rec, ok := r.(map[string]any)
		if !ok {
			continue
		}
		reason := "routing"
		if rec["type"] == string(constants.DNSRecordTypeTXT) {
			reason = "ownership"
		}
		data = append(data, map[string]any{
			"object": "dns_record", "type": rec["type"], "name": rec["name"], "value": rec["value"], "reason": reason,
		})
	}
	return map[string]any{
		"object":    "list",
		"data":      data,
		"page_info": map[string]any{"next_page_url": nil, "previous_page_url": nil, "has_next_page": false, "has_prev_page": false},
	}
}

func (settingsForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

const (
	integrationsRoute = "/v1/settings/integrations"
	integrationRoute  = "/v1/settings/integrations/{id}"
)

// TransformRequestBody upgrades preview.9's create (credentials as a JSON string) and its update (status inactive).
func (settingsForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch {
	case method == http.MethodPost && route == integrationsRoute:
		// A string that isn't a JSON object is left as it is, so it fails validation as a malformed credentials value.
		if raw, ok := data["credentials"].(string); ok {
			var creds map[string]any
			if err := json.Unmarshal([]byte(raw), &creds); err == nil && creds != nil {
				data["credentials"] = creds
			}
		}
	case method == http.MethodPatch && route == integrationRoute:
		if data["status"] == preview9IntegrationInactive {
			data["status"] = string(constants.PauseStatusDisabled)
		}
	}
	return data
}

func (settingsForgeToPreview9) TransformQuery(_ object.Type, _ string, query url.Values) url.Values {
	return query
}

// integrationPath is a single integration's route; the stripe and hubspot sub-routes have more segments or are GETs.
var integrationPath = regexp.MustCompile(`^/v1/settings/integrations/[^/]+$`)

// RoutePreview9Settings serves preview.9's PUT /v1/settings/integrations/{id} from the forge PATCH, which takes the
// same partial body. Only a request naming preview.9 is rerouted.
func RoutePreview9Settings(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && integrationPath.MatchString(r.URL.Path) && r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() {
			r = r.Clone(r.Context())
			r.Method = http.MethodPatch
		}
		next.ServeHTTP(w, r)
	})
}
