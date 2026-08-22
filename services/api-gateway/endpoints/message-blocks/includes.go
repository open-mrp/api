package blockep

import (
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"github.com/open-mrp/api/shared/constants"
)

// blockIncludeFields is the whitelist of ?include= keys exposed by the message block endpoints.
// blocked_user is fetched by id via the account_user loader; its user/role/department sub-objects
// recurse through the AccountUser definition's loaders.
var blockIncludeFields = []string{
	"blocked_user",
	"blocked_user.user",
	"blocked_user.role",
	"blocked_user.department",
}

func blockIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeMessagingBlock,
		Fields:     blockIncludeFields,
	})
}
