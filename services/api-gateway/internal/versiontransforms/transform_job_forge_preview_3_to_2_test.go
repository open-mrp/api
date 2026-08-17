package versiontransforms

import (
	"encoding/json"
	"slices"
	"testing"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/version"
)

// a completed bulk create with one written row and one rejected one — the case the
// preview.2 split has to reconstruct
func jobPreview3Payload() map[string]any {
	return map[string]any{
		"id":            "jb_123",
		"object":        "job",
		"type":          "bulk_create",
		"resource_type": "production_run",
		"status":        "completed",
		"created_by": map[string]any{
			"id":     "acus_123",
			"object": "actor",
			"type":   "user",
			"name":   "Jane Doe",
			"handle": "jane@example.com",
		},
		"results": map[string]any{
			"object":    "list",
			"page_info": map[string]any{"has_next_page": false},
			"data": []any{
				map[string]any{
					"object": "job_result",
					"index":  float64(0),
					"status": "created",
					"resource": map[string]any{
						"id": "prun_1", "object": "entity", "type": "production_run",
					},
					"sub_resources": map[string]any{
						"object": "list",
						"data": []any{
							map[string]any{"id": "bch_1", "object": "entity", "type": "batch"},
						},
					},
					"error": nil,
				},
				map[string]any{
					"object":        "job_result",
					"index":         float64(1),
					"status":        "failed",
					"resource":      nil,
					"sub_resources": nil,
					"error": map[string]any{
						"code": "validation_failed", "type": "invalid_request_error", "message": "bad row",
					},
				},
			},
		},
		"error":        nil,
		"export":       nil,
		"created_at":   "2026-01-01T00:00:00Z",
		"updated_at":   "2026-01-02T00:00:00Z",
		"started_at":   "2026-01-01T00:00:00Z",
		"completed_at": "2026-01-02T00:00:00Z",
		"failed_at":    nil,
		"cancelled_at": nil,
	}
}

func TestTransformJob_SplitsResultsBackIntoResultsAndErrors(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	result := tr.Transform(constants.ObjectTypeJob, jobPreview3Payload())

	if result["type"] != "bulkcreate" {
		t.Errorf("preview.2 spelled the type without a separator, got %v", result["type"])
	}
	if _, ok := result["resource_type"]; ok {
		t.Error("preview.2 had no resource_type")
	}

	results, ok := result["results"].([]any)
	if !ok {
		t.Fatalf("preview.2 carried results as a bare array, got %T", result["results"])
	}
	if len(results) != 1 {
		t.Fatalf("only the written row belongs in results, got %v", results)
	}
	written := results[0].(map[string]any)
	if written["id"] != "prun_1" || written["action"] != "created" || written["index"] != float64(0) {
		t.Errorf("the written row did not flatten: %v", written)
	}
	subIDs, ok := written["sub_resource_ids"].([]any)
	if !ok || len(subIDs) != 1 || subIDs[0] != "bch_1" {
		t.Errorf("sub-resources must flatten back to bare ids, got %v", written["sub_resource_ids"])
	}

	errors, ok := result["errors"].([]any)
	if !ok || len(errors) != 1 {
		t.Fatalf("the rejected row belongs in errors, got %v", result["errors"])
	}
	rowErr := errors[0].(map[string]any)
	if rowErr["index"] != float64(1) {
		t.Errorf("the row error must keep its index, got %v", rowErr["index"])
	}
	if rowErr["error"].(map[string]any)["message"] != "bad row" {
		t.Errorf("the row's error object must carry through, got %v", rowErr["error"])
	}

	// A row failure is not a job failure, so preview.2's summary stays empty.
	if result["error_summary"] != nil {
		t.Errorf("no whole-job failure, so no summary: %v", result["error_summary"])
	}
	if _, ok := result["error"]; ok {
		t.Error("preview.2 had no error field")
	}
}

func TestTransformJob_HoistsTheCreatorOntoTheJob(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	result := tr.Transform(constants.ObjectTypeJob, jobPreview3Payload())

	if result["created_by_id"] != "acus_123" {
		t.Errorf("expected the actor's id hoisted, got %v", result["created_by_id"])
	}
	if result["created_by_name"] != "Jane Doe" {
		t.Errorf("expected the actor's name hoisted, got %v", result["created_by_name"])
	}
	if result["created_by_email"] != "jane@example.com" {
		t.Errorf("expected the actor's handle as the email, got %v", result["created_by_email"])
	}
	// An actor carries no username, and a downgrade may not invent one.
	if result["created_by_username"] != nil {
		t.Errorf("expected a null username rather than a guess, got %v", result["created_by_username"])
	}
	if _, ok := result["created_by"]; ok {
		t.Error("preview.2 had no created_by object")
	}
}

