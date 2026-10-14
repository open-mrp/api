package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// Customer-keyed twin of AccountGroupProductLineAccess: the same empty-Subs shape, embedding the customer and its product lines.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeCustomerProductLineAccess,
		Load:       resourceloaders.LoadCustomerProductLineAccess,
	})
}
