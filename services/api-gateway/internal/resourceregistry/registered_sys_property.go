package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// SysProperty is an account-scoped counter with inline SysPropertyType built from denormalized proto fields.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSysProperty,
		Load:       resourceloaders.LoadSysProperties,
	})
}
