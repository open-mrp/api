package attachmentep

import (
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"github.com/open-mrp/api/shared/constants"
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
