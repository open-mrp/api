package itemep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	"github.com/open-mrp/api/services/api-gateway/internal/export"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ItemSvc interface {
	ListItems(ctx context.Context, req *ListItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError)
	GetItem(ctx context.Context, req *RetrieveItemRequest) (*apiresource.Item, *apierror.APIError)
	GetItemInventory(ctx context.Context, req *RetrieveItemInventoryRequest) (*apiresource.ItemInventory, *apierror.APIError)
	GetItemLotDefault(ctx context.Context, req *RetrieveItemLotDefaultRequest) (*apiresource.ItemLotDefault, *apierror.APIError)
	GetItemCosts(ctx context.Context, req *GetItemCostsRequest) (*apiresource.ItemCosts, *apierror.APIError)
	GetItemTrends(ctx context.Context, req *GetItemTrendsRequest) (*apiresource.ItemTrends, *apierror.APIError)
	ExportItems(ctx context.Context, req *ExportItemsRequest) (*httptransport.FileDownload, *apierror.APIError)
	AddItemAttribute(ctx context.Context, req *AddItemAttributeRequest) (*apiresource.Item, *apierror.APIError)
	RemoveItemAttribute(ctx context.Context, req *RemoveItemAttributeRequest) (*apiresource.Item, *apierror.APIError)
	ChangeItemCategory(ctx context.Context, req *ChangeItemCategoryRequest) (*apiresource.Item, *apierror.APIError)
	UpdateItemInventory(ctx context.Context, req *UpdateItemInventoryRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkCreateItems(ctx context.Context, req *BulkCreateItemsRequest) (*apiresource.BulkCreateItemsResponse, *apierror.APIError)
	BulkReconcileItems(ctx context.Context, req *BulkReconcileItemsRequest) (*apiresource.BulkReconcileItemsResponse, *apierror.APIError)
}

type ItemSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type itemSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var itemSvcTracer = tracing.GetTracer("api-gateway.endpoints.items.service")

func (c *ItemSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("item endpoint service: core client is required")
	}
	return nil
}

func NewItemSvc(config *ItemSvcConfig) ItemSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &itemSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *itemSvcImpl) ListItems(ctx context.Context, req *ListItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
	onlyInitialSubassemblies := req.SubassemblyFilter != nil &&
		*req.SubassemblyFilter == constants.SubassemblyFilterInitialOnly

	pbReq := &pb.ListItemsRequest{
		Cursor:                   req.Cursor,
		Limit:                    req.Limit,
		Query:                    req.Query,
		Types:                    req.Types,
		CategoryIds:              req.CategoryIDs,
		AttributeIds:             req.AttributeIDs,
		SupplierId:               req.SupplierID,
		IsExactMatch:             false,
		OnlyInitialSubassemblies: onlyInitialSubassemblies,
		ProductLineIds:           req.ProductLineIDs,
		CustomerIds:              req.CustomerIDs,
	}

	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListItemsResponse, error) {
			return m.coreClient.ListItems(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Items))
	for i, item := range resp.Items {
		ids[i] = item.Id
	}
	loaded, apiErr := resourceloaders.LoadItems(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.Item, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.Item)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *itemSvcImpl) GetItem(ctx context.Context, req *RetrieveItemRequest) (*apiresource.Item, *apierror.APIError) {
	return loadItemByID(ctx, req.ItemID)
}

func (m *itemSvcImpl) GetItemInventory(ctx context.Context, req *RetrieveItemInventoryRequest) (*apiresource.ItemInventory, *apierror.APIError) {
	pbReq := &pb.GetItemInventoryRequest{
		Id: req.ItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.get_inventory", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetItemInventoryResponse, error) {
			return m.coreClient.GetItemInventory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, ItemInventoryUnitIDs(resp)...)
	if apiErr != nil {
		return nil, apiErr
	}

	return ItemInventoryPresenter(ctx, resp, units), nil
}

func (m *itemSvcImpl) GetItemLotDefault(ctx context.Context, req *RetrieveItemLotDefaultRequest) (*apiresource.ItemLotDefault, *apierror.APIError) {
	pbReq := &pb.GetItemLotDefaultRequest{ItemId: req.ItemID}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.get_lot_default", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetItemLotDefaultResponse, error) {
			return m.coreClient.GetItemLotDefault(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	lot := &apiresource.ItemLotDefault{
		Object:   constants.ObjectTypeItemLotDefault,
		Item:     apiresource.NewEntity(resp.ItemId, constants.ObjectTypeItem, nil, nil),
		Quantity: resp.Quantity,
		Source:   constants.ItemLotSource(resp.Source),
	}
	if resp.ProductLineId != nil && *resp.ProductLineId != "" {
		lot.ProductLine = apiresource.NewEntity(*resp.ProductLineId, constants.ObjectTypeProductLine, nil, nil)
	}
	// The unit id is loader-side metadata rather than a public field: the `unit` include
	// resolves it into the nested sub-resource, keyed by the item the lot was resolved for.
	if resp.UnitId != nil && *resp.UnitId != "" {
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeItemLotDefault, resp.ItemId, "unit_id", *resp.UnitId)
	}
	return lot, nil
}

func (m *itemSvcImpl) GetItemCosts(ctx context.Context, req *GetItemCostsRequest) (*apiresource.ItemCosts, *apierror.APIError) {
	pbReq := &pb.GetItemCostsRequest{
		Id: req.ItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.get_costs", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetItemCostsResponse, error) {
			return m.coreClient.GetItemCosts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, resp.NumeratorUnitId, resp.UnitId)
	if apiErr != nil {
		return nil, apiErr
	}

	return ItemCostsPresenter(resp, units), nil
}

