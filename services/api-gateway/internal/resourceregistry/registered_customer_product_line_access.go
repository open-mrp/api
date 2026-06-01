package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// Customer-keyed twin of AccountGroupProductLineAccess. Same empty-Subs
	// shape with denormalized Customer + inline ProductLines list.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeCustomerProductLineAccess,
		Load:       resourceloaders.LoadCustomerProductLineAccess,
	})
}
