package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// Address is a leaf resource: scalars + inline (non-expandable) Geolocation only. No expandable sub-resources, so the Definition declares no Subs. Geolocation is always populated as part of the loader response.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAddress,
		Load:       resourceloaders.LoadAddresses,
	})
}