func TestTransformJob_WholeJobFailureBecomesAnIndexlessErrorAndSummary(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	payload := jobPreview3Payload()
	payload["status"] = "failed"
	payload["results"] = nil
	payload["error"] = map[string]any{
		"code": "internal_error", "type": "api_error", "message": "the whole batch fell over",
	}

	result := tr.Transform(constants.ObjectTypeJob, payload)

	if result["error_summary"] != "the whole batch fell over" {
		t.Errorf("the summary preview.2 read is the failure's message, got %v", result["error_summary"])
	}
	errors, ok := result["errors"].([]any)
	if !ok || len(errors) != 1 {
		t.Fatalf("the whole-job failure belongs in errors, got %v", result["errors"])
	}
	entry := errors[0].(map[string]any)
	if _, ok := entry["index"]; ok {
		t.Error("preview.2 marked a whole-job failure by the absence of an index")
	}
	if entry["error"].(map[string]any)["message"] != "the whole batch fell over" {
		t.Errorf("the failure object must carry through, got %v", entry["error"])
	}
	// A job with nothing recorded kept a null results list in preview.2 too.
	if result["results"] != nil {
		t.Errorf("expected results to stay null, got %v", result["results"])
	}
}

// A job that ran and wrote nothing is not a job that has recorded nothing, and preview.2
// drew that line the same way.
func TestTransformJob_KeepsAnEmptyResultsListDistinctFromNone(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	payload := jobPreview3Payload()
	payload["results"] = map[string]any{
		"object": "list", "page_info": map[string]any{}, "data": []any{},
	}

	result := tr.Transform(constants.ObjectTypeJob, payload)

	results, ok := result["results"].([]any)
	if !ok || len(results) != 0 {
		t.Errorf("expected an empty array, got %v", result["results"])
	}
}

func TestTransformJob_StripsTheExportDiscriminator(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	payload := jobPreview3Payload()
	payload["export"] = map[string]any{"object": "job_export", "url": "https://files.example.com/x.xlsx"}

	result := tr.Transform(constants.ObjectTypeJob, payload)

	export := result["export"].(map[string]any)
	if _, ok := export["object"]; ok {
		t.Error("preview.2's export carried only a url")
	}
	if export["url"] != "https://files.example.com/x.xlsx" {
		t.Errorf("the download link must survive, got %v", export["url"])
	}
}

func TestTransformJob_ListEnvelope(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	payload := map[string]any{
		"object": "list",
		"data":   []any{jobPreview3Payload()},
	}

	result := tr.Transform(constants.ObjectTypeJob, payload)

	job := result["data"].([]any)[0].(map[string]any)
	if job["created_by_id"] != "acus_123" {
		t.Errorf("a job inside a list envelope must be downgraded too, got %v", job)
	}
}

func TestTransformJob_RequestIsIdentity(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	in := map[string]any{"anything": "at all"}
	if out := tr.TransformRequest(constants.ObjectTypeJob, in); out["anything"] != "at all" {
		t.Errorf("a job is never submitted, so the request shape is untouched: %v", out)
	}
}

func TestTransformJob_DefaultRegistryEndToEnd(t *testing.T) {
	t.Parallel()

	result := version.Transform(
		version.V1_0_Forge_Preview3,
		version.V1_0_Forge_Preview2,
		constants.ObjectTypeJob,
		jobPreview3Payload(),
	)

	if result["created_by_email"] != "jane@example.com" {
		t.Errorf("expected the default registry to apply the downgrade, got %v", result["created_by_email"])
	}

	forced := version.ForcedIncludes(version.V1_0_Forge_Preview3, version.V1_0_Forge_Preview2, constants.ObjectTypeJob)
	if len(forced) != 1 || forced[0] != "created_by" {
		t.Errorf("expected forced includes [created_by] from the default registry, got %v", forced)
	}
}

// A preview.1 caller reaches the job through both transformers chained, so the job
// downgrade must not be skipped just because the older transformer names other types.
func TestTransformJob_ChainsDownToPreview1(t *testing.T) {
	t.Parallel()

	result := version.Transform(
		version.V1_0_Forge_Preview3,
		version.V1_0_Forge_Preview1,
		constants.ObjectTypeJob,
		jobPreview3Payload(),
	)

	if result["type"] != "bulkcreate" {
		t.Errorf("the job downgrade must still apply two versions back, got %v", result["type"])
	}
	if result["created_by_id"] != "acus_123" {
		t.Errorf("expected the creator hoisted for a preview.1 caller, got %v", result["created_by_id"])
	}
}

// --- The preview.2 contract, driven off the real resource ---

