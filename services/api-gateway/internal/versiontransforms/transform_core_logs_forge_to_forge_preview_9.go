package versiontransforms

import (
	"net/url"
	"strings"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped core's logs, jobs and search:
//   - audit_event.actor, request_log.actor, email_log.sent_by and async_job.created_by became ActorRefs {type, id,
//     name, client}, always present, no longer expandable; a system actor has no id. preview.9 had expandable actors
//     {id, object: actor, type, name, handle, avatar_url, role}.
//   - audit events and request logs filter on occurred_after / occurred_before (were starts_at / ends_at); audit
//     events filter on root_resource_types[] / root_resource_ids[] (were the singular root_resource_type / _id).
//   - request_log.query_params is a map of value lists, always present (a parameter sent once was a string).
//   - email_log gained related.
//   - search returns search_result {type, resource_id, resource} instead of entity references; messaging_contact
//     results became account_user, and the customer filter became customer_ids[].
//
// preview.9 callers get the old shapes back. Handles, avatars and roles of actors are gone: they read as null, and
// a system actor reads as null.
type coreLogsForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(coreLogsForgeToPreview9{})
}

const (
	auditEventsRoute = "/v1/core/audit-events"
	requestLogsRoute = "/v1/core/request-logs"
	searchRoute      = "/v1/core/search"
)

// preview9ActorFields name, per object, the field that held an expandable actor in preview.9. A job may already
// read as `job` when this runs after the async job rename.
var preview9ActorFields = map[string]string{
	string(constants.ObjectTypeAuditEvent): "actor",
	string(constants.ObjectTypeRequestLog): "actor",
	string(constants.ObjectTypeEmailLog):   "sent_by",
	"async_job":                            "created_by",
	"job":                                  "created_by",
}

// preview9DroppedIncludes are include keys preview.9 could send that now name always-present fields.
var preview9DroppedIncludes = map[string]bool{"actor": true, "sent_by": true, "created_by": true, "query_params": true}

// namesDroppedInclude reports whether any segment of an include key is one of preview9DroppedIncludes, such as
// request.actor or actor.role.
func namesDroppedInclude(key string) bool {
	for seg := range strings.SplitSeq(key, ".") {
		if preview9DroppedIncludes[seg] {
			return true
		}
	}
	return false
}

func (coreLogsForgeToPreview9) ForcedIncludes(objectType object.Type) []string {
	if objectType == constants.ObjectTypeSearchResult {
		return []string{"entity"}
	}
	return nil
}

func (coreLogsForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		name, _ := obj["object"].(string)
		if field, ok := preview9ActorFields[name]; ok {
			if v, present := obj[field]; present {
				obj[field] = preview9Actor(v)
			}
		}
		switch name {
		case string(constants.ObjectTypeRequestLog):
			if params, ok := obj["query_params"].(map[string]any); ok {
				obj["query_params"] = preview9QueryParams(params)
			}
		case string(constants.ObjectTypeEmailLog):
			delete(obj, "related")
		case string(constants.ObjectTypeSearchResult):
			downgradeSearchResult(obj)
		}
	})
	return data
}

// preview9Actor is an ActorRef in preview.9's actor shape, or nil for a system actor or anything else.
func preview9Actor(v any) any {
	ref, ok := v.(map[string]any)
	if !ok || ref["object"] != nil {
		return v
	}
	id, _ := ref["id"].(string)
	if id == "" || ref["type"] == string(constants.ActorTypeSystem) {
		return nil
	}
	return map[string]any{
		"id":         id,
		"object":     string(constants.ObjectTypeActor),
		"type":       ref["type"],
		"name":       ref["name"],
		"handle":     nil,
		"avatar_url": nil,
		"role":       nil,
	}
}

// preview9QueryParams turns a parameter sent once back into a string.
func preview9QueryParams(params map[string]any) map[string]any {
	out := make(map[string]any, len(params))
	for k, v := range params {
		if values, ok := v.([]any); ok && len(values) == 1 {
			out[k] = values[0]
			continue
		}
		out[k] = v
	}
	return out
}

// downgradeSearchResult rewrites a search result in place as the entity reference preview.9 returned.
func downgradeSearchResult(obj map[string]any) {
	entity, _ := obj["entity"].(map[string]any)
	if entity == nil {
		entity = map[string]any{"id": obj["resource_id"], "type": obj["type"], "name": nil, "handle": nil}
	}
	for k := range obj {
		delete(obj, k)
	}
	for k, v := range entity {
		obj[k] = v
	}
	obj["object"] = string(constants.ObjectTypeEntity)
	if obj["type"] == string(constants.ObjectTypeAccountUser) {
		obj["type"] = string(constants.ObjectTypeMessagingContact)
	}
}

// TransformQuery upgrades preview.9's date and root filters, the search filters, and drops include keys that name
// fields now always present.
func (coreLogsForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	switch route {
	case auditEventsRoute:
		renameQuery(query, "starts_at", "occurred_after")
		renameQuery(query, "ends_at", "occurred_before")
		renameQuery(query, "root_resource_type", "root_resource_types[]")
		renameQuery(query, "root_resource_id", "root_resource_ids[]")
	case requestLogsRoute:
		renameQuery(query, "starts_at", "occurred_after")
		renameQuery(query, "ends_at", "occurred_before")
	case searchRoute:
		renameQuery(query, "customer", "customer_ids[]")
		for _, key := range []string{"types", "types[]"} {
			renameQueryValues(query, key, map[string]string{string(constants.ObjectTypeMessagingContact): string(constants.ObjectTypeAccountUser)})
		}
	}
	for _, param := range []string{"include", "include[]"} {
		values, ok := query[param]
		if !ok {
			continue
		}
		var kept []string
		for _, v := range values {
			for _, key := range strings.Split(v, ",") {
				key = strings.TrimSpace(key)
				if key == "" || (namesDroppedInclude(key) && !routeOffers(route, key)) {
					continue
				}
				kept = append(kept, key)
			}
		}
		switch {
		case len(kept) == 0:
			query.Del(param)
		case param == "include":
			query[param] = []string{strings.Join(kept, ",")}
		default:
			query[param] = kept
		}
	}
	return query
}
