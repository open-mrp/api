//go:build e2e

package api_test

import (
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Document settings: GET /v1/identity/document-settings, GET/PATCH /v1/identity/document-settings/{document_type}.
//
// Every write goes to an account the test registers for itself, so a saved setting never leaks into
// another test's reads and a rerun starts from unsaved defaults.

const documentSettingsPath = "/v1/identity/document-settings"

// documentTypes is every document type, in the order the list returns them.
var documentTypes = []string{
	"invoice", "order_acknowledgement", "purchase_order", "pack_list",
	"pick_ticket", "batch_traveler", "price_list", "transaction_receipt",
}

func getDocumentSetting(t *testing.T, c *Client, documentType string) map[string]any {
	t.Helper()
	status, body, err := c.GetListRaw(documentSettingsPath+"/"+documentType, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func patchDocumentSetting(t *testing.T, c *Client, documentType string, body map[string]any) map[string]any {
	t.Helper()
	status, resp, err := c.Patch(documentSettingsPath+"/"+documentType, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	return parseJSON(resp)
}

// assertUnsavedDocumentSetting asserts the shape of a type the account has never saved.
func assertUnsavedDocumentSetting(t *testing.T, got map[string]any, documentType string) {
	t.Helper()
	assertObjectField(t, got, "document_setting")
	assert.Equal(t, documentType, jsonField(got, "document_type"))
	assertNilField(t, got, "id")
	assertNilField(t, got, "footer_text")
	assertNilField(t, got, "created_at")
	assertNilField(t, got, "updated_at")
	control := jsonObject(got, "document_control")
	require.NotNil(t, control, "document_control is always an object: %v", got)
	assertObjectField(t, control, "document_control")
	assertNilField(t, control, "process_owner")
	assertNilField(t, control, "document_number")
	assertNilField(t, control, "revision")
}

// --- List ---

func TestDocumentSettings_ListUnsavedDefaults(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	status, body, err := acct.owner.GetListRaw(documentSettingsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assertObjectField(t, got, "list")

	data := jsonArray(got, "data")
	require.Len(t, data, len(documentTypes))
	for i, documentType := range documentTypes {
		assertUnsavedDocumentSetting(t, data[i].(map[string]any), documentType)
	}
}

func TestDocumentSettings_ListShowsSavedType(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	saved := patchDocumentSetting(t, acct.owner, "batch_traveler", map[string]any{"footer_text": "Saved"})

	status, body, err := acct.owner.GetListRaw(documentSettingsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	data := jsonArray(parseJSON(body), "data")
	require.Len(t, data, len(documentTypes))
	for i, documentType := range documentTypes {
		entry := data[i].(map[string]any)
		if documentType != "batch_traveler" {
			assertUnsavedDocumentSetting(t, entry, documentType)
			continue
		}
		assert.Equal(t, jsonField(saved, "id"), jsonField(entry, "id"))
		assert.Equal(t, "Saved", jsonField(entry, "footer_text"))
	}
}

// --- CRUD ---

func TestDocumentSettings_RetrieveUnsaved(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	assertUnsavedDocumentSetting(t, getDocumentSetting(t, acct.owner, "invoice"), "invoice")
}

func TestDocumentSettings_UpdateAllFields(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	got := patchDocumentSetting(t, acct.owner, "batch_traveler", map[string]any{
		"document_control": map[string]any{
			"process_owner":   "Quality Manager",
			"document_number": "FRM-QUAL-001",
			"revision":        "Rev. Date 10/18/21",
		},
		"footer_text": "Retain with the batch record.",
	})

	assertIDFormat(t, jsonField(got, "id"), id.DocumentSettingIDPrefix)
	assertObjectField(t, got, "document_setting")
	assert.Equal(t, "batch_traveler", jsonField(got, "document_type"))
	assert.Equal(t, "Retain with the batch record.", jsonField(got, "footer_text"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	control := jsonObject(got, "document_control")
	assertObjectField(t, control, "document_control")
	assert.Equal(t, "Quality Manager", jsonField(control, "process_owner"))
	assert.Equal(t, "FRM-QUAL-001", jsonField(control, "document_number"))
	assert.Equal(t, "Rev. Date 10/18/21", jsonField(control, "revision"))

	assert.Equal(t, got, getDocumentSetting(t, acct.owner, "batch_traveler"), "retrieve reads back what the update returned")

	// A second update changes only what it names, on the same setting.
	again := patchDocumentSetting(t, acct.owner, "batch_traveler", map[string]any{
		"document_control": map[string]any{"revision": "Rev. D"},
	})
	assert.Equal(t, jsonField(got, "id"), jsonField(again, "id"))
	assert.Equal(t, jsonField(got, "created_at"), jsonField(again, "created_at"))
	assert.Equal(t, "Retain with the batch record.", jsonField(again, "footer_text"))
	againControl := jsonObject(again, "document_control")
	assert.Equal(t, "Quality Manager", jsonField(againControl, "process_owner"))
	assert.Equal(t, "FRM-QUAL-001", jsonField(againControl, "document_number"))
	assert.Equal(t, "Rev. D", jsonField(againControl, "revision"))

	// Other types are untouched.
	assertUnsavedDocumentSetting(t, getDocumentSetting(t, acct.owner, "invoice"), "invoice")
}

func TestDocumentSettings_NullClearsFields(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	patchDocumentSetting(t, acct.owner, "pack_list", map[string]any{
		"document_control": map[string]any{
			"process_owner":   "Shipping Lead",
			"document_number": "FRM-SHIP-002",
			"revision":        "Rev. A",
		},
		"footer_text": "Thank you for your order.",
	})

	got := patchDocumentSetting(t, acct.owner, "pack_list", map[string]any{
		"document_control": map[string]any{"process_owner": nil},
		"footer_text":      nil,
	})
	assertNilField(t, got, "footer_text")
	control := jsonObject(got, "document_control")
	assertNilField(t, control, "process_owner")
	assert.Equal(t, "FRM-SHIP-002", jsonField(control, "document_number"))
	assert.Equal(t, "Rev. A", jsonField(control, "revision"))
}

// --- Idempotency ---

func TestDocumentSettings_UpdateIdempotency(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	key := newIdempotencyKey()
	path := documentSettingsPath + "/invoice"
	body := map[string]any{"footer_text": "Payment due within 30 days."}

	status, first, err := acct.owner.Patch(path, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, first)
	status, second, err := acct.owner.Patch(path, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, second)

	assert.Equal(t, jsonField(parseJSON(first), "id"), jsonField(parseJSON(second), "id"))
	assert.Equal(t, jsonField(parseJSON(first), "updated_at"), jsonField(parseJSON(second), "updated_at"))
}

// --- Validation ---

func TestDocumentSettings_Validation(t *testing.T) {
	t.Parallel()

	t.Run("unknown document type on retrieve", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(documentSettingsPath+"/letterhead", nil)
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("unknown document type on update", func(t *testing.T) {
		status, body, err := apiClient.Patch(documentSettingsPath+"/letterhead", map[string]any{"footer_text": "x"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("empty body", func(t *testing.T) {
		status, body, err := apiClient.Patch(documentSettingsPath+"/invoice", map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("null document_control", func(t *testing.T) {
		status, body, err := apiClient.Patch(documentSettingsPath+"/invoice", map[string]any{"document_control": nil}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("revision too long", func(t *testing.T) {
		status, body, err := apiClient.Patch(documentSettingsPath+"/invoice", map[string]any{
			"document_control": map[string]any{"revision": strings.Repeat("r", 256)},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("footer too long", func(t *testing.T) {
		status, body, err := apiClient.Patch(documentSettingsPath+"/invoice", map[string]any{
			"footer_text": strings.Repeat("f", 2001),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})
}
