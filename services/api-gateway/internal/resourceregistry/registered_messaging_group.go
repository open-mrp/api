package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// Rosters are loaded by ID when another resource includes one, such as an email inbox's group.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeMessagingGroup,
		Load:       resourceloaders.LoadMessagingGroups,
	})
}
