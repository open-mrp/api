package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// OrderDiscount is a pure leaf resource — scalars + a denormalized
	// order_count subquery only. Same empty-Subs Definition pattern as
	// AccountGroup, Address, and ProductType.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeOrderDiscount,
		Load:       resourceloaders.LoadOrderDiscounts,
	})
}
