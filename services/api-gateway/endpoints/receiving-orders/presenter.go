package receivingorderep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ReceivingOrderSummaryPresenter(info *pb.ReceivingOrderSummaryInfo) apiresource.ReceivingOrderSummary {
	if info == nil {
		return apiresource.ReceivingOrderSummary{}
	}

	s := apiresource.ReceivingOrderSummary{
		ID:     info.Id,
		Object: constants.ObjectTypeReceivingOrder,
		Number: info.Number,
		PurchaseOrder: &apiresource.SalesOrder{
			ID:     info.PurchaseOrderId,
			Object: constants.ObjectTypeSalesOrder,
			Number: info.PurchaseOrderNumber,
		},
		LineCount:            info.LineCount,
		CompletionPercentage: info.CompletionPercentage,
		CompletedAt:          grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:            grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.SupplierId != nil {
		supplier := &apiresource.Account{
			ID:     *info.SupplierId,
			Object: constants.ObjectTypeAccount,
		}
		if info.SupplierName != nil {
			supplier.Name = *info.SupplierName
		}
		s.Supplier = supplier
	}

	return s
}

func ReceivingOrderPresenter(info *pb.ReceivingOrderInfo) apiresource.ReceivingOrder {
	if info == nil {
		return apiresource.ReceivingOrder{}
	}

	r := apiresource.ReceivingOrder{
		ID:     info.Id,
		Object: constants.ObjectTypeReceivingOrder,
		Number: info.Number,
		PurchaseOrder: &apiresource.SalesOrder{
			ID:     info.PurchaseOrderId,
			Object: constants.ObjectTypeSalesOrder,
			Number: info.PurchaseOrderNumber,
		},
		CompletedAt: grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.SupplierId != nil {
		supplier := &apiresource.Account{
			ID:     *info.SupplierId,
			Object: constants.ObjectTypeAccount,
		}
		if info.SupplierName != nil {
			supplier.Name = *info.SupplierName
		}
		r.Supplier = supplier
	}

	if info.Note != nil {
		note := *info.Note
		r.Note = &note
	}

	if len(info.Lines) > 0 {
		lines := make([]apiresource.ReceivingOrderLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = ReceivingOrderLinePresenter(l)
		}
		r.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	return r
}

func ReceivingOrderLinePresenter(info *pb.ReceivingOrderLineInfo) apiresource.ReceivingOrderLine {
	if info == nil {
		return apiresource.ReceivingOrderLine{}
	}

	line := apiresource.ReceivingOrderLine{
		ID:     info.Id,
		Object: constants.ObjectTypeReceivingOrderLine,
		Quantity: &apiresource.Quantity{
			ID:           info.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        info.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(info.QuantityValue, info.QuantityUnitAbbreviation, ""),
			Unit: &apiresource.Unit{
				ID:           info.QuantityUnitId,
				Object:       constants.ObjectTypeUnit,
				Abbreviation: info.QuantityUnitAbbreviation,
			},
		},
		OrderLine: &apiresource.SalesOrderLineDetail{
			ID:     info.OrderLineId,
			Object: constants.ObjectTypeSalesOrderLine,
			QuantityOrdered: &apiresource.Quantity{
				ID:           "",
				Object:       constants.ObjectTypeQuantity,
				Value:        info.OrderLineQuantityOrdered,
				DisplayValue: apiresource.FormatDisplayValue(info.OrderLineQuantityOrdered, info.OrderLineUnitAbbreviation, ""),
				Unit: &apiresource.Unit{
					ID:           info.OrderLineUnitId,
					Object:       constants.ObjectTypeUnit,
					Abbreviation: info.OrderLineUnitAbbreviation,
				},
			},
		},
		StockedAt: grpcutil.TimestampToTimePtr(info.StockedAt),
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.OrderLineItemId != nil {
		item := &apiresource.Item{
			ID:     *info.OrderLineItemId,
			Object: constants.ObjectTypeItem,
		}
		if info.OrderLineItemSku != nil {
			item.SKU = *info.OrderLineItemSku
		}
		line.OrderLine.Item = item
	}

	if info.OrderLineItemDescription != nil {
		line.OrderLine.ProductDescription = info.OrderLineItemDescription
	}

	if info.OrderLineItemSku != nil {
		line.OrderLine.ProductSKU = *info.OrderLineItemSku
	}

	if info.RejectedQuantityValue != nil {
		line.RejectedQuantity = &apiresource.Quantity{
			ID:           "",
			Object:       constants.ObjectTypeQuantity,
			Value:        *info.RejectedQuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(*info.RejectedQuantityValue, info.QuantityUnitAbbreviation, ""),
			Unit: &apiresource.Unit{
				ID:           info.QuantityUnitId,
				Object:       constants.ObjectTypeUnit,
				Abbreviation: info.QuantityUnitAbbreviation,
			},
		}
	}

	return line
}

func ReceivingOrderListPresenter(resp *pb.ListReceivingOrdersResponse) *apiresource.List[apiresource.ReceivingOrderSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.ReceivingOrderSummary](nil, apiresource.PageInfo{})
	}

	orders := make([]apiresource.ReceivingOrderSummary, len(resp.ReceivingOrders))
	for i, o := range resp.ReceivingOrders {
		orders[i] = ReceivingOrderSummaryPresenter(o)
	}

	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
