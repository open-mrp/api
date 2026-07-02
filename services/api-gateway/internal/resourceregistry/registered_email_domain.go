package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// Email domains have no expandable sub-resources of their own; they are only
	// registered so they can be loaded as an email inbox's ?include=email_domain.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeEmailDomain,
		Load:       resourceloaders.LoadEmailDomains,
	})
}
