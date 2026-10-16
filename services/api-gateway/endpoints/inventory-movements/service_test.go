package inventorymovementep

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resource/resourcetest"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/include"
)

func TestMovementFromProto(t *testing.T) {
	t.Parallel()
	ctx := include.WithLoadMeta(context.Background())
	reason := "damaged"
	locationID, lotID, lotNumber := "loc_1", "lot_1", "L-7"
	relatedType, relatedID := "shipment", "shp_1"
	userID := "us_1"
	info := &pb.InventoryChangeLogInfo{
		Id: "inmv_1", ItemId: "it_1", QuantityValue: "-4.000000", QuantityUnitId: "un_1", QuantityUnitType: "count",
		Type: "adjustment", Actor: &pb.Actor{Type: "user", Id: &userID}, Reason: &reason,
		LocationId: &locationID, LotId: &lotID, LotNumber: &lotNumber, RelatedType: &relatedType, RelatedId: &relatedID,
		UnitCost:  &pb.UnitPrice{Amount: "2.50", Currency: "usd", PerUnitId: "un_1"},
		CreatedAt: timestamppb.Now(),
	}

	mv := MovementFromProto(ctx, info)
	assert.Equal(t, constants.InventoryMovementTypeAdjustment, mv.Type)
	assert.Equal(t, "-4", mv.Quantity.Value)
	require.NotNil(t, mv.Reason)
	assert.Equal(t, constants.InventoryMovementReasonDamaged, *mv.Reason)
	assert.Equal(t, constants.ActorTypeUser, mv.Actor.Type)
	assert.Equal(t, mv.CreatedAt, mv.OccurredAt)
	assert.Nil(t, mv.Item, "item is expandable")
	assert.Nil(t, mv.Lot, "lot is expandable")
	assert.Nil(t, mv.Related, "related is expandable")

	meta := include.GetLoadMeta(ctx)
	itemID, _ := meta.GetString(constants.ObjectTypeInventoryMovement, "inmv_1", "item_id")
	assert.Equal(t, "it_1", itemID)
	lot, ok := meta.Get(constants.ObjectTypeInventoryMovement, "inmv_1", "lot")
	require.True(t, ok)
	assert.Equal(t, "L-7", lot.(*apiresource.Lot).LotNumber)

	mv.Item, mv.Location, mv.Lot, mv.Related = apiresource.SampleItem, apiresource.SampleLocation, apiresource.SampleLot, apiresource.SampleRecord
	mv.Quantity.Unit, mv.UnitCost.PerUnit = apiresource.SampleUnit, apiresource.SampleUnit
	resourcetest.ValidateResourceStruct(t, "InventoryMovement", mv)
}

func TestLevelFromProto_AsOfLeavesDemandNull(t *testing.T) {
	t.Parallel()
	location := "loc_1"
	level := LevelFromProto(&pb.InventoryLevelInfo{ItemId: "it_1", LocationId: &location, UnitId: "un_1", OnHand: "12.500"})
	assert.Equal(t, "12.5", level.OnHand.Value)
	assert.Nil(t, level.Reserved)
	assert.Nil(t, level.AvailableToPromise)
	assert.Nil(t, level.Short)
	itemID, locationID := level.Refs()
	assert.Equal(t, "it_1", itemID)
	assert.Equal(t, "loc_1", locationID)
}

func TestLotSizeDefaultFromProto(t *testing.T) {
	t.Parallel()
	category, unit := "itct_1", "un_1"
	d := LotSizeDefaultFromProto(&pb.GetItemLotDefaultResponse{ItemId: "it_1", Quantity: 60, UnitId: &unit, Source: "item_category", ItemCategoryId: &category})
	require.NotNil(t, d.LotSize)
	assert.Equal(t, "60", d.LotSize.Value)
	require.NotNil(t, d.Source)
	assert.Equal(t, constants.ItemLotSourceItemCategory, *d.Source)
	_, categoryID := d.Refs()
	assert.Equal(t, "itct_1", categoryID)

	none := LotSizeDefaultFromProto(&pb.GetItemLotDefaultResponse{ItemId: "it_1", UnitId: &unit})
	assert.Nil(t, none.LotSize, "no lot convention is null, not 0")
	assert.Nil(t, none.Source)
}
