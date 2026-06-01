package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// ChildAccount has an always-present (non-expandable) inline Account
	// reference built from denormalized proto fields. Empty-Subs Definition.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeChildAccount,
		Load:       resourceloaders.LoadChildAccounts,
	})
}
