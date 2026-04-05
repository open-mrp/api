package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func rateToProto(r *domain.Rate) *pb.RateInfo {
	if r == nil {
		return nil
	}
	return &pb.RateInfo{
		Id:                r.ID,
		Value:             r.Value,
		NumeratorUnitId:   r.NumeratorUnitID,
		DenominatorUnitId: r.DenominatorUnitID,
		CreatedAt:         timestamppb.New(r.CreatedAt),
		UpdatedAt:         timestamppb.New(r.UpdatedAt),
	}
}

func itemCategoryToProto(c *domain.ItemCategory) *pb.ItemCategoryInfo {
	if c == nil {
		return nil
	}
	return &pb.ItemCategoryInfo{
		Id:                   c.ID,
		Name:                 c.Name,
		ItemCategoryTypeCode: c.ItemCategoryTypeCode,
		UnitGroupId:          c.UnitGroupID,
	}
}

func itemAttributeToProto(a *domain.ItemAttribute) *pb.ItemAttributeInfo {
	if a == nil {
		return nil
	}
	return &pb.ItemAttributeInfo{
		Id:         a.ID,
		Value:      a.Value,
		ColorCode:  a.ColorCode,
		SortOrder:  a.Order,
		PropertyId: a.PropertyID,
	}
}

func itemToProto(i *domain.Item) *pb.ItemInfo {
	if i == nil {
		return nil
	}

	pbItem := &pb.ItemInfo{
		Id:           i.ID,
		Sku:          i.SKU,
		Description:  i.Description,
		Notes:        i.Notes,
		ItemTypeCode: i.ItemTypeCode,
		Category:     itemCategoryToProto(i.Category),
		UnitValue:    rateToProto(i.UnitValue),
		UnitCost:     rateToProto(i.UnitCost),
		BurnRate:     rateToProto(i.BurnRate),
		AccountId:    i.AccountID,
		IsDirty:      i.IsDirty,
		CreatedAt:    timestamppb.New(i.CreatedAt),
		UpdatedAt:    timestamppb.New(i.UpdatedAt),
	}

	if i.Attributes != nil {
		pbItem.Attributes = make([]*pb.ItemAttributeInfo, len(i.Attributes))
		for j, a := range i.Attributes {
			pbItem.Attributes[j] = itemAttributeToProto(a)
		}
	}

	return pbItem
}

