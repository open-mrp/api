package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// AccountGroupProductLineAccess pairs an account group with the product lines accessible to it. Both nested fields are non-expandable — the loader embeds them from the group and product line loaders, so the definition registers no Subs.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAccountGroupProductLineAccess,
		Load:       resourceloaders.LoadAccountGroupProductLineAccess,
	})
}
