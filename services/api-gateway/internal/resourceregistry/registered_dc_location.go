package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// DCLocation has an inline (non-expandable) DCLocationCustomer reference built from denormalized proto fields.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeDCLocation,
		Load:       resourceloaders.LoadDCLocations,
	})
}
