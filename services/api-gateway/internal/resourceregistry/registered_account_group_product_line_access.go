package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// AccountGroupProductLineAccess pairs an account group with the product lines accessible to it. Both nested fields are non-expandable — the loader materializes them from denormalized proto fields, so the definition registers no Subs.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAccountGroupProductLineAccess,
		Load:       resourceloaders.LoadAccountGroupProductLineAccess,
	})
}
