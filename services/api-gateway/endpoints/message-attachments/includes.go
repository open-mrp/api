package attachmentep

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

func attachmentUploadTargetIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeAttachmentUploadTarget,
		Fields: []string{
			"attachment",
			"attachment.resource",
		},
	})
}