// marshalResource renders a resource exactly as the gateway would put it on the wire, so
// the tests below run against the live struct definition rather than a hand-written map.
// A fixture can drift from the struct silently; this cannot.
func marshalResource(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

// preview2JobKeys is the exact top-level key set a preview.2 client was written against.
// A pinned consumer breaks if a key it reads disappears, and an added key is a shape
// change it never agreed to — so this is asserted both ways.
var preview2JobKeys = []string{
	"id", "object", "type", "status",
	"created_by_id", "created_by_name", "created_by_username", "created_by_email",
	"results", "errors", "error_summary", "export",
	"started_at", "completed_at", "failed_at", "cancelled_at", "created_at", "updated_at",
}

func TestTransformJob_ProducesExactlyThePreview2KeySet(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	result := tr.Transform(constants.ObjectTypeJob, marshalResource(t, apiresource.SampleJob))

	for _, key := range preview2JobKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("preview.2 clients read %q; the downgrade dropped it", key)
		}
	}
	for key := range result {
		if !slices.Contains(preview2JobKeys, key) {
			t.Errorf("%q is not part of the preview.2 shape; the downgrade leaked it", key)
		}
	}
}

// The sample resource is the one the OpenAPI spec and SDK snippets are generated from, so
// downgrading it is the closest thing to a real preview.2 response.
func TestTransformJob_DowngradesTheRealSampleResource(t *testing.T) {
	t.Parallel()
	tr := &jobForgePreview3To2{}

	result := tr.Transform(constants.ObjectTypeJob, marshalResource(t, apiresource.SampleJob))

	if result["type"] != "bulkcreate" {
		t.Errorf("type: got %v", result["type"])
	}
	results, ok := result["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("results: got %v", result["results"])
	}
	row := results[0].(map[string]any)
	if row["id"] != apiresource.SampleProductionRunID {
		t.Errorf("the written row must carry the bare resource id, got %v", row["id"])
	}
	if row["action"] != "created" {
		t.Errorf("action: got %v", row["action"])
	}
	subIDs, ok := row["sub_resource_ids"].([]any)
	if !ok || len(subIDs) != 1 || subIDs[0] != apiresource.SampleBatchID {
		t.Errorf("sub_resource_ids: got %v", row["sub_resource_ids"])
	}
}

func TestTransformJob_TypeMapping(t *testing.T) {
	t.Parallel()
	cases := []struct{ latest, preview2 string }{
		{"bulk_create", "bulkcreate"},
		{"bulk_upsert", "bulkupsert"},
		{"export", "export"}, // never had a separator, so it is carried through untouched
	}
	for _, tc := range cases {
		t.Run(tc.latest, func(t *testing.T) {
			t.Parallel()
			payload := jobPreview3Payload()
			payload["type"] = tc.latest
			if got := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)["type"]; got != tc.preview2 {
				t.Errorf("%s: got %v, want %v", tc.latest, got, tc.preview2)
			}
		})
	}
}

// A job raised by an API key has no account user to attribute, and preview.2 carried four
// nulls for it rather than omitting the keys.
func TestTransformJob_NullCreatorHoistsNulls(t *testing.T) {
	t.Parallel()
	payload := jobPreview3Payload()
	payload["created_by"] = nil

	result := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)

	for _, key := range []string{"created_by_id", "created_by_name", "created_by_username", "created_by_email"} {
		v, present := result[key]
		if !present {
			t.Errorf("%s must still be present", key)
		}
		if v != nil {
			t.Errorf("%s must be null when the job has no creator, got %v", key, v)
		}
	}
}

// A batch where every row was rejected still ran, so preview.2 saw an empty results array
// rather than the null that means "nothing recorded yet".
func TestTransformJob_AllRowsFailedLeavesAnEmptyResultsArray(t *testing.T) {
	t.Parallel()
	payload := jobPreview3Payload()
	rows := payload["results"].(map[string]any)["data"].([]any)
	payload["results"].(map[string]any)["data"] = []any{rows[1]} // the failed row only

	result := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)

	results, ok := result["results"].([]any)
	if !ok || len(results) != 0 {
		t.Errorf("expected an empty array, got %#v", result["results"])
	}
	if errors, _ := result["errors"].([]any); len(errors) != 1 {
		t.Errorf("expected the rejected row in errors, got %v", result["errors"])
	}
}

// A job can fail as a whole after some rows were already rejected. preview.2 carried both
// kinds in one array, telling them apart by the absence of an index.
func TestTransformJob_RowFailuresAndAWholeJobFailureCoexist(t *testing.T) {
	t.Parallel()
	payload := jobPreview3Payload()
	payload["error"] = map[string]any{"code": "internal_error", "type": "api_error", "message": "fell over"}

	result := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)

	errors, ok := result["errors"].([]any)
	if !ok || len(errors) != 2 {
		t.Fatalf("expected the row failure and the job failure, got %v", result["errors"])
	}
	var indexed, indexless int
	for _, e := range errors {
		if _, has := e.(map[string]any)["index"]; has {
			indexed++
		} else {
			indexless++
		}
	}
	if indexed != 1 || indexless != 1 {
		t.Errorf("expected one indexed and one index-less entry, got %d/%d", indexed, indexless)
	}
	if result["error_summary"] != "fell over" {
		t.Errorf("error_summary: got %v", result["error_summary"])
	}
}

