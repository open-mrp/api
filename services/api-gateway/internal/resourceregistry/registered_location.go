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
		ObjectType: constants.ObjectTypeLocation,
		Load:       resourceloaders.LoadLocations,
		Subs: []resourcekit.SubField{
			{
				Key:         "parent",
				Target:      constants.ObjectTypeLocation,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractParentIDFromLocation,
				Populate:    populateParentOnLocation,
			},
			{
				Key:         "children",
				Target:      constants.ObjectTypeLocation,
				Cardinality: resourcekit.CardinalityList,
				ExtractIDs:  extractChildIDsFromLocation,
				Populate:    populateChildrenOnLocation,
			},
		},
	})
}

func extractParentIDFromLocation(ctx context.Context, parent any) []string {
	loc := parent.(*apiresource.Location)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeLocation, loc.ID, "parent_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateParentOnLocation(ctx context.Context, parent any, loaded map[string]any) {
	loc := parent.(*apiresource.Location)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeLocation, loc.ID, "parent_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		loc.Parent = v.(*apiresource.Location)
	}
}

func extractChildIDsFromLocation(ctx context.Context, parent any) []string {
	loc := parent.(*apiresource.Location)
	ids, _ := resourcekit.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeLocation, loc.ID, "child_ids")
	return ids
}

func populateChildrenOnLocation(ctx context.Context, parent any, loaded map[string]any) {
	loc := parent.(*apiresource.Location)
	ids, _ := resourcekit.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeLocation, loc.ID, "child_ids")

	items := make([]apiresource.Location, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.Location)))
		}
	}

	loc.Children = apiresource.NewList(items, apiresource.PageInfo{})
}
