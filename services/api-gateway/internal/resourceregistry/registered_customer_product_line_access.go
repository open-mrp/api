package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// Customer-keyed twin of AccountGroupProductLineAccess: the same empty-Subs shape, embedding the customer and its product lines.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeCustomerProductLineAccess,
		Load:       resourceloaders.LoadCustomerProductLineAccess,
	})
}
