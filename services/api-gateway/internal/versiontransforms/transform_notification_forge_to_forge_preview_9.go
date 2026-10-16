package versiontransforms

import (
	"net/http"
	"net/url"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped notifications and preferences:
//   - notification category → type, change_count → occurrences (always a count), and the record it is about →
//     related.record. preview.9's resource link stays as an internal field, so it reads as before.
//   - the list filters category and status → types[] and statuses[]; omitting statuses[] lists every state, where
//     preview.9's omitted status meant the active feed.
//   - per-category preferences (a list, and PUT one category) → one preferences object read with GET and changed with
//     PATCH. preview.9's list and PUT are served by their own route (RoutePreview9NotificationPreferences).
type notificationForgeToPreview9 struct {
	forgeToPreview9
	noRequestChange
}

func init() {
	version.RegisterTransformer(notificationForgeToPreview9{})
}

func (notificationForgeToPreview9) ObjectTypes() []object.Type {
	return []object.Type{constants.ObjectTypeNotification}
}

func (notificationForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] != string(constants.ObjectTypeNotification) {
			return
		}
		if v, ok := obj["type"]; ok {
			obj["category"] = v
			delete(obj, "type")
		}
		if n, ok := obj["occurrences"].(float64); ok && n > 1 {
			obj["change_count"] = n
		} else {
			obj["change_count"] = nil
		}
		delete(obj, "occurrences")
		delete(obj, "related")
	})
	return data
}

const notificationsRoute = "/v1/messaging/notifications"

// TransformQuery upgrades preview.9's single category and status filters; with no status it asked for the active
// feed.
func (notificationForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route != notificationsRoute {
		return query
	}
	if category := query.Get("category"); category != "" {
		query.Set("types[]", category)
	}
	query.Del("category")
	if status := query.Get("status"); status != "" {
		query.Set("statuses[]", status)
	} else {
		query["statuses[]"] = []string{
			string(constants.NotificationStatusUnseen), string(constants.NotificationStatusSeen), string(constants.NotificationStatusRead),
		}
	}
	query.Del("status")
	return query
}

const (
	preferencesRoute         = "/v1/messaging/preferences"
	preview9PreferencesRoute = "/v1/messaging/preview-9-preferences"
)

// RoutePreview9NotificationPreferences sends preview.9's GET (a list) and PUT (one category) of
// /v1/messaging/preferences to the route that still serves them.
func RoutePreview9NotificationPreferences(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == http.MethodGet || r.Method == http.MethodPut) && r.URL.Path == preferencesRoute &&
			r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() {
			r = r.Clone(r.Context())
			r.URL.Path = preview9PreferencesRoute
			r.URL.RawPath = ""
		}
		next.ServeHTTP(w, r)
	})
}
