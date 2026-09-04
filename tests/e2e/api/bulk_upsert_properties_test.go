//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const propertiesBulkUpsertPath = propertiesPath + "/actions/bulk-upsert"

func bulkUpsertProperties(t *testing.T, properties ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(properties))
	for i, p := range properties {
		rows[i] = p
	}
	status, body, err := apiClient.Post(propertiesBulkUpsertPath, map[string]any{"properties": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

// names one property with no attributes, the shape most of these rows need
func bulkPropertyRow(name string) map[string]any {
	return map[string]any{"name": name}
}

// posts a bulk upsert, requires the 202 job acknowledgment, and returns the completed job
func bulkUpsertPropertiesJob(t *testing.T, properties ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertProperties(t, properties...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// posts a bulk upsert, follows the job to completion, and returns the created/updated ids
func bulkUpsertPropertyIDs(t *testing.T, properties ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertPropertiesJob(t, properties...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

func cleanupPropertyIDs(ids []string) {
	for _, propertyID := range ids {
		if propertyID != "" {
			apiClient.Delete(propertiesPath + "/" + propertyID)
		}
	}
}

// returns the `value` of every attribute defined under a property, in listed order
func propertyAttributeValues(t *testing.T, propertyID string) []string {
	t.Helper()
	list, _, err := apiClient.GetList(propertiesPath+"/"+propertyID+"/attributes", nil)
	require.NoError(t, err)
	values := make([]string, 0, len(list.Data))
	for _, item := range list.Data {
		values = append(values, DataItemField(item, "value"))
	}
	return values
}

func TestProperties_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	created, updated := bulkUpsertPropertyIDs(t,
		bulkPropertyRow(uniqueName("e2e-bup-prop-a")),
		bulkPropertyRow(uniqueName("e2e-bup-prop-b")),
	)
	defer cleanupPropertyIDs(created)

	require.Len(t, created, 2)
	for _, createdID := range created {
		assertIDFormat(t, createdID, id.PropertyIDPrefix)
	}
	assert.Empty(t, updated)
}

// defines a property's selectable values in the same request, in the order given, with
// the swatch honored where supplied
func TestProperties_BulkUpsert_CreatesAttributes(t *testing.T) {
	t.Parallel()

	// Attribute values are unique per account, so use per-run values.
	red := uniqueName("e2e-bup-attr-red")
	blue := uniqueName("e2e-bup-attr-blue")

	created, _ := bulkUpsertPropertyIDs(t, map[string]any{
		"name": uniqueName("e2e-bup-prop-attrs"),
		"attributes": []any{
			map[string]any{"value": red, "color": "red"},
			map[string]any{"value": blue},
		},
	})
	require.Len(t, created, 1)
	defer cleanupPropertyIDs(created)

	assert.Equal(t, []string{red, blue}, propertyAttributeValues(t, created[0]))

	list, _, err := apiClient.GetList(propertiesPath+"/"+created[0]+"/attributes", nil)
	require.NoError(t, err)
	require.Len(t, list.Data, 2)
	assert.Equal(t, "red", DataItemField(list.Data[0], "color"), "an explicit swatch is honored")
	assert.NotEmpty(t, DataItemField(list.Data[1], "color"), "an omitted swatch is assigned")
}

// re-upserts a property: only the values it does not already carry are added, none removed
func TestProperties_BulkUpsert_AttributesAreAdditive(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-prop-additive")
	first := uniqueName("e2e-bup-additive-1")
	second := uniqueName("e2e-bup-additive-2")

	created, _ := bulkUpsertPropertyIDs(t, map[string]any{
		"name":       name,
		"attributes": []any{map[string]any{"value": first}},
	})
	require.Len(t, created, 1)
	defer cleanupPropertyIDs(created)

	_, updated := bulkUpsertPropertyIDs(t, map[string]any{
		"name": name,
		"attributes": []any{
			map[string]any{"value": first},
			map[string]any{"value": second},
		},
	})
	require.Equal(t, []string{created[0]}, updated)

	assert.Equal(t, []string{first, second}, propertyAttributeValues(t, created[0]))
}

// matches an existing property by name case-insensitively and renames it to the
// request's casing
func TestProperties_BulkUpsert_UpdatesExistingCaseInsensitive(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-prop-ren")
	createResp, err := apiClient.PostFull(propertiesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)
	propertyID := jsonField(parseJSON(createResp.Body), "id")
	require.NotEmpty(t, propertyID)
	defer apiClient.Delete(propertiesPath + "/" + propertyID)

	upper := strings.ToUpper(name)
	created, updated := bulkUpsertPropertyIDs(t, bulkPropertyRow(upper))

	assert.Empty(t, created, "existing name must update, not create")
	require.Len(t, updated, 1)
	assert.Equal(t, propertyID, updated[0])

	// Rename semantics: the property now carries the request's casing.
	getStatus, getBody, err := apiClient.GetListRaw(propertiesPath+"/"+propertyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, upper, jsonField(parseJSON(getBody), "name"))
}

func TestProperties_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingName := uniqueName("e2e-bup-prop-mix-exist")
	newName := uniqueName("e2e-bup-prop-mix-new")

	seeded, _ := bulkUpsertPropertyIDs(t, bulkPropertyRow(existingName))
	defer cleanupPropertyIDs(seeded)

	created, updated := bulkUpsertPropertyIDs(t, bulkPropertyRow(existingName), bulkPropertyRow(newName))
	defer cleanupPropertyIDs(created)

	assert.Len(t, created, 1)
	assert.Len(t, updated, 1)
}

// rejects duplicate names within one request, including names differing only by casing
func TestProperties_BulkUpsert_RejectsDuplicateNameInRequest(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-prop-dup")
	status, body := bulkUpsertProperties(t, bulkPropertyRow(name), bulkPropertyRow(strings.ToUpper(name)))
	requireStatus(t, 400, status, body)

	errObj := jsonObject(parseJSON(body), "error")
	assert.Equal(t, "invalid_request_error", jsonField(errObj, "type"))
	assert.Equal(t, "properties[1].name", jsonField(errObj, "param"))
}

// rejects a value claimed by two properties: one value belongs to one property per
// account, so no job is raised
func TestProperties_BulkUpsert_RejectsValueUnderTwoProperties(t *testing.T) {
	t.Parallel()

	value := uniqueName("e2e-bup-prop-shared")
	status, body := bulkUpsertProperties(t,
		map[string]any{"name": uniqueName("e2e-bup-prop-x"), "attributes": []any{map[string]any{"value": value}}},
		map[string]any{"name": uniqueName("e2e-bup-prop-y"), "attributes": []any{map[string]any{"value": value}}},
	)
	requireStatus(t, 400, status, body)

	errObj := jsonObject(parseJSON(body), "error")
	assert.Equal(t, "invalid_request_error", jsonField(errObj, "type"))
	assert.Equal(t, "properties[1].attributes[0].value", jsonField(errObj, "param"))
}

func TestProperties_BulkUpsert_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(propertiesBulkUpsertPath, map[string]any{"properties": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// re-runs the same bulk upsert: the existing property is matched, not duplicated
func TestProperties_BulkUpsert_ReimportIsStable(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-prop-stable")

	created, _ := bulkUpsertPropertyIDs(t, bulkPropertyRow(name))
	require.Len(t, created, 1)
	defer cleanupPropertyIDs(created)

	secondCreated, secondUpdated := bulkUpsertPropertyIDs(t, bulkPropertyRow(name))
	assert.Empty(t, secondCreated)
	require.Len(t, secondUpdated, 1)
	assert.Equal(t, created[0], secondUpdated[0])

	// Exactly one property with this name exists.
	list, _, err := apiClient.GetList(propertiesPath, url.Values{"q": {name}})
	require.NoError(t, err)
	matches := 0
	for _, item := range list.Data {
		if strings.EqualFold(DataItemField(item, "name"), name) {
			matches++
		}
	}
	assert.Equal(t, 1, matches)
}
