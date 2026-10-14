package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// ChildAccount has an always-present (non-expandable) inline Account reference built from denormalized proto fields. Empty-Subs Definition.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeChildAccount,
		Load:       resourceloaders.LoadChildAccounts,
	})
}
