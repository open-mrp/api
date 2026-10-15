package resourceregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

type snapshotDoc struct {
	ID     string
	ShipTo *apiresource.AddressSnapshot
}

func TestAddressSnapshotSub_ExpandsTheSourceAddress(t *testing.T) {
	t.Parallel()

	ctx := include.WithLoadMeta(context.Background())
	sourceID := "ad_src"
	saved := resourceloaders.StashAddressSnapshot(ctx, constants.ObjectTypeSalesOrder, "or_1", "ship_to", &pb.AddressSnapshot{AddressId: &sourceID, Name: "Dock", Country: "US"})
	inline := resourceloaders.StashAddressSnapshot(ctx, constants.ObjectTypeSalesOrder, "or_2", "ship_to", &pb.AddressSnapshot{Name: "Site B", Country: "CA"})
	docs := []*snapshotDoc{{ID: "or_1", ShipTo: saved}, {ID: "or_2", ShipTo: inline}, {ID: "or_3"}}

	sub := AddressSnapshotSub(constants.ObjectTypeSalesOrder, "ship_to", func(parent any) (string, *apiresource.AddressSnapshot) {
		d := parent.(*snapshotDoc)
		return d.ID, d.ShipTo
	})
	assert.Equal(t, "ship_to.address", sub.Key)

	var ids []string
	for _, d := range docs {
		ids = append(ids, sub.ExtractIDs(ctx, d)...)
	}
	assert.Equal(t, []string{"ad_src"}, ids, "only a snapshot copied from a saved address has one to expand")

	address := &apiresource.Address{ID: "ad_src", Object: constants.ObjectTypeAddress}
	for _, d := range docs {
		sub.Populate(ctx, d, map[string]any{"ad_src": address})
	}
	assert.Same(t, address, docs[0].ShipTo.Address)
	assert.Nil(t, docs[1].ShipTo.Address)
	assert.Nil(t, docs[2].ShipTo)
}
