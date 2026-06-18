package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// DCLocation has an inline (non-expandable) DCLocationCustomer reference built from denormalized proto fields.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeDCLocation,
		Load:       resourceloaders.LoadDCLocations,
	})
}
