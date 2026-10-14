package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// OrderDiscount is a pure leaf resource — scalars + a denormalized order_count subquery only. Same empty-Subs Definition pattern as AccountGroup, Address, and ProductType.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeOrderDiscount,
		Load:       resourceloaders.LoadOrderDiscounts,
	})
}
