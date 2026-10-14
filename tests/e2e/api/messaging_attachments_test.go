//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase-4 slice-3 coverage: the attachment upload pipeline. The e2e object store is a permissive
// stub (presigned URLs are stubbed and FileExists reports true), so this exercises the API contract
// — minting an upload target, attaching by s3_key on send, and surfacing attachments on read — not
// real S3 transfer.

func createUploadURL(t *testing.T, c *Client, conversationID, filename, contentType string) map[string]any {
	t.Helper()
	resp, err := c.PostFull(conversationsPath+"/"+conversationID+"/attachments/actions/upload-url", map[string]any{
		"filename":     filename,
		"content_type": contentType,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

func TestAttachments_UploadURLShape(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	target := createUploadURL(t, dane, convID, "diagram.png", "image/png")
	assert.Equal(t, "attachment_upload_target", jsonField(target, "object"))
	assert.Nil(t, target["attachment"], "attachment is null without include")
	assert.NotEmpty(t, jsonField(target, "upload_url"))
	s3Key := jsonField(target, "s3_key")
	// Uploads land under a staging prefix scoped to the account + conversation; they are promoted to
	// the permanent chat/ prefix on send, and unattached staged objects expire via bucket lifecycle.
	assert.True(t, strings.HasPrefix(s3Key, "staged/"+SeedAccountID+"/"+convID+"/"), "s3 key is a staging key scoped to the account + conversation, got %q", s3Key)
	assert.True(t, strings.HasSuffix(s3Key, "/diagram.png"))
}

func TestAttachments_UploadURLIncludeAttachment(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	resp, err := dane.PostFull(withQuery(conversationsPath+"/"+convID+"/attachments/actions/upload-url", url.Values{"include": {"attachment"}}), map[string]any{
		"filename":     "diagram.png",
		"content_type": "image/png",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	target := parseJSON(resp.Body)

	att, ok := target["attachment"].(map[string]any)
	require.True(t, ok, "attachment is expanded when requested")
	assert.Equal(t, "message_attachment", jsonField(att, "object"))
	assert.Equal(t, "image", jsonField(att, "kind"))
	assert.Equal(t, "diagram.png", jsonField(att, "filename"))
	assert.Equal(t, "image/png", jsonField(att, "content_type"))
	assertIDFormat(t, jsonField(att, "id"), "mgah")
}

func TestAttachments_SendWithImageAttachment(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	target := createUploadURL(t, dane, convID, "diagram.png", "image/png")
	s3Key := jsonField(target, "s3_key")

	// Send referencing the (stub-uploaded) object by s3_key.
	resp, err := dane.PostFull(withQuery(conversationsPath+"/"+convID+"/messages", messageIncludeQuery), map[string]any{
		"body":              "see attached",
		"client_message_id": uniqueName("cmid"),
		"attachments": []map[string]any{
			{"kind": "image", "s3_key": s3Key, "filename": "diagram.png", "content_type": "image/png", "size_bytes": 1024},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	msg := parseJSON(resp.Body)

	atts, ok := listData(msg, "attachments")
	require.True(t, ok, "the message carries attachments")
	require.Len(t, atts, 1)
	att, _ := atts[0].(map[string]any)
	assert.Equal(t, "message_attachment", jsonField(att, "object"))
	assert.Equal(t, "image", jsonField(att, "kind"))
	assert.Equal(t, "diagram.png", jsonField(att, "filename"))
	assert.NotEmpty(t, jsonField(att, "url"), "a presigned download url is surfaced on read")
	assertIDFormat(t, jsonField(att, "id"), "mgah")
}

func TestAttachments_ForeignKeyRejected(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	// A key scoped to a different conversation must be rejected even though the stub reports it exists.
	resp, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              "sneaky",
		"client_message_id": uniqueName("cmid"),
		"attachments": []map[string]any{
			{"kind": "image", "s3_key": "staged/" + SeedAccountID + "/cv_someoneelse/mgah_x/secret.png", "filename": "secret.png"},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, resp.StatusCode, resp.Body)
}

func TestAttachments_ResourceAttachment(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	// attachments.record is expandable — request the nested include so the linked record's stub is loaded.
	attachmentIncludeQuery := url.Values{"include": {"attachments", "attachments.record"}}
	resp, err := dane.PostFull(withQuery(conversationsPath+"/"+convID+"/messages", attachmentIncludeQuery), map[string]any{
		"body":              "linking an order",
		"client_message_id": uniqueName("cmid"),
		"attachments": []map[string]any{
			{"kind": "resource", "record": map[string]any{"type": "sales_order", "id": SeedSalesOrderID}},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	msg := parseJSON(resp.Body)
	atts, _ := listData(msg, "attachments")
	require.Len(t, atts, 1)
	att, _ := atts[0].(map[string]any)
	assert.Equal(t, "resource", jsonField(att, "kind"))
	res, _ := att["record"].(map[string]any)
	require.NotNil(t, res, "a resource attachment carries the linked record")
	assert.Equal(t, "record", jsonField(res, "object"))
	assert.Equal(t, "sales_order", jsonField(res, "type"))
	assert.Equal(t, SeedSalesOrderID, jsonField(res, "id"))
	assert.NotEmpty(t, jsonField(res, "number"))
}
