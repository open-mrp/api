package pickep

import (
	"context"
	"time"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func finalizePickCustomer(c *apiresource.Customer, fallbackCreated, fallbackUpdated time.Time) {
	if c == nil {
		return
	}
	if c.Status == "" {
		c.Status = constants.AccountStatusCodeNormal
	}
	if c.CommissionPolicy == "" {
		c.CommissionPolicy = constants.CommissionPolicyApplied
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = fallbackCreated
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = fallbackUpdated
	}
}

func stubCustomerForPickList(info *pb.PickSummaryInfo) *apiresource.Customer {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	c := &apiresource.Customer{
		ID:               info.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             info.CustomerName,
		Number:           info.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
	}
	finalizePickCustomer(c, now, now)
	return c
}

func stubCustomerForPickDetail(info *pb.PickInfo) *apiresource.Customer {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	c := &apiresource.Customer{
		ID:               info.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             info.CustomerName,
		Number:           info.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
	}
	finalizePickCustomer(c, now, now)
	return c
}

func stubSalesOrderForPickSummary(info *pb.PickSummaryInfo) *apiresource.SalesOrderDetail {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	return &apiresource.SalesOrderDetail{
		ID:     info.SalesOrderId,
		Object: constants.ObjectTypeSalesOrder,
		Number: info.SalesOrderNumber,
		Status: &apiresource.SalesOrderStatusDetail{
			Code:   string(constants.SalesOrderStatusCodeIssued),
			Object: constants.ObjectTypeSalesOrderStatus,
			Name:   "Issued",
		},
		Type: &apiresource.SalesOrderType{
			Code:   "standard",
			Object: constants.ObjectTypeSalesOrderType,
			Name:   "Standard",
		},
		Priority: &apiresource.Priority{
			Code:   constants.PriorityCode(info.PriorityCode),
			Object: constants.ObjectTypePriority,
			Name:   info.PriorityName,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func stubSalesOrderForPickDetail(info *pb.PickInfo) *apiresource.SalesOrderDetail {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	return &apiresource.SalesOrderDetail{
		ID:     info.SalesOrderId,
		Object: constants.ObjectTypeSalesOrder,
		Number: info.SalesOrderNumber,
		Status: &apiresource.SalesOrderStatusDetail{
			Code:   string(constants.SalesOrderStatusCodeIssued),
			Object: constants.ObjectTypeSalesOrderStatus,
			Name:   "Issued",
		},
		Type: &apiresource.SalesOrderType{
			Code:   "standard",
			Object: constants.ObjectTypeSalesOrderType,
			Name:   "Standard",
		},
		Priority: &apiresource.Priority{
			Code:   constants.PriorityCode(info.PriorityCode),
			Object: constants.ObjectTypePriority,
			Name:   info.PriorityName,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func stubUnitForPickLine(id, name, abbr string, ts time.Time) *apiresource.Unit {
	if id == "" {
		id = "un_unknown"
	}
	if name == "" {
		name = abbr
	}
	if abbr == "" {
		abbr = "—"
	}
	return &apiresource.Unit{
		ID:                id,
		Object:            constants.ObjectTypeUnit,
		Name:              name,
		Abbreviation:      abbr,
		Type:              constants.UnitTypeQuantity,
		RatioNumerator:    "1",
		RatioDenominator:  "1",
		OffsetNumerator:   "0",
		OffsetDenominator: "1",
		CreatedAt:         ts,
		UpdatedAt:         ts,
	}
}

func stubSalesOrderLineForPick(info *pb.PickLineInfo) *apiresource.SalesOrderLineDetail {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	sku := info.OrderLineSku
	if sku == "" {
		sku = "—"
	}
	var productDesc *string
	if info.OrderLineDescription != nil && *info.OrderLineDescription != "" {
		productDesc = info.OrderLineDescription
	}

	unitPriceValue := info.UnitPriceValue
	if unitPriceValue == "" {
		unitPriceValue = "0"
	}
	unitPriceID := info.UnitPriceId
	if unitPriceID == "" {
		unitPriceID = "ra_stub"
	}

	return &apiresource.SalesOrderLineDetail{
		ID:                 info.SalesOrderLineId,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     info.OrderLineItemNumber,
		ProductSKU:         sku,
		ProductDescription: productDesc,
		QuantityOrdered: &apiresource.Quantity{
			ID:     info.OrderedQuantityId,
			Object: constants.ObjectTypeQuantity,
			Value:  info.OrderedQuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(
				info.OrderedQuantityValue,
				info.OrderedQuantityUnitAbbreviation,
				string(constants.UnitTypeQuantity),
			),
			Unit: stubUnitForPickLine(
				info.OrderedQuantityUnitId,
				info.OrderedQuantityUnitName,
				info.OrderedQuantityUnitAbbreviation,
				now,
			),
		},
		UnitPrice: &apiresource.Rate{
			ID:     unitPriceID,
			Object: constants.ObjectTypeRate,
			Value:  unitPriceValue,
			NumeratorUnit: stubUnitForPickLine(
				info.UnitPriceNumeratorUnitId,
				info.UnitPriceNumeratorUnitAbbreviation,
				info.UnitPriceNumeratorUnitAbbreviation,
				now,
			),
			DenominatorUnit: stubUnitForPickLine(
				info.UnitPriceDenominatorUnitId,
				info.UnitPriceDenominatorUnitAbbreviation,
				info.UnitPriceDenominatorUnitAbbreviation,
				now,
			),
			DisplayValue: apiresource.FormatRateDisplayValue(
				unitPriceValue,
				info.UnitPriceNumeratorUnitAbbreviation,
				"",
				info.UnitPriceDenominatorUnitAbbreviation,
			),
			CreatedAt: now,
			UpdatedAt: now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func PickSummaryPresenter(info *pb.PickSummaryInfo) apiresource.PickSummary {
	s := apiresource.PickSummary{
		ID:       info.Id,
		Object:   constants.ObjectTypePick,
		Number:   info.Number,
		Customer: stubCustomerForPickList(info),
		Priority: &apiresource.Priority{
			Code:   constants.PriorityCode(info.PriorityCode),
			Object: constants.ObjectTypePriority,
			Name:   info.PriorityName,
		},
		SalesOrder: stubSalesOrderForPickSummary(info),
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}
	s.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)
	return s
}

func PickDetailPresenter(info *pb.PickInfo) apiresource.PickDetail {
	d := apiresource.PickDetail{
		ID:       info.Id,
		Object:   constants.ObjectTypePick,
		Number:   info.Number,
		Customer: stubCustomerForPickDetail(info),
		Priority: &apiresource.Priority{
			Code:   constants.PriorityCode(info.PriorityCode),
			Object: constants.ObjectTypePriority,
			Name:   info.PriorityName,
		},
		SalesOrder: stubSalesOrderForPickDetail(info),
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}
	d.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)

	if len(info.Lines) > 0 {
		lines := make([]apiresource.PickLineDetail, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = PickLineDetailPresenter(l)
		}
		d.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	depts := make([]apiresource.Department, len(info.Departments))
	deptTS := grpcutil.TimestampToTime(info.CreatedAt)
	for i, dep := range info.Departments {
		name := dep.Name
		if name == "" {
			name = "Department"
		}
		depts[i] = apiresource.Department{
			ID:        dep.Id,
			Object:    constants.ObjectTypeDepartment,
			Name:      name,
			CreatedAt: deptTS,
			UpdatedAt: deptTS,
		}
	}
	d.Departments = apiresource.NewList(depts, apiresource.PageInfo{})

	return d
}

func PickLineDetailPresenter(info *pb.PickLineInfo) apiresource.PickLineDetail {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	d := apiresource.PickLineDetail{
		ID:     info.Id,
		Object: constants.ObjectTypePickLine,
		Quantity: &apiresource.Quantity{
			ID:     info.QuantityId,
			Object: constants.ObjectTypeQuantity,
			Value:  info.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(
				info.QuantityValue,
				info.QuantityUnitAbbreviation,
				string(constants.UnitTypeQuantity),
			),
			Unit: stubUnitForPickLine(
				info.QuantityUnitId,
				info.QuantityUnitName,
				info.QuantityUnitAbbreviation,
				now,
			),
		},
		OrderedQuantity: &apiresource.Quantity{
			ID:     info.OrderedQuantityId,
			Object: constants.ObjectTypeQuantity,
			Value:  info.OrderedQuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(
				info.OrderedQuantityValue,
				info.OrderedQuantityUnitAbbreviation,
				string(constants.UnitTypeQuantity),
			),
			Unit: stubUnitForPickLine(
				info.OrderedQuantityUnitId,
				info.OrderedQuantityUnitName,
				info.OrderedQuantityUnitAbbreviation,
				now,
			),
		},
		SalesOrderLine: stubSalesOrderLineForPick(info),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	d.PackedAt = grpcutil.TimestampToTimePtr(info.PackedAt)
	return d
}

func PickListPresenter(ctx context.Context, resp *pb.ListPicksResponse) *apiresource.List[apiresource.PickSummary] {
	picks := make([]apiresource.PickSummary, len(resp.Picks))
	for i, p := range resp.Picks {
		picks[i] = PickSummaryPresenter(p)
	}
	return apiresource.NewList(picks, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
