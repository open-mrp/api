package inventorychangelogep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestInventoryChangeLogPresenter(t *testing.T) {
	t.Parallel()
	userName := "John Doe"
	stationName := "Station A"
	userID := "us_01abc"
	stationID := "scst_01abc"
	itemTypeCode := "product"
	stationType := "init_batch"

	icl := &pb.InventoryChangeLogInfo{
		Id:                       "icl_01abc",
		ItemId:                   "it_01abc",
		ItemSku:                  "SKU-001",
		QuantityId:               "qty_01abc",
		QuantityValue:            "100.000000000000000000000000000000",
		QuantityUnitId:           "un_01abc",
		QuantityUnitName:         "Kilogram",
		QuantityUnitAbbreviation: "kg",
		QuantityUnitType:         "mass",
		ActionTypeCode:           "receipt",
		ScanningStationId:        &stationID,
		ScanningStationName:      &stationName,
		ItemTypeCode:             &itemTypeCode,
		ScanningStationType:      &stationType,
		ResponsibleUserId:        &userID,
		ResponsibleUserName:      &userName,
		CreatedAt:                timestamppb.Now(),
		UpdatedAt:                timestamppb.Now(),
	}

	result := InventoryChangeLogPresenter(icl)
	resourcetest.ValidateResourceStruct(t, "InventoryChangeLog", result)
	resourcetest.ValidateExpandableStubs(t, "InventoryChangeLog", result)
}
