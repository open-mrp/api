package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// ProductType is a system-only lookup with no expandable sub-resources.
	// Same empty-Subs Definition pattern as AccountGroup and Address.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeProductType,
		Load:       resourceloaders.LoadProductTypes,
	})
}
