package blockep

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// blockIncludeFields is the whitelist of ?include= keys exposed by the message block endpoints.
// blocked_user is fetched by id via the account_user loader; its user/role/work_center sub-objects
// recurse through the AccountUser definition's loaders.
var blockIncludeFields = []string{
	"blocked_user",
	"blocked_user.user",
	"blocked_user.role",
	"blocked_user.work_center",
}

func blockIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeMessagingBlock,
		Fields:     blockIncludeFields,
	})
}
