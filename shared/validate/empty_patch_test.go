package validate

import (
	"testing"

	"github.com/open-mrp/api/shared/field"
	"github.com/stretchr/testify/assert"
)

type patchReqBasic struct {
	ID   string  `path:"id" validate:"required"`
	Name *string `json:"name,omitempty"`
	Note *string `json:"note,omitempty"`
}

type patchReqAllNonBody struct {
	ID    string `path:"id" validate:"required"`
	Token string `header:"X-Token"`
	Q     string `query:"q"`
}

type PatchEmbeddedFields struct {
	Name *string `json:"name,omitempty"`
}

type patchReqEmbedded struct {
	ID string `path:"id"`
	PatchEmbeddedFields
}

func TestRejectEmptyPatchBody_EmptyObject(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{}`), &patchReqBasic{})
	assert.NotNil(t, err, "empty body should be rejected")
}

func TestRejectEmptyPatchBody_UnknownFieldsOnly(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{"unknown_field": "value"}`), &patchReqBasic{})
	assert.NotNil(t, err, "body with only unknown fields should be rejected")
}

func TestRejectEmptyPatchBody_ValidField(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{"name": "test"}`), &patchReqBasic{})
	assert.Nil(t, err, "body with a valid field should be accepted")
}

func TestRejectEmptyPatchBody_MixedValidAndUnknown(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{"name": "test", "unknown": 1}`), &patchReqBasic{})
	assert.Nil(t, err, "body with at least one valid field should be accepted")
}

func TestRejectEmptyPatchBody_OnlyNonBodyTags(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{}`), &patchReqAllNonBody{})
	assert.NotNil(t, err, "struct with only path/query/header fields and empty body should be rejected")
}

func TestRejectEmptyPatchBody_NonBodyFieldInBody(t *testing.T) {
	t.Parallel(
	// Sending a field name that matches a path-tagged field should still be rejected
	// because path fields are not body fields.
	)

	err := RejectEmptyPatchBody([]byte(`{"id": "abc"}`), &patchReqAllNonBody{})
	assert.NotNil(t, err, "path-tagged field sent in body should not count as a body field")
}

func TestRejectEmptyPatchBody_EmbeddedStruct(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{}`), &patchReqEmbedded{})
	assert.NotNil(t, err, "empty body with embedded struct should be rejected")

	err = RejectEmptyPatchBody([]byte(`{"name": "test"}`), &patchReqEmbedded{})
	assert.Nil(t, err, "valid field from embedded struct should be accepted")
}

func TestRejectEmptyPatchBody_NonStruct(t *testing.T) {
	t.Parallel()
	s := "just a string"
	err := RejectEmptyPatchBody([]byte(`{}`), &s)
	assert.Nil(t, err, "non-struct input should be accepted (no-op)")
}

func TestRejectEmptyPatchBody_EmptyBytes(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte{}, &patchReqBasic{})
	assert.Nil(t, err, "empty bytes should be accepted (no body to validate)")
}

func TestRejectEmptyPatchBody_NilPointer(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{}`), (*patchReqBasic)(nil))
	assert.Nil(t, err, "nil pointer should be accepted (no-op)")
}

type patchReqValueFields struct {
	ID   string                  `path:"id" validate:"required"`
	Name field.Optional[string]  `json:"name,omitzero"`
	Note field.Clearable[string] `json:"note,omitzero"`
}

func TestRejectEmptyPatchBody_ValueFieldsEmptyObject(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{}`), &patchReqValueFields{})
	assert.NotNil(t, err, "empty body should be rejected on an Optional/Clearable request")
}

func TestRejectEmptyPatchBody_OptionalFieldPresent(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{"name": "test"}`), &patchReqValueFields{})
	assert.Nil(t, err, "a field.Optional key should count as a body field")
}

func TestRejectEmptyPatchBody_ClearableFieldCleared(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{"note": null}`), &patchReqValueFields{})
	assert.Nil(t, err, "clearing a field.Clearable is an update, not an empty body")
}

func TestRejectEmptyPatchBody_ValueFieldsUnknownFieldsOnly(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{"unknown_field": "value"}`), &patchReqValueFields{})
	assert.NotNil(t, err, "body with only unknown fields should be rejected")
}

// Non-object bodies are the decoder's to reject; this guard only ever inspects a JSON object.
func TestRejectEmptyPatchBody_NonObjectBodies(t *testing.T) {
	t.Parallel()
	for _, body := range []string{`[]`, `[{"name":"test"}]`, `"just a string"`, `null`, `123`} {
		err := RejectEmptyPatchBody([]byte(body), &patchReqBasic{})
		assert.Nil(t, err, "non-object body %s should be a no-op", body)
	}
}

func TestRejectEmptyPatchBody_MalformedJSON(t *testing.T) {
	t.Parallel()
	err := RejectEmptyPatchBody([]byte(`{"name": `), &patchReqBasic{})
	assert.Nil(t, err, "a body that does not parse should be a no-op")
}
