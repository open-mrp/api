package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAttribute,
		Load:       resourceloaders.LoadAttributes,
		Subs: []include.SubField{
			{
				Key:         "property",
				Target:      constants.ObjectTypeProperty,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractPropertyIDFromAttribute,
				Populate:    populatePropertyOnAttribute,
			},
		},
	})
}

// extractPropertyIDFromAttribute loads the property only when the attribute's parent didn't read it already.
func extractPropertyIDFromAttribute(ctx context.Context, parent any) []string {
	a := parent.(*apiresource.Attribute)
	meta := include.GetLoadMeta(ctx)
	if v, ok := meta.Get(constants.ObjectTypeAttribute, a.ID, "property"); ok && v != nil {
		return nil
	}
	id, _ := meta.GetString(constants.ObjectTypeAttribute, a.ID, "property_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populatePropertyOnAttribute(ctx context.Context, parent any, loaded map[string]any) {
	a := parent.(*apiresource.Attribute)
	meta := include.GetLoadMeta(ctx)
	if v, ok := meta.Get(constants.ObjectTypeAttribute, a.ID, "property"); ok && v != nil {
		p := *v.(*apiresource.Property)
		a.Property = &p
		return
	}
	id, _ := meta.GetString(constants.ObjectTypeAttribute, a.ID, "property_id")
	if v, ok := loaded[id]; ok {
		p := *v.(*apiresource.Property)
		a.Property = &p
	}
}
