package attachmentep

import (
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/constants"
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
