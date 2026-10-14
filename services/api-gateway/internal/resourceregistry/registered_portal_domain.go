package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// Portal domains have no expandable sub-resources of their own; they are registered so they can be hydrated as a sub-resource include where referenced.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypePortalDomain,
		Load:       resourceloaders.LoadPortalDomains,
	})
}