func (h *gRPCHandler) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	var startDate, endDate *timestamppb.Timestamp
	if req.StartDate != nil {
		startDate = req.StartDate
	}
	if req.EndDate != nil {
		endDate = req.EndDate
	}

	params := domain.ListItemsParams{
		Cursor:                   req.Cursor,
		Limit:                    req.Limit,
		Query:                    req.Query,
		Types:                    req.Types,
		CategoryIDs:              req.CategoryIds,
		AttributeIDs:             req.AttributeIds,
		SupplierID:               req.SupplierId,
		IsExactMatch:             req.IsExactMatch,
		OnlyInitialSubassemblies: req.OnlyInitialSubassemblies,
	}

	if startDate != nil {
		t := startDate.AsTime()
		params.StartDate = &t
	}
	if endDate != nil {
		t := endDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.itemSvc.ListItems(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.ItemInfo, len(result.Items))
	for i, item := range result.Items {
		pbItems[i] = itemToProto(item)
	}

	return &pb.ListItemsResponse{
		Items: pbItems,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.GetItemResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	item, apiErr := h.itemSvc.GetItem(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetItemResponse{
		Item: itemToProto(item),
	}, nil
}

func (h *gRPCHandler) GetItemInventory(ctx context.Context, req *pb.GetItemInventoryRequest) (*pb.GetItemInventoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	inv, apiErr := h.itemSvc.GetItemInventory(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetItemInventoryResponse{
		OnHand:             &pb.QuantityInfo{Value: inv.OnHand, UnitId: inv.OnHandUnitID},
		Reserved:           &pb.QuantityInfo{Value: inv.Reserved, UnitId: inv.ReservedUnitID},
		AvailableToPromise: &pb.QuantityInfo{Value: inv.AvailableToPromise, UnitId: inv.ATPUnitID},
		Short:              &pb.QuantityInfo{Value: inv.Short, UnitId: inv.ShortUnitID},
	}, nil
}

func (h *gRPCHandler) GetItemCosts(ctx context.Context, req *pb.GetItemCostsRequest) (*pb.GetItemCostsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	costs, apiErr := h.itemSvc.GetItemCosts(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetItemCostsResponse{
		DirectMaterialCost: costs.DirectMaterialCost,
		DirectLaborCost:    costs.DirectLaborCost,
		OverheadCost:       costs.OverheadCost,
		TotalCost:          costs.TotalCost,
		UnitId:             costs.UnitID,
	}, nil
}

func (h *gRPCHandler) GetItemTrends(ctx context.Context, req *pb.GetItemTrendsRequest) (*pb.GetItemTrendsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	trends, apiErr := h.itemSvc.GetItemTrends(ctx, req.Id, req.TrendType)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	points := make([]*pb.ItemTrendPoint, len(trends.Points))
	for i, p := range trends.Points {
		points[i] = &pb.ItemTrendPoint{
			Date:  timestamppb.New(p.Date),
			Value: p.Value,
		}
	}

	return &pb.GetItemTrendsResponse{
		TrendType: trends.TrendType,
		Points:    points,
	}, nil
}

func (h *gRPCHandler) ExportItems(ctx context.Context, req *pb.ExportItemsRequest) (*pb.ExportItemsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.itemSvc.ExportItems(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.ExportItemInfo, len(result.Items))
	for i, item := range result.Items {
		pbItems[i] = &pb.ExportItemInfo{
			Id:             item.ID,
			Sku:            item.SKU,
			Description:    item.Description,
			Notes:          item.Notes,
			ItemTypeCode:   item.ItemTypeCode,
			CategoryName:   item.CategoryName,
			AccountId:      item.AccountID,
			CreatedAt:      timestamppb.New(item.CreatedAt),
			UpdatedAt:      timestamppb.New(item.UpdatedAt),
			OnHandQuantity: item.OnHandQuantity,
			OnHandUnitId:   item.OnHandUnitID,
		}
	}

	return &pb.ExportItemsResponse{
		Items: pbItems,
		Count: result.Count,
	}, nil
}

func (h *gRPCHandler) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.UpdateItemResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateItemParams{
		ItemID:            req.Id,
		SKU:               req.Sku,
		Description:       req.Description,
		UpdateDescription: req.UpdateDescription,
		Notes:             req.Notes,
		UpdateNotes:       req.UpdateNotes,
	}

	item, apiErr := h.itemSvc.UpdateItem(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateItemResponse{
		Item: itemToProto(item),
	}, nil
}

func (h *gRPCHandler) AddItemAttribute(ctx context.Context, req *pb.AddItemAttributeRequest) (*pb.AddItemAttributeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	item, apiErr := h.itemSvc.AddItemAttribute(ctx, req.ItemId, req.AttributeId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.AddItemAttributeResponse{
		Item: itemToProto(item),
	}, nil
}

func (h *gRPCHandler) RemoveItemAttribute(ctx context.Context, req *pb.RemoveItemAttributeRequest) (*pb.RemoveItemAttributeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	item, apiErr := h.itemSvc.RemoveItemAttribute(ctx, req.ItemId, req.AttributeId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RemoveItemAttributeResponse{
		Item: itemToProto(item),
	}, nil
}

func (h *gRPCHandler) ChangeItemCategory(ctx context.Context, req *pb.ChangeItemCategoryRequest) (*pb.ChangeItemCategoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	item, apiErr := h.itemSvc.ChangeItemCategory(ctx, req.ItemId, req.CategoryId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ChangeItemCategoryResponse{
		Item: itemToProto(item),
	}, nil
}

func (h *gRPCHandler) ListInventories(ctx context.Context, req *pb.ListInventoriesRequest) (*pb.ListInventoriesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListInventoriesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.itemSvc.ListInventories(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.InventoryItemProto, len(result.Items))
	for i, item := range result.Items {
		pbItems[i] = &pb.InventoryItemProto{
			Item:                   itemToProto(item.Item),
			OnHandQuantity:         item.OnHandQuantity,
			OnHandUnitId:           item.OnHandUnitID,
			OnHandUnitAbbreviation: item.OnHandUnitAbbrev,
			OnHandUnitType:         item.OnHandUnitType,
		}
	}

	return &pb.ListInventoriesResponse{
		Items: pbItems,
		Count: result.Count,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) BulkCreateItems(ctx context.Context, req *pb.BulkCreateItemsRequest) (*pb.BulkCreateItemsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	items := make([]domain.BulkCreateItemInput, len(req.Items))
	for i, item := range req.Items {
		items[i] = domain.BulkCreateItemInput{
			SKU:            item.Sku,
			Description:    item.Description,
			ItemCategoryID: item.ItemCategoryId,
			ProductLineID:  item.ProductLineId,
		}
	}

	results, apiErr := h.itemSvc.BulkCreateItems(ctx, domain.BulkCreateItemsParams{
		Items: items,
		Type:  req.Type,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbResults := make([]*pb.BulkCreateItemResult, len(results))
	for i, r := range results {
		pbResults[i] = &pb.BulkCreateItemResult{
			Sku:     r.SKU,
			Success: r.Success,
			Error:   r.Error,
			ItemId:  r.ItemID,
		}
	}

	return &pb.BulkCreateItemsResponse{Results: pbResults}, nil
}

func (h *gRPCHandler) UpdateItemInventory(ctx context.Context, req *pb.UpdateItemInventoryRequest) (*pb.UpdateItemInventoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateItemInventoryParams{
		ItemID:         req.ItemId,
		QuantityChange: req.QuantityChange,
		Reconcile:      req.Reconcile,
		CustomerID:     req.CustomerId,
		LocationID:     req.LocationId,
		LotNumber:      req.LotNumber,
		UnitID:         req.UnitId,
	}

	apiErr := h.itemSvc.UpdateItemInventory(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateItemInventoryResponse{}, nil
}

func (h *gRPCHandler) BulkReconcileItems(ctx context.Context, req *pb.BulkReconcileItemsRequest) (*pb.BulkReconcileItemsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	data := make([]domain.BulkReconcileItemInput, len(req.Data))
	for i, d := range req.Data {
		data[i] = domain.BulkReconcileItemInput{
			SKU:      d.Sku,
			Unit:     d.Unit,
			Quantity: d.Quantity,
		}
	}

	result, apiErr := h.itemSvc.BulkReconcileItems(ctx, domain.BulkReconcileItemsParams{
		Data:          data,
		ReconcileType: req.ReconcileType,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.BulkReconcileItemsResponse{}
	for _, r := range result.ReconciledItems {
		resp.ReconciledItems = append(resp.ReconciledItems, &pb.ReconciledItemProto{
			ItemId: r.ItemID, Sku: r.SKU,
			PreviousQuantity: r.PreviousQuantity, NewQuantity: r.NewQuantity,
		})
	}
	for _, s := range result.SkippedItems {
		resp.SkippedItems = append(resp.SkippedItems, &pb.SkippedItemProto{Sku: s.SKU, Reason: s.Reason})
	}
	for _, e := range result.Errors {
		resp.Errors = append(resp.Errors, &pb.ReconcileErrorProto{Sku: e.SKU, Error: e.Error})
	}

	return resp, nil
}
