package emailbridgeep

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// emailInboxIncludeFields is the whitelist of ?include= keys exposed by the email inbox endpoints. Each references another resource by id, fetched via that resource's loader when requested.
var emailInboxIncludeFields = []string{
	"email_domain",
	"group",
	"agent_config",
}

func emailInboxIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeEmailInbox,
		Fields:     emailInboxIncludeFields,
	})
}

func emailSenderIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeEmailSender,
		Fields:     []string{"email_domain"},
	})
}
