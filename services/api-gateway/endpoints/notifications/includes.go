package notificationep

import (
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/constants"
)

// notificationIncludeFields is the whitelist of ?include= keys exposed by the notification endpoints.
// sender and resource are built inline by the service and stashed into LoadMeta; the include resolver
// surfaces each only when the caller requests it.
var notificationIncludeFields = []string{
	"sender",
	"resource",
}

func notificationIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeNotification,
		Fields:     notificationIncludeFields,
	})
}
