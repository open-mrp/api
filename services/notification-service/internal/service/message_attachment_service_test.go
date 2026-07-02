package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/notification-service/internal/domain"
	s3 "github.com/augno/api/shared/cloud/s3"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attachmentSvc(fileExists bool) *conversationSvcImpl {
	return &conversationSvcImpl{
		objectStore: &s3.StubClient{FileExistsResult: fileExists},
		chatBucket:  "chat-bucket",
	}
}

func ap(s string) *string { return &s }

func TestBuildAttachments_UploadedHappyPath(t *testing.T) {
	svc := attachmentSvc(true)
	stagedKey := "staged/ac_1/cv_1/mgah_x/diagram.png"
	out, apiErr := svc.buildAttachments(context.Background(), "cv_1", "ac_1", "mg_1", []domain.AttachmentInput{
		{Kind: string(constants.MessageAttachmentKindImage), S3Key: ap(stagedKey), Filename: ap("diagram.png"), ContentType: ap("image/png")},
	})
	require.Nil(t, apiErr)
	require.Len(t, out, 1)
	assert.Equal(t, "mg_1", out[0].MessageID)
	require.NotNil(t, out[0].S3Key)
	// The row references the promoted permanent key, not the staging key.
	assert.Equal(t, "chat/ac_1/cv_1/mgah_x/diagram.png", *out[0].S3Key)
	assertIDPrefix(t, out[0].ID, "mgah")
}

func TestBuildAttachments_WrongPrefixRejected(t *testing.T) {
	svc := attachmentSvc(true)
	// A staging key for a different conversation must be rejected (prevents attaching others' objects).
	_, apiErr := svc.buildAttachments(context.Background(), "cv_1", "ac_1", "mg_1", []domain.AttachmentInput{
		{Kind: string(constants.MessageAttachmentKindImage), S3Key: ap("staged/ac_1/cv_OTHER/mgah_x/x.png")},
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeParameterInvalid, apiErr.Code)
}

func TestBuildAttachments_MissingObjectRejected(t *testing.T) {
	svc := attachmentSvc(false) // FileExists returns false
	_, apiErr := svc.buildAttachments(context.Background(), "cv_1", "ac_1", "mg_1", []domain.AttachmentInput{
		{Kind: string(constants.MessageAttachmentKindImage), S3Key: ap("staged/ac_1/cv_1/mgah_x/x.png")},
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeParameterInvalid, apiErr.Code)
}

func TestBuildAttachments_UploadedMissingKeyRejected(t *testing.T) {
	svc := attachmentSvc(true)
	_, apiErr := svc.buildAttachments(context.Background(), "cv_1", "ac_1", "mg_1", []domain.AttachmentInput{
		{Kind: string(constants.MessageAttachmentKindFile)},
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeParameterMissing, apiErr.Code)
}

func TestBuildAttachments_LinkRequiresURL(t *testing.T) {
	svc := attachmentSvc(true)
	_, apiErr := svc.buildAttachments(context.Background(), "cv_1", "ac_1", "mg_1", []domain.AttachmentInput{
		{Kind: string(constants.MessageAttachmentKindLink)},
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeParameterMissing, apiErr.Code)
}

func TestBuildAttachments_ResourceHappyPath(t *testing.T) {
	svc := attachmentSvc(true)
	out, apiErr := svc.buildAttachments(context.Background(), "cv_1", "ac_1", "mg_1", []domain.AttachmentInput{
		{Kind: string(constants.MessageAttachmentKindResource), ResourceType: ap("sales_order"), ResourceID: ap("so_1")},
	})
	require.Nil(t, apiErr)
	require.Len(t, out, 1)
	require.NotNil(t, out[0].ResourceID)
	assert.Equal(t, "so_1", *out[0].ResourceID)
}

func TestBuildAttachments_InvalidKindRejected(t *testing.T) {
	svc := attachmentSvc(true)
	_, apiErr := svc.buildAttachments(context.Background(), "cv_1", "ac_1", "mg_1", []domain.AttachmentInput{
		{Kind: "bogus"},
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeParameterInvalid, apiErr.Code)
}

func assertIDPrefix(t *testing.T, id, prefix string) {
	t.Helper()
	assert.True(t, len(id) > len(prefix)+1 && id[:len(prefix)+1] == prefix+"_", "id %q should have prefix %q", id, prefix)
}
