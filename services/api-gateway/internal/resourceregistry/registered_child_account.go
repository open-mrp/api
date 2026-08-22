package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// ChildAccount has an always-present (non-expandable) inline Account reference built from denormalized proto fields. Empty-Subs Definition.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeChildAccount,
		Load:       resourceloaders.LoadChildAccounts,
	})
}
