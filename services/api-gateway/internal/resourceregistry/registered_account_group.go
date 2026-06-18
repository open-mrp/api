package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// AccountGroup is a pure leaf resource: scalars + enums only, no expandable sub-resources. The Definition declares no Subs so the V2 resolver will reject any `?include[]=...` request when paired with an empty endpoint allow-list.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAccountGroup,
		Load:       resourceloaders.LoadAccountGroups,
	})
}
