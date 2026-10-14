package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// AccountIntegration is a pure leaf — scalar fields + integration code + is_active. Same empty-Subs Definition as AccountGroup/Address.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAccountIntegration,
		Load:       resourceloaders.LoadAccountIntegrations,
	})
}
