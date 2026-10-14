package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// SysProperty is an account-scoped counter with inline SysPropertyType built from denormalized proto fields.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSysProperty,
		Load:       resourceloaders.LoadSysProperties,
	})
}
