package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

// AddressSnapshotSub is the `<key>.address` include of a document's address snapshot (bill_to, ship_to): the saved
// address the snapshot came from, stashed by resourceloaders.StashAddressSnapshot. snapshotOf gives the document's ID
// and its snapshot, nil when it has none.
func AddressSnapshotSub(parentType constants.ObjectType, key string, snapshotOf func(parent any) (string, *apiresource.AddressSnapshot)) include.SubField {
	sourceID := func(ctx context.Context, parent any) string {
		parentID, snap := snapshotOf(parent)
		if snap == nil {
			return ""
		}
		id, _ := include.GetLoadMeta(ctx).GetString(parentType, parentID, resourceloaders.AddressSnapshotMetaKey(key))
		return id
	}
	return include.SubField{
		Key:         key + ".address",
		Target:      constants.ObjectTypeAddress,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			if id := sourceID(ctx, parent); id != "" {
				return []string{id}
			}
			return nil
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			id := sourceID(ctx, parent)
			if v, ok := loaded[id]; ok && id != "" {
				_, snap := snapshotOf(parent)
				snap.Address = v.(*apiresource.Address)
			}
		},
	}
}
