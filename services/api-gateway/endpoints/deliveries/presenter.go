package deliveryep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func DeliverySummaryPresenter(d *pb.DeliverySummaryInfo) apiresource.DeliverySummary {
	if d == nil {
		return apiresource.DeliverySummary{}
	}

	return apiresource.DeliverySummary{
		ID:     d.Id,
		Object: constants.ObjectTypeDelivery,
		Number: d.Number,
		PurchaseOrder: &apiresource.SalesOrder{
			ID:     d.PurchaseOrderId,
			Object: constants.ObjectTypeSalesOrder,
			Number: d.PurchaseOrderNumber,
		},
		Status:     constants.DeliveryStatus(d.Status),
		LineCount:  d.LineCount,
		AcceptedAt: grpcutil.TimestampToTimePtr(d.AcceptedAt),
		RejectedAt: grpcutil.TimestampToTimePtr(d.RejectedAt),
		CreatedAt:  grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func DeliveryPresenter(d *pb.DeliveryInfo) apiresource.Delivery {
	if d == nil {
		return apiresource.Delivery{}
	}

	lines := make([]apiresource.DeliveryLine, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = DeliveryLinePresenter(l)
	}

	return apiresource.Delivery{
		ID:     d.Id,
		Object: constants.ObjectTypeDelivery,
		Number: d.Number,
		PurchaseOrder: &apiresource.SalesOrder{
			ID:     d.PurchaseOrderId,
			Object: constants.ObjectTypeSalesOrder,
			Number: d.PurchaseOrderNumber,
		},
		Status:     constants.DeliveryStatus(d.Status),
		Lines:      apiresource.NewList(lines, apiresource.PageInfo{}),
		AcceptedAt: grpcutil.TimestampToTimePtr(d.AcceptedAt),
		RejectedAt: grpcutil.TimestampToTimePtr(d.RejectedAt),
		CreatedAt:  grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func DeliveryLinePresenter(l *pb.DeliveryLineInfo) apiresource.DeliveryLine {
	if l == nil {
		return apiresource.DeliveryLine{}
	}

	line := apiresource.DeliveryLine{
		ID:     l.Id,
		Object: constants.ObjectTypeDeliveryLine,
		Quantity: &apiresource.Quantity{
			ID:           l.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        l.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(l.QuantityValue, l.QuantityUnitAbbreviation, ""),
			Unit: &apiresource.Unit{
				ID:     l.QuantityUnitId,
				Object: constants.ObjectTypeUnit,
			},
		},
		UnitCost: &apiresource.Rate{
			ID:     l.UnitCostId,
			Object: constants.ObjectTypeRate,
			Value:  l.UnitCostValue,
			NumeratorUnit: &apiresource.Unit{
				ID:     l.UnitCostNumeratorUnitId,
				Object: constants.ObjectTypeUnit,
			},
			DenominatorUnit: &apiresource.Unit{
				ID:     l.UnitCostDenominatorUnitId,
				Object: constants.ObjectTypeUnit,
			},
			DisplayValue: "",
		},
		CreatedAt: grpcutil.TimestampToTime(l.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(l.UpdatedAt),
	}

	if l.ItemId != nil {
		item := &apiresource.Item{
			ID:     *l.ItemId,
			Object: constants.ObjectTypeItem,
		}
		if l.ItemSku != nil {
			item.SKU = *l.ItemSku
		}
		line.Item = item
	}

	if l.LocationId != nil {
		loc := &apiresource.Location{
			ID:     *l.LocationId,
			Object: constants.ObjectTypeLocation,
		}
		if l.LocationName != nil {
			loc.Name = *l.LocationName
		}
		line.Location = loc
	}

	if l.LotId != nil {
		lot := &apiresource.Lot{
			ID:     *l.LotId,
			Object: constants.ObjectTypeLot,
		}
		if l.LotNumber != nil {
			lot.LotNumber = *l.LotNumber
		}
		line.Lot = lot
	}

	line.AcceptedAt = grpcutil.TimestampToTimePtr(l.AcceptedAt)
	line.RejectedAt = grpcutil.TimestampToTimePtr(l.RejectedAt)

	return line
}

func DeliveryListPresenter(resp *pb.ListDeliveriesResponse) *apiresource.List[apiresource.DeliverySummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.DeliverySummary](nil, apiresource.PageInfo{})
	}

	deliveries := make([]apiresource.DeliverySummary, len(resp.Deliveries))
	for i, d := range resp.Deliveries {
		deliveries[i] = DeliverySummaryPresenter(d)
	}

	return apiresource.NewList(deliveries, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
