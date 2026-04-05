package pickep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func PickSummaryPresenter(info *pb.PickSummaryInfo) apiresource.PickSummary {
	s := apiresource.PickSummary{
		ID:     info.Id,
		Object: constants.ObjectTypePick,
		Number: info.Number,
		Customer: &apiresource.Customer{
			ID:     info.CustomerId,
			Object: constants.ObjectTypeCustomer,
			Name:   info.CustomerName,
			Number: info.CustomerNumber,
		},
		Priority: &apiresource.Priority{
			Code:   constants.PriorityCode(info.PriorityCode),
			Object: constants.ObjectTypePriority,
			Name:   info.PriorityName,
		},
		SalesOrder: &apiresource.PickSalesOrder{
			ID:     info.SalesOrderId,
			Object: constants.ObjectTypeSalesOrder,
		},
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	s.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)
	return s
}

func PickDetailPresenter(info *pb.PickInfo) apiresource.PickDetail {
	d := apiresource.PickDetail{
		ID:     info.Id,
		Object: constants.ObjectTypePick,
		Number: info.Number,
		Customer: &apiresource.Customer{
			ID:     info.CustomerId,
			Object: constants.ObjectTypeCustomer,
			Name:   info.CustomerName,
			Number: info.CustomerNumber,
		},
		Priority: &apiresource.Priority{
			Code:   constants.PriorityCode(info.PriorityCode),
			Object: constants.ObjectTypePriority,
			Name:   info.PriorityName,
		},
		SalesOrder: &apiresource.PickSalesOrder{
			ID:     info.SalesOrderId,
			Object: constants.ObjectTypeSalesOrder,
		},
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	d.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)

	if len(info.Lines) > 0 {
		lines := make([]apiresource.PickLineDetail, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = PickLineDetailPresenter(l)
		}
		d.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	if len(info.Departments) > 0 {
		depts := make([]apiresource.PickDepartment, len(info.Departments))
		for i, dep := range info.Departments {
			depts[i] = apiresource.PickDepartment{
				ID:     dep.Id,
				Object: constants.ObjectTypeDepartment,
				Name:   dep.Name,
			}
		}
		d.Departments = depts
	}

	return d
}

func PickLineDetailPresenter(info *pb.PickLineInfo) apiresource.PickLineDetail {
	d := apiresource.PickLineDetail{
		ID:     info.Id,
		Object: constants.ObjectTypePickLine,
		Quantity: &apiresource.Quantity{
			ID:     info.QuantityId,
			Object: constants.ObjectTypeQuantity,
			Value:  info.QuantityValue,
			Unit: &apiresource.Unit{
				ID:           info.QuantityUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         info.QuantityUnitName,
				Abbreviation: info.QuantityUnitAbbreviation,
			},
		},
		OrderedQuantity: &apiresource.Quantity{
			ID:     "",
			Object: constants.ObjectTypeQuantity,
			Value:  info.OrderedQuantityValue,
			Unit: &apiresource.Unit{
				ID:           info.OrderedQuantityUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         info.OrderedQuantityUnitName,
				Abbreviation: info.OrderedQuantityUnitAbbreviation,
			},
		},
		SalesOrderLine: &apiresource.PickSalesOrderLine{
			ID:                 info.SalesOrderLineId,
			Object:             constants.ObjectTypeSalesOrderLine,
			LineItemNumber:     info.OrderLineItemNumber,
			ProductSKU:         info.OrderLineSku,
			ProductDescription: info.OrderLineDescription,
		},
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	d.PackedAt = grpcutil.TimestampToTimePtr(info.PackedAt)
	return d
}

func PickListPresenter(resp *pb.ListPicksResponse) *apiresource.List[apiresource.PickSummary] {
	picks := make([]apiresource.PickSummary, len(resp.Picks))
	for i, p := range resp.Picks {
		picks[i] = PickSummaryPresenter(p)
	}
	return apiresource.NewList(picks, apiresource.PageInfo{
		NextCursor:  resp.PageInfo.NextCursor,
		PrevCursor:  resp.PageInfo.PrevCursor,
		HasNextPage: resp.PageInfo.HasNextPage,
		HasPrevPage: resp.PageInfo.HasPrevPage,
	})
}
