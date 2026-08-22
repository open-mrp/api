package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePermissionGroup,
		Load:       resourceloaders.LoadPermissionGroups,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnPermissionGroup},
		},
	})
}

func populateOwnerOnPermissionGroup(_ context.Context, parent any, _ map[string]any) {
	parent.(*apiresource.PermissionGroup).Owner = &apiresource.Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeSystem,
	}
}
