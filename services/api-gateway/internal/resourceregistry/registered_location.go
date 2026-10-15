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
		ObjectType: constants.ObjectTypeLocation,
		Load:       resourceloaders.LoadLocations,
		Subs: []include.SubField{
			{
				Key:         "address",
				Target:      constants.ObjectTypeAddress,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  locationMetaID("address_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					loc := parent.(*apiresource.Location)
					if v, ok := loaded[locationMetaString(ctx, loc, "address_id")]; ok {
						loc.Address = v.(*apiresource.Address)
					}
				},
			},
			{
				Key:         "parent",
				Target:      constants.ObjectTypeLocation,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  locationMetaID("parent_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					loc := parent.(*apiresource.Location)
					if v, ok := loaded[locationMetaString(ctx, loc, "parent_id")]; ok {
						loc.Parent = v.(*apiresource.Location)
					}
				},
			},
			{
				Key:         "children",
				Target:      constants.ObjectTypeLocation,
				Cardinality: include.CardinalityList,
				ExtractIDs:  extractChildIDsFromLocation,
				Populate:    populateChildrenOnLocation,
			},
		},
	})
}

// locationMetaString is a location's load-meta string under key, "" when unset.
func locationMetaString(ctx context.Context, loc *apiresource.Location, key string) string {
	v, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeLocation, loc.ID, key)
	return v
}

// locationMetaID extracts the one ID a location's load meta holds under key.
func locationMetaID(key string) func(ctx context.Context, parent any) []string {
	return func(ctx context.Context, parent any) []string {
		if id := locationMetaString(ctx, parent.(*apiresource.Location), key); id != "" {
			return []string{id}
		}
		return nil
	}
}

func extractChildIDsFromLocation(ctx context.Context, parent any) []string {
	loc := parent.(*apiresource.Location)
	ids, _ := include.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeLocation, loc.ID, "child_ids")
	return ids
}

// populateChildrenOnLocation embeds the first children; the loader carries one more than fit, so a longer list links
// to the location list filtered to this parent.
func populateChildrenOnLocation(ctx context.Context, parent any, loaded map[string]any) {
	loc := parent.(*apiresource.Location)
	ids, _ := include.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeLocation, loc.ID, "child_ids")

	items := make([]apiresource.Location, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.Location)))
		}
	}

	loc.Children = apiresource.NewEmbeddedList(items, false, resourceloaders.LocationChildrenListPath(loc.ID), nil)
}
