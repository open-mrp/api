package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeProperty,
		Load:       resourceloaders.LoadProperties,
		Subs: []resourcekit.SubField{
			{Key: "attributes", Populate: populateAttributesOnProperty},
		},
	})
}

func populateAttributesOnProperty(ctx context.Context, parent any, _ map[string]any) {
	p := parent.(*apiresource.Property)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProperty, p.ID, "attributes_list")
	if !ok || v == nil {
		return
	}
	p.Attributes = v.(*apiresource.List[apiresource.Attribute])
}
