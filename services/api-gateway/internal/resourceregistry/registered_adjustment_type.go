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
		ObjectType: constants.ObjectTypeAdjustmentType,
		Load:       resourceloaders.LoadAdjustmentTypes,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnAdjustmentType},
		},
	})
}

func populateOwnerOnAdjustmentType(_ context.Context, parent any, _ map[string]any) {
	parent.(*apiresource.AdjustmentType).Owner = &apiresource.Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeSystem,
	}
}
