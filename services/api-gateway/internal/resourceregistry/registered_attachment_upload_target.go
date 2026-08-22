package resourceregistry

import (
	"context"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAttachmentUploadTarget,
		Load: func(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
			return nil, nil
		},
		Subs: []resourcekit.SubField{
			// attachment is built inline by the upload-url service and stashed in LoadMeta keyed by s3_key; ExtractRefs lets the resolver recurse into attachment.resource.
			{
				Key:         "attachment",
				Target:      constants.ObjectTypeMessageAttachment,
				Cardinality: resourcekit.CardinalityOnePtr,
				Populate:    populateAttachmentOnUploadTarget,
				ExtractRefs: extractAttachmentRefFromUploadTarget,
			},
		},
	})
}

func populateAttachmentOnUploadTarget(ctx context.Context, parent any, _ map[string]any) {
	t := parent.(*apiresource.AttachmentUploadTarget)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeAttachmentUploadTarget, t.S3Key, "attachment")
	if !ok {
		return
	}
	t.Attachment = v.(*apiresource.MessageAttachment)
}

func extractAttachmentRefFromUploadTarget(_ context.Context, parent any) []any {
	t := parent.(*apiresource.AttachmentUploadTarget)
	if t.Attachment == nil {
		return nil
	}
	return []any{t.Attachment}
}
