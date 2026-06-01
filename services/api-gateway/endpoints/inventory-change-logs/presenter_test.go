package inventorychangelogep

import (
	"testing"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestInventoryChangeLogFromProto(t *testing.T) {
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

	result := inventoryChangeLogFromProto(icl)
	result.Item = &apiresource.Item{
		ID:           icl.ItemId,
		Object:       "item",
		SKU:          icl.ItemSku,
		ItemTypeCode: "product",
	}
	result.Quantity = &apiresource.Quantity{
		ID:           icl.QuantityId,
		Object:       "quantity",
		Value:        icl.QuantityValue,
		DisplayValue: apiresource.FormatDisplayValue(icl.QuantityValue, icl.QuantityUnitAbbreviation, icl.QuantityUnitType),
	}
	result.ResponsibleUser = &apiresource.User{
		ID:     userID,
		Object: "user",
	}
	result.ResponsibleScanningStation = &apiresource.ScanningStation{
		ID:     stationID,
		Object: "scanning_station",
		Name:   stationName,
		Type:   "init_batch",
	}
	resourcetest.ValidateResourceStruct(t, "InventoryChangeLog", result)
	resourcetest.ValidateExpandableStubs(t, "InventoryChangeLog", result)
}
