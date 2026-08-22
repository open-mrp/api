package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&jobForgePreview3To2{})
}

// jobForgePreview3To2 downgrades job payloads from 1.0.forge-preview.3 to 1.0.forge-preview.2.
//
// preview.3 reshaped the job around one row-indexed list: `results` became a List whose
// entries carry a `status` (created/updated/failed), an Entity for the resource produced
// and its sub-resources, and — for a rejected row — the error that rejected it. The
// separate `errors` array and the `error_summary` string went away, a whole-job failure
// moved to `error`, the four hoisted `created_by_*` fields became an expandable `created_by`
// actor, and `resource_type` was added. The downgrade splits the merged list back into
// results and errors, rebuilds the summary from the whole-job error, and hoists the actor.
//
// `created_by_username` cannot be rebuilt: an actor carries a display name and a handle
// (the email for a user), not the username preview.2 read straight off the user row. It
// downgrades to null rather than to a guess.
type jobForgePreview3To2 struct{}

func (t *jobForgePreview3To2) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview3
}

func (t *jobForgePreview3To2) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview2
}

func (t *jobForgePreview3To2) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeJob}
}

func (t *jobForgePreview3To2) ForcedIncludes(objectType constants.ObjectType) []string {
	if objectType == constants.ObjectTypeJob {
		// preview.2 carried the creator's name and email unconditionally, so the actor has
		// to be resolved for the downgrade to hoist real values onto every job response.
		return []string{"created_by"}
	}
	return nil
}

func (t *jobForgePreview3To2) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	downgradeJobsIn(data)
	return data
}

func (t *jobForgePreview3To2) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	// A job is never submitted, only read, so no request shape changed.
	return data
}

// downgradeJobsIn walks the payload and rewrites every job object in place — a single job
// and jobs inside a list envelope.
func downgradeJobsIn(node any) {
	switch v := node.(type) {
	case map[string]any:
		if v["object"] == string(constants.ObjectTypeJob) {
			downgradeJob(v)
		}
		for _, child := range v {
			downgradeJobsIn(child)
		}
	case []any:
		for _, child := range v {
			downgradeJobsIn(child)
		}
	}
}

// preview.2 spelled the bulk job types without a separator.
var preview2JobTypes = map[string]string{
	string(constants.JobTypeBulkCreate): "bulkcreate",
	string(constants.JobTypeBulkUpsert): "bulkupsert",
}

func downgradeJob(job map[string]any) {
	if legacy, ok := preview2JobTypes[asString(job["type"])]; ok {
		job["type"] = legacy
	}
	delete(job, "resource_type")

	downgradeJobCreatedBy(job)
	downgradeJobOutcomes(job)
	downgradeJobExport(job)
}

// hoists the actor's fields back onto the job, as preview.2 carried them.
func downgradeJobCreatedBy(job map[string]any) {
	actor, _ := job["created_by"].(map[string]any)
	delete(job, "created_by")

	var id, name, email any
	if actor != nil {
		id = actor["id"]
		name = actor["name"]
		email = actor["handle"]
	}

	job["created_by_id"] = id
	job["created_by_name"] = name
	job["created_by_username"] = nil
	job["created_by_email"] = email
}

// splits the merged row list back into the two preview.2 arrays: written rows in results,
// rejected ones in errors, with a whole-job failure as the index-less error entry it used
// to be recorded as.
func downgradeJobOutcomes(job map[string]any) {
	rows, hadResults := preview2JobRows(job["results"])

	var results []any
	var errors []any
	for _, row := range rows {
		if row["status"] == string(constants.JobResultStatusFailed) {
			errors = append(errors, map[string]any{"index": row["index"], "error": row["error"]})
			continue
		}
		results = append(results, preview2JobResult(row))
	}

	if jobError, ok := job["error"].(map[string]any); ok {
		errors = append(errors, map[string]any{"error": jobError})
		job["error_summary"] = jobError["message"]
	} else {
		job["error_summary"] = nil
	}
	delete(job, "error")

	// A null results list meant "nothing recorded yet" in preview.2 as well, so an absent
	// list stays absent rather than becoming an empty one. Assigned as an untyped nil:
	// a typed nil slice in an interface is not nil to a reader inspecting the map.
	switch {
	case !hadResults:
		job["results"] = nil
	case results == nil:
		job["results"] = []any{}
	default:
		job["results"] = results
	}
	if errors == nil {
		job["errors"] = nil
	} else {
		job["errors"] = errors
	}
}

// unwraps the preview.3 List envelope into the bare rows preview.2 knew, reporting whether
// the job carried a list at all.
func preview2JobRows(results any) ([]map[string]any, bool) {
	list, ok := results.(map[string]any)
	if !ok {
		return nil, false
	}
	data, _ := list["data"].([]any)
	rows := make([]map[string]any, 0, len(data))
	for _, entry := range data {
		if row, ok := entry.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows, true
}

// flattens one written row back to preview.2's bare ids.
func preview2JobResult(row map[string]any) map[string]any {
	out := map[string]any{
		"index":  row["index"],
		"action": row["status"],
		"id":     nil,
	}
	if resource, ok := row["resource"].(map[string]any); ok {
		out["id"] = resource["id"]
	}
	if ids := preview2SubResourceIDs(row["sub_resources"]); len(ids) > 0 {
		// preview.2 omitted the key entirely for an operation that produces no sub-resources.
		out["sub_resource_ids"] = ids
	}
	return out
}

func preview2SubResourceIDs(subResources any) []any {
	list, ok := subResources.(map[string]any)
	if !ok {
		return nil
	}
	data, _ := list["data"].([]any)
	ids := make([]any, 0, len(data))
	for _, entry := range data {
		if entity, ok := entry.(map[string]any); ok {
			ids = append(ids, entity["id"])
		}
	}
	return ids
}

// preview.2's export carried only the url, with no object discriminator.
func downgradeJobExport(job map[string]any) {
	export, ok := job["export"].(map[string]any)
	if !ok {
		return
	}
	job["export"] = map[string]any{"url": export["url"]}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
