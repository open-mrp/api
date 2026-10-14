package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeTenancy,
		Load:       resourceloaders.LoadTenancies,
	})
}
