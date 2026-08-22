package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// AccountIntegration is a pure leaf — scalar fields + integration code + is_active. Same empty-Subs Definition as AccountGroup/Address.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAccountIntegration,
		Load:       resourceloaders.LoadAccountIntegrations,
	})
}
