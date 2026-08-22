package emailbridgeep

import (
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"github.com/open-mrp/api/shared/constants"
)

// emailInboxIncludeFields is the whitelist of ?include= keys exposed by the email
// inbox endpoints. Each references another resource by id (the bound domain and
// agent definition), fetched via that resource's batch loader when requested.
var emailInboxIncludeFields = []string{
	"email_domain",
	"agent_config",
}

func emailInboxIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeEmailInbox,
		Fields:     emailInboxIncludeFields,
	})
}
