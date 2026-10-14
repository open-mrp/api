package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// Email domains have no expandable sub-resources of their own; they are only
	// registered so they can be loaded as an email inbox's ?include=email_domain.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeEmailDomain,
		Load:       resourceloaders.LoadEmailDomains,
	})
}
