package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// Customer-keyed twin of AccountGroupProductLineAccess. Same empty-Subs shape with denormalized Customer + inline ProductLines list.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeCustomerProductLineAccess,
		Load:       resourceloaders.LoadCustomerProductLineAccess,
	})
}