// preview.2 omitted sub_resource_ids entirely for an operation that produces none, rather
// than emitting an empty array.
func TestTransformJob_OmitsSubResourceIDsWhenThereAreNone(t *testing.T) {
	t.Parallel()
	payload := jobPreview3Payload()
	row := payload["results"].(map[string]any)["data"].([]any)[0].(map[string]any)
	row["sub_resources"] = map[string]any{"object": "list", "data": []any{}}

	result := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)

	written := result["results"].([]any)[0].(map[string]any)
	if _, present := written["sub_resource_ids"]; present {
		t.Errorf("the key must be absent, got %v", written["sub_resource_ids"])
	}
}

// The current version bounds the row list and says so via page_info. preview.2 had no way
// to express that, so the flag is dropped — the rows it did keep are still accurate.
func TestTransformJob_DropsTheTruncationFlagItCannotExpress(t *testing.T) {
	t.Parallel()
	payload := jobPreview3Payload()
	payload["results"].(map[string]any)["page_info"] = map[string]any{"has_next_page": true}

	result := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)

	results, ok := result["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("the retained rows must survive, got %v", result["results"])
	}
	for key := range result {
		if key == "page_info" || key == "truncated" {
			t.Errorf("preview.2 had no %q on a job", key)
		}
	}
}

// The transformer names only `job` in ObjectTypes, and the walk keys on the object
// discriminator — a sibling resource in the same payload must come through untouched.
func TestTransformJob_LeavesOtherObjectsAlone(t *testing.T) {
	t.Parallel()
	sibling := map[string]any{
		"object": "actor", "id": "acus_9", "name": "Jane Doe", "handle": "jane@example.com",
	}
	payload := map[string]any{
		"object": "list",
		"data":   []any{jobPreview3Payload(), sibling},
	}

	result := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)

	got := result["data"].([]any)[1].(map[string]any)
	if got["object"] != "actor" || got["name"] != "Jane Doe" || got["handle"] != "jane@example.com" {
		t.Errorf("a non-job object must be untouched, got %v", got)
	}
	if _, leaked := got["created_by_id"]; leaked {
		t.Error("the job downgrade was applied to an actor")
	}
}

// Row order is the request's order; the downgrade must not shuffle it.
func TestTransformJob_PreservesRowOrder(t *testing.T) {
	t.Parallel()
	payload := jobPreview3Payload()
	data := payload["results"].(map[string]any)["data"].([]any)
	first := data[0].(map[string]any)
	extra := map[string]any{
		"object": "job_result", "index": float64(2), "status": "updated",
		"resource": map[string]any{"id": "pnrn_2", "object": "entity", "type": "production_run"},
	}
	payload["results"].(map[string]any)["data"] = []any{first, extra, data[1]}

	result := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)

	results := result["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("expected the two written rows, got %v", results)
	}
	if results[0].(map[string]any)["index"] != float64(0) || results[1].(map[string]any)["index"] != float64(2) {
		t.Errorf("row order was not preserved: %v", results)
	}
}

// A payload with no results key at all must not panic or invent one.
func TestTransformJob_ToleratesAnAbsentResultsKey(t *testing.T) {
	t.Parallel()
	payload := jobPreview3Payload()
	delete(payload, "results")

	result := (&jobForgePreview3To2{}).Transform(constants.ObjectTypeJob, payload)

	if result["results"] != nil {
		t.Errorf("expected a null results list, got %#v", result["results"])
	}
}

// The contract's other half: a client on the current version must see the current shape,
// so the registry must not apply the transformer to a same-version request.
func TestTransformJob_LatestIsNotDowngraded(t *testing.T) {
	t.Parallel()

	result := version.Transform(
		version.Latest, version.Latest,
		constants.ObjectTypeJob,
		marshalResource(t, apiresource.SampleJob),
	)

	if result["type"] != "bulk_create" {
		t.Errorf("a same-version response must keep the current type, got %v", result["type"])
	}
	if _, ok := result["results"].(map[string]any); !ok {
		t.Errorf("results must stay a List envelope, got %T", result["results"])
	}
	if _, leaked := result["created_by_id"]; leaked {
		t.Error("the downgrade ran against the current version")
	}
	if _, ok := result["resource_type"]; !ok {
		t.Error("resource_type must survive on the current version")
	}
}
