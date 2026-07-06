package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// Portal domains have no expandable sub-resources of their own; they are registered so they can be hydrated as a sub-resource include where referenced.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePortalDomain,
		Load:       resourceloaders.LoadPortalDomains,
	})
}
