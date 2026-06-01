package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// SysProperty is an account-scoped counter with inline SysPropertyType
	// built from denormalized proto fields.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSysProperty,
		Load:       resourceloaders.LoadSysProperties,
	})
}
