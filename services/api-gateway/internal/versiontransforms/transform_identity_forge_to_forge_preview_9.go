package versiontransforms

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped the identity resources:
//   - account user: department became work_center, is_commission_eligible became commission_eligibility
//     (eligible | ineligible), and portal_access was added. The list lists every status unless statuses[] narrows it
//     (removed_scope is gone) and filters by commission_eligibility. The status actions are POSTs that answer with
//     the account user (RoutePreview9Identity reroutes preview.9's PUTs and list filters).
//   - role: access, grants, revokes and template were added; permissions is now the effective list from the catalog.
//     A role created without permissions holds the baseline every member holds, where preview.9 created an empty one.
//   - API key: status and client were added. Create and rotate answer { secret, api_key } instead of
//     { api_key_secret, api_key_info }.
//
// preview.9 callers send and read the old shapes.
type identityForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(identityForgeToPreview9{})
}

const (
	accountUsersRoute = "/v1/identity/account-users"
	accountUserRoute  = "/v1/identity/account-users/{id}"
	rolesRoute        = "/v1/identity/roles"
)

func (identityForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		switch obj["object"] {
		case string(constants.ObjectTypeAccountUser):
			renameKey(obj, "work_center", "department")
			if v, ok := obj["commission_eligibility"]; ok {
				delete(obj, "commission_eligibility")
				switch v {
				case string(constants.CommissionEligibilityEligible):
					obj["is_commission_eligible"] = true
				case string(constants.CommissionEligibilityIneligible):
					obj["is_commission_eligible"] = false
				default:
					obj["is_commission_eligible"] = nil
				}
			}
			delete(obj, "portal_access")
		case string(constants.ObjectTypeRole):
			for _, k := range []string{"template", "access", "grants", "revokes"} {
				delete(obj, k)
			}
		case string(constants.ObjectTypeAPIKey):
			delete(obj, "status")
			delete(obj, "client")
		case string(constants.ObjectTypeCreatedAPIKey):
			renameKey(obj, "secret", "api_key_secret")
			renameKey(obj, "api_key", "api_key_info")
		}
	})
	return data
}

// TransformRequestBody upgrades preview.9's account-user writes and keeps its permission-less role create empty.
func (identityForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch {
	case (method == http.MethodPost && route == accountUsersRoute) || (method == http.MethodPatch && route == accountUserRoute):
		renameKey(data, "department_id", "work_center_id")
		if v, ok := data["is_commission_eligible"]; ok {
			delete(data, "is_commission_eligible")
			if b, isBool := v.(bool); isBool {
				data["commission_eligibility"] = string(constants.CommissionEligibilityOf(b))
			} else {
				data["commission_eligibility"] = v
			}
		}
	case method == http.MethodPost && route == rolesRoute:
		if _, ok := data["permissions"]; !ok {
			data["permissions"] = []any{}
		}
	}
	return data
}

func (identityForgeToPreview9) TransformQuery(_ object.Type, _ string, query url.Values) url.Values {
	return query
}

// upgradePreview9AccountUserList upgrades preview.9's account-user list filters: removed members were hidden unless
// removed_scope was `included`, and commission eligibility was a boolean.
func upgradePreview9AccountUserList(query url.Values) url.Values {
	scope := query.Get("removed_scope")
	query.Del("removed_scope")
	if scope != string(constants.RemovedResourceScopeIncluded) && !query.Has("statuses") && !query.Has("statuses[]") {
		query["statuses[]"] = []string{string(constants.AccountUserStatusActive), string(constants.AccountUserStatusDisabled)}
	}
	if query.Has("is_commission_eligible") {
		v := query.Get("is_commission_eligible")
		query.Del("is_commission_eligible")
		switch v {
		case "true":
			v = string(constants.CommissionEligibilityEligible)
		case "false":
			v = string(constants.CommissionEligibilityIneligible)
		}
		query.Set("commission_eligibility", v)
	}
	return query
}

// preview9AccountUserAction matches preview.9's PUT account-user status actions.
var preview9AccountUserAction = regexp.MustCompile(`^/v1/identity/account-users/[^/]+/actions/(activate|disable|remove)$`)

// RoutePreview9Identity serves preview.9's PUT account-user status actions from the forge POSTs, and upgrades its
// account-user list filters. It wraps the router: a query transformer can't tell the list from create, which share the
// route. Only a request naming preview.9 is changed.
func RoutePreview9Identity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() {
			switch {
			case r.Method == http.MethodPut && preview9AccountUserAction.MatchString(r.URL.Path):
				r = r.Clone(r.Context())
				r.Method = http.MethodPost
			case r.Method == http.MethodGet && r.URL.Path == accountUsersRoute:
				r = r.Clone(r.Context())
				r.URL.RawQuery = upgradePreview9AccountUserList(r.URL.Query()).Encode()
			}
		}
		next.ServeHTTP(w, r)
	})
}
