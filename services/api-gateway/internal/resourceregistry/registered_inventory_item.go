package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeInventoryItem,
		Load:       resourceloaders.LoadInventoryItems,
		// The on-hand figure is a ComputedQuantity built per request from the abbreviation and type
		// the core service returns. No unit id crosses that boundary, so `quantity.unit` cannot be
		// resolved and is not offered; declaring it pointed at the stored-quantity resolver, which
		// panicked on the type it was handed.
	})
}
