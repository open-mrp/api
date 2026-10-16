package grpc

import (
	"context"
	"fmt"

	"github.com/open-mrp/apikit/field"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func (h *gRPCHandler) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	money := newMoneyRows(ctx)
	params := domain.CreateItemParams{
		Type:         constants.ItemTypeCode(req.Type),
		SKU:          req.Sku,
		Description:  req.Description,
		CategoryID:   req.CategoryId,
		AttributeIDs: req.AttributeIds,
		UnitCost:     money.rate("unit_cost", req.UnitCost),
		Sales:        itemSalesInputFromProto(money, req.Sales),
		Material:     itemMaterialInputFromProto(req.Material),
	}
	if req.Status != nil {
		params.Status = constants.ArchiveStatus(*req.Status)
	}
	if req.Supply != nil {
		supply := constants.Supply(*req.Supply)
		params.Supply = &supply
	}
	if apiErr := money.err(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	params.LegacyUnits = req.LegacyUnits
	params.Units.StockingUnitID = req.StockingUnitId
	params.Units.DefaultOrderUnitID = req.DefaultOrderUnitId
	if len(req.OrderUnits) > 0 {
		orderUnits, apiErr := orderUnitsFromProto("", req.OrderUnits)
		if apiErr != nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
		}
		params.Units.OrderUnits = &orderUnits
	}

	item, apiErr := h.itemSvc.CreateItem(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.CreateItemResponse{Item: itemToProto(item)}, nil
}

func (h *gRPCHandler) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.UpdateItemResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	money := newMoneyRows(ctx)
	params := domain.UpdateItemParams{
		ItemID:            req.Id,
		SKU:               req.Sku,
		Description:       req.Description,
		UpdateDescription: req.UpdateDescription,
		CategoryID:        req.CategoryId,
		Supply:            fieldpb.EnumClearableFromProto[constants.Supply](req.Supply),
		UnitCost:          money.rate("unit_cost", req.UnitCost),
		Sales:             itemSalesInputFromProto(money, req.Sales),
		RemoveSales:       req.ClearSales,
		Material:          itemMaterialInputFromProto(req.Material),
	}
	if req.AttributeIds != nil {
		params.AttributeIDs = field.Some(append([]string{}, req.AttributeIds.Value...))
	}
	if req.Status != nil {
		status := constants.ArchiveStatus(*req.Status)
		params.Status = &status
	}
	if apiErr := money.err(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	params.Units.DefaultOrderUnitID = req.DefaultOrderUnitId
	if req.OrderUnits != nil {
		orderUnits, apiErr := orderUnitsFromProto("", req.OrderUnits.Values)
		if apiErr != nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
		}
		params.Units.OrderUnits = &orderUnits
	}

	item, apiErr := h.itemSvc.UpdateItem(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.UpdateItemResponse{Item: itemToProto(item)}, nil
}

func (h *gRPCHandler) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if _, apiErr := h.itemSvc.DeleteItem(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) BulkUpsertItems(ctx context.Context, req *pb.BulkUpsertItemsRequest) (*pb.BulkUpsertItemsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	money := newMoneyRows(ctx)
	rows := make([]domain.UpsertItemParams, len(req.Items))
	for i, in := range req.Items {
		attrs := make([]domain.UpsertItemPropertyParams, len(in.Attributes))
		for j, a := range in.Attributes {
			attrs[j] = domain.UpsertItemPropertyParams{Name: a.Name, Value: a.Value}
		}
		row := domain.UpsertItemParams{
			Type:          constants.ItemTypeCode(in.Type),
			SKU:           in.Sku,
			Description:   in.Description,
			Category:      objectIdentifierFromProto(in.Category),
			Attributes:    attrs,
			HasAttributes: in.HasAttributes,
			UnitCost:      money.rate("unit_cost", in.UnitCost),
		}
		if in.Status != nil {
			status := constants.ArchiveStatus(*in.Status)
			row.Status = &status
		}
		if in.Supply != nil {
			if in.Supply.Clear || in.Supply.Value == nil {
				row.ClearSupply = true
			} else {
				supply := constants.Supply(*in.Supply.Value)
				row.Supply = &supply
			}
		}
		if s := in.Sales; s != nil {
			sales := &domain.UpsertItemSalesRow{UnitPrice: money.rate("sales.unit_price", s.UnitPrice)}
			sales.ProductLineID, sales.ClearProductLine = stringPatchParts(s.ProductLineId)
			if s.PortalVisibility != nil {
				v := constants.CustomerPortalVisibility(*s.PortalVisibility)
				sales.PortalVisibility = &v
			}
			sales.LeadTime, sales.ClearLeadTime = quantityPatchParts(s.LeadTime)
			row.Sales = sales
		}
		row.Units.StockingUnitID = in.StockingUnitId
		row.Units.DefaultOrderUnitID = in.DefaultOrderUnitId
		if in.OrderUnits != nil {
			orderUnits, apiErr := orderUnitsFromProto(fmt.Sprintf("items[%d].", i), in.OrderUnits.Values)
			if apiErr != nil {
				return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
			}
			row.Units.OrderUnits = &orderUnits
		}
		if m := in.Material; m != nil {
			material := &domain.UpsertItemMaterialRow{}
			material.OrderPoint, material.ClearOrderPoint = quantityPatchParts(m.OrderPoint)
			material.LeadTime, material.ClearLeadTime = quantityPatchParts(m.LeadTime)
			row.Material = material
		}
		rows[i] = row
	}
	if apiErr := money.err(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	job, apiErr := h.itemSvc.BulkUpsertItems(ctx, domain.BulkUpsertItemsParams{Items: rows})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.BulkUpsertItemsResponse{Job: jobToProto(job)}, nil
}

// itemSalesInputFromProto converts a sales write, recording a price conversion failure on money.
func itemSalesInputFromProto(money *moneyRows, s *pb.ItemSalesInput) *domain.ItemSalesInput {
	if s == nil {
		return nil
	}
	in := &domain.ItemSalesInput{
		UnitPrice:     money.rate("sales.unit_price", s.UnitPrice),
		ProductLineID: fieldpb.StringClearableFromProto(s.ProductLineId),
		LeadTime:      quantityClearableFromProto(s.LeadTime),
	}
	if s.PortalVisibility != nil {
		v := constants.CustomerPortalVisibility(*s.PortalVisibility)
		in.PortalVisibility = &v
	}
	return in
}

func itemMaterialInputFromProto(m *pb.ItemMaterialInput) *domain.ItemMaterialInput {
	if m == nil {
		return nil
	}
	return &domain.ItemMaterialInput{
		OrderPoint: quantityClearableFromProto(m.OrderPoint),
		LeadTime:   quantityClearableFromProto(m.LeadTime),
	}
}

func quantityClearableFromProto(p *pb.QuantityPatch) field.Clearable[domain.QuantityInput] {
	f := fieldpb.QuantityClearableFromProto(p)
	if v, ok := f.Value(); ok {
		return field.Set(domain.QuantityInput{Value: v.Value, UnitID: v.UnitID})
	}
	if f.IsClear() {
		return field.Clear[domain.QuantityInput]()
	}
	return field.Unset[domain.QuantityInput]()
}

func quantityPatchParts(p *pb.QuantityPatch) (*domain.QuantityInput, bool) {
	f := quantityClearableFromProto(p)
	if v, ok := f.Value(); ok {
		return &v, false
	}
	return nil, f.IsClear()
}

func stringPatchParts(p *pb.StringPatch) (*string, bool) {
	f := fieldpb.StringClearableFromProto(p)
	if v, ok := f.Value(); ok {
		return &v, false
	}
	return nil, f.IsClear()
}

// portalReadyFilter narrows a list to visible or hidden sold items, or is nil when both or neither are asked for.
func portalReadyFilter(visibilities []string) *bool {
	visible, hidden := false, false
	for _, v := range visibilities {
		switch constants.CustomerPortalVisibility(v) {
		case constants.CustomerPortalVisibilityVisible:
			visible = true
		case constants.CustomerPortalVisibilityHidden:
			hidden = true
		}
	}
	if visible == hidden {
		return nil
	}
	return &visible
}

// itemSectionsToProto presents an item's status, supply and sections.
func itemSectionsToProto(i *domain.Item, out *pb.ItemInfo) {
	out.Status = string(i.Status)
	out.Supply = i.SupplyCode
	if s := i.Sales; s != nil {
		sales := &pb.ItemSalesInfo{
			UnitPrice:        domain.UnitPricePtrToProto(i.SalesUnitPrice),
			PortalVisibility: string(constants.CustomerPortalVisibilityHidden),
			ProductLineId:    s.ProductLineID,
			LeadTime:         quantityValueToProto(s.LeadTime),
		}
		if s.IsPortalReady {
			sales.PortalVisibility = string(constants.CustomerPortalVisibilityVisible)
		}
		out.Sales = sales
	}
	if m := i.MaterialSection; m != nil {
		out.Material = &pb.ItemMaterialInfo{
			OrderPoint: quantityValueToProto(m.OrderPoint),
			LeadTime:   quantityValueToProto(m.LeadTime),
		}
	}
}

func quantityValueToProto(q *domain.QuantityValue) *pb.QuantityValue {
	if q == nil {
		return nil
	}
	return &pb.QuantityValue{Value: q.Value, UnitId: q.UnitID, UnitType: q.UnitType}
}
