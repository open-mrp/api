package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// AccountGroup is a pure leaf resource: scalars + enums only, no expandable sub-resources. The Definition declares no Subs so the V2 resolver will reject any `?include[]=...` request when paired with an empty endpoint allow-list.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAccountGroup,
		Load:       resourceloaders.LoadAccountGroups,
	})
}
