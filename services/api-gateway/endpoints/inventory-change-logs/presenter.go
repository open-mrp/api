package inventorychangelogep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func InventoryChangeLogPresenter(icl *pb.InventoryChangeLogInfo) apiresource.InventoryChangeLog {
	if icl == nil {
		return apiresource.InventoryChangeLog{}
	}

	result := apiresource.InventoryChangeLog{
		ID:             icl.Id,
		Object:         constants.ObjectTypeInventoryChangeLog,
		ActionTypeCode: constants.InventoryActionType(icl.ActionTypeCode),
		Quantity: &apiresource.Quantity{
			ID:     icl.QuantityId,
			Object: constants.ObjectTypeQuantity,
			Value:  icl.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(
				icl.QuantityValue,
				icl.QuantityUnitAbbreviation,
				icl.QuantityUnitType,
			),
			Unit: &apiresource.Unit{
				ID:                icl.QuantityUnitId,
				Object:            constants.ObjectTypeUnit,
				Name:              icl.QuantityUnitName,
				Abbreviation:      icl.QuantityUnitAbbreviation,
				Type:              constants.UnitType(icl.QuantityUnitType),
				RatioNumerator:    icl.QuantityUnitRatioNumerator,
				RatioDenominator:  icl.QuantityUnitRatioDenominator,
				OffsetNumerator:   icl.QuantityUnitOffsetNumerator,
				OffsetDenominator: icl.QuantityUnitOffsetDenominator,
				CreatedAt:         grpcutil.TimestampToTime(icl.QuantityUnitCreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(icl.QuantityUnitUpdatedAt),
			},
		},
		Item: &apiresource.Item{
			ID:        icl.ItemId,
			Object:    constants.ObjectTypeItem,
			SKU:       icl.ItemSku,
			CreatedAt: grpcutil.TimestampToTime(icl.ItemCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(icl.ItemUpdatedAt),
		},
		CreatedAt: grpcutil.TimestampToTime(icl.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(icl.UpdatedAt),
	}

	if icl.ItemTypeCode != nil {
		result.Item.ItemTypeCode = constants.ItemTypeCode(*icl.ItemTypeCode)
	}

	if icl.ResponsibleUserId != nil {
		user := &apiresource.User{
			ID:     *icl.ResponsibleUserId,
			Object: constants.ObjectTypeUser,
			Name:   icl.ResponsibleUserName,
		}
		if icl.ResponsibleUserCreatedAt != nil {
			user.CreatedAt = icl.ResponsibleUserCreatedAt.AsTime()
		}
		if icl.ResponsibleUserUpdatedAt != nil {
			user.UpdatedAt = icl.ResponsibleUserUpdatedAt.AsTime()
		}
		result.ResponsibleUser = user
	}

	if icl.ScanningStationId != nil {
		name := ""
		if icl.ScanningStationName != nil {
			name = *icl.ScanningStationName
		}
		ss := &apiresource.ScanningStation{
			ID:     *icl.ScanningStationId,
			Object: constants.ObjectTypeScanningStation,
			Name:   name,
		}
		if icl.ScanningStationType != nil {
			ss.Type = constants.ScanningStationType(*icl.ScanningStationType)
		}
		if icl.ScanningStationCreatedAt != nil {
			ss.CreatedAt = icl.ScanningStationCreatedAt.AsTime()
		}
		if icl.ScanningStationUpdatedAt != nil {
			ss.UpdatedAt = icl.ScanningStationUpdatedAt.AsTime()
		}
		result.ResponsibleScanningStation = ss
	}

	return result
}

func InventoryChangeLogListPresenter(resp *pb.ListInventoryChangeLogsResponse) *apiresource.List[apiresource.InventoryChangeLog] {
	if resp == nil {
		return apiresource.NewList[apiresource.InventoryChangeLog](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.InventoryChangeLog, len(resp.InventoryChangeLogs))
	for i, icl := range resp.InventoryChangeLogs {
		items[i] = InventoryChangeLogPresenter(icl)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func ExportInventoryChangeLogsPresenter(resp *pb.ExportInventoryChangeLogsResponse) *apiresource.ExportInventoryChangeLogsResponse {
	if resp == nil {
		return nil
	}

	items := make([]*apiresource.InventoryChangeLog, len(resp.InventoryChangeLogs))
	for i, icl := range resp.InventoryChangeLogs {
		presented := InventoryChangeLogPresenter(icl)
		items[i] = &presented
	}

	return &apiresource.ExportInventoryChangeLogsResponse{
		Object: constants.ObjectTypeList,
		Items:  items,
		Count:  resp.Count,
	}
}