func (m *itemSvcImpl) GetItemTrends(ctx context.Context, req *GetItemTrendsRequest) (*apiresource.ItemTrends, *apierror.APIError) {
	pbReq := &pb.GetItemTrendsRequest{
		Id:        req.ItemID,
		TrendType: string(req.TrendType),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.get_trends", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetItemTrendsResponse, error) {
			return m.coreClient.GetItemTrends(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ItemTrendsPresenter(resp), nil
}

func (m *itemSvcImpl) ExportItems(ctx context.Context, req *ExportItemsRequest) (*httptransport.FileDownload, *apierror.APIError) {
	pbReq := &pb.ExportItemsRequest{}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportItemsResponse, error) {
			return m.coreClient.ExportItems(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	body, err := export.ItemsToExcel(resp)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to build export file.")
	}

	return &httptransport.FileDownload{
		ContentType: export.ExcelContentType,
		Filename:    "items.xlsx",
		Body:        body,
	}, nil
}

func (m *itemSvcImpl) AddItemAttribute(ctx context.Context, req *AddItemAttributeRequest) (*apiresource.Item, *apierror.APIError) {
	pbReq := &pb.AddItemAttributeRequest{
		ItemId:      req.ItemID,
		AttributeId: req.AttributeID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.add_attribute", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AddItemAttributeResponse, error) {
			return m.coreClient.AddItemAttribute(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadItemByID(ctx, resp.Item.Id)
}

func (m *itemSvcImpl) RemoveItemAttribute(ctx context.Context, req *RemoveItemAttributeRequest) (*apiresource.Item, *apierror.APIError) {
	pbReq := &pb.RemoveItemAttributeRequest{
		ItemId:      req.ItemID,
		AttributeId: req.AttributeID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.remove_attribute", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RemoveItemAttributeResponse, error) {
			return m.coreClient.RemoveItemAttribute(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadItemByID(ctx, resp.Item.Id)
}

func (m *itemSvcImpl) ChangeItemCategory(ctx context.Context, req *ChangeItemCategoryRequest) (*apiresource.Item, *apierror.APIError) {
	pbReq := &pb.ChangeItemCategoryRequest{
		ItemId:     req.ItemID,
		CategoryId: req.CategoryID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.change_category", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangeItemCategoryResponse, error) {
			return m.coreClient.ChangeItemCategory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadItemByID(ctx, resp.Item.Id)
}

func (m *itemSvcImpl) UpdateItemInventory(ctx context.Context, req *UpdateItemInventoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	var reconcile *bool
	if op, ok := req.Operation.Value(); ok {
		v := op == constants.InventoryUpdateOperationReconcile
		reconcile = &v
	}

	pbReq := &pb.UpdateItemInventoryRequest{
		ItemId:     req.ItemID,
		Measure:    req.Quantity.Value,
		UnitId:     req.Quantity.UnitID,
		Reconcile:  reconcile,
		CustomerId: req.CustomerID.Ptr(),
		LocationId: req.LocationID.Ptr(),
		LotNumber:  req.LotNumber.Ptr(),
	}

	_, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.update_inventory", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateItemInventoryResponse, error) {
			return m.coreClient.UpdateItemInventory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *itemSvcImpl) BulkCreateItems(ctx context.Context, req *BulkCreateItemsRequest) (*apiresource.BulkCreateItemsResponse, *apierror.APIError) {
	pbItems := make([]*pb.BulkCreateItemInput, len(req.Items))
	for i, item := range req.Items {
		pbItems[i] = &pb.BulkCreateItemInput{
			Sku:            item.SKU,
			Description:    item.Description.Ptr(),
			ItemCategoryId: item.ItemCategoryID,
			ProductLineId:  item.ProductLineID.Ptr(),
		}
	}

	pbReq := &pb.BulkCreateItemsRequest{
		Items: pbItems,
		Type:  string(req.Type),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.bulk_create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkCreateItemsResponse, error) {
			return m.coreClient.BulkCreateItems(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	results := make([]apiresource.BulkCreateItemResult, len(resp.Results))
	for i, r := range resp.Results {
		status := constants.BulkResultStatusFailed
		if r.Success {
			status = constants.BulkResultStatusCreated
		}
		results[i] = apiresource.BulkCreateItemResult{
			SKU:    r.Sku,
			Status: status,
			Error:  r.Error,
			ItemID: r.ItemId,
		}
	}

	return &apiresource.BulkCreateItemsResponse{
		Object: "list",
		Data:   results,
	}, nil
}

func (m *itemSvcImpl) BulkReconcileItems(ctx context.Context, req *BulkReconcileItemsRequest) (*apiresource.BulkReconcileItemsResponse, *apierror.APIError) {
	pbData := make([]*pb.BulkReconcileItemInput, len(req.Data))
	for i, d := range req.Data {
		pbData[i] = &pb.BulkReconcileItemInput{
			Sku:     d.SKU,
			Unit:    d.Unit,
			Measure: d.Quantity,
		}
	}

	pbReq := &pb.BulkReconcileItemsRequest{
		Data:          pbData,
		ReconcileType: string(req.ReconcileType),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.bulk_reconcile", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkReconcileItemsResponse, error) {
			return m.coreClient.BulkReconcileItems(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, BulkReconcileUnitIDs(resp)...)
	if apiErr != nil {
		return nil, apiErr
	}

	return BulkReconcileItemsPresenter(resp, units), nil
}

func loadItemByID(ctx context.Context, id string) (*apiresource.Item, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadItems(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Item not found.")
	}
	return v.(*apiresource.Item), nil
}
