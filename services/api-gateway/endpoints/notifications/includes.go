package notificationep

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// notificationIncludeFields is the whitelist of ?include= keys exposed by the notification endpoints. resource is
// preview.9's (internal).
var notificationIncludeFields = []string{
	"sender",
	"related.record",
	"resource",
}

func notificationIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeNotification,
		Fields:     notificationIncludeFields,
	})
}
