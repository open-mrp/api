package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeProperty,
		Load:       resourceloaders.LoadProperties,
		Subs: []include.SubField{
			{Key: "attributes", Populate: populateAttributesOnProperty},
		},
	})
}

func populateAttributesOnProperty(ctx context.Context, parent any, _ map[string]any) {
	p := parent.(*apiresource.Property)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProperty, p.ID, "attributes_list")
	if !ok || v == nil {
		return
	}
	p.Attributes = v.(*apiresource.List[apiresource.Attribute])
}
