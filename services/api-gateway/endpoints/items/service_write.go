package itemep

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/field"
)

func (m *itemSvcImpl) CreateItem(ctx context.Context, req *CreateItemRequest) (*apiresource.Item, *apierror.APIError) {
	pbReq := &pb.CreateItemRequest{
		Type:         string(req.Type),
		Sku:          req.SKU,
		Description:  req.Description.Ptr(),
		CategoryId:   req.CategoryID,
		AttributeIds: req.AttributeIDs,
		Status:       req.Status.Ptr().StringPtr(),
		Supply:       req.Supply.Ptr().StringPtr(),
		UnitCost:     apirequest.OptionalUnitPriceToProto(req.UnitCost),
	}
	if sales, ok := req.Sales.Value(); ok {
		pbReq.Sales = &pb.ItemSalesInput{
			UnitPrice:        apirequest.OptionalUnitPriceToProto(sales.UnitPrice),
			PortalVisibility: optionalEnumString(sales.PortalVisibility),
			LeadTime:         optionalQuantityPatch(sales.LeadTime),
		}
		if lineID, ok := sales.ProductLineID.Value(); ok {
			pbReq.Sales.ProductLineId = &pb.StringPatch{Value: &lineID}
		}
	}
	if material, ok := req.Material.Value(); ok {
		pbReq.Material = &pb.ItemMaterialInput{
			OrderPoint: optionalQuantityPatch(material.OrderPoint),
			LeadTime:   optionalQuantityPatch(material.LeadTime),
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateItemResponse, error) {
			return m.coreClient.CreateItem(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return writtenItem(ctx, resp.Item)
}

func (m *itemSvcImpl) UpdateItem(ctx context.Context, req *UpdateItemRequest) (*apiresource.Item, *apierror.APIError) {
	pbReq := &pb.UpdateItemRequest{
		Id:         req.ItemID,
		Sku:        req.SKU.Ptr(),
		CategoryId: req.CategoryID.Ptr(),
		Status:     req.Status.Ptr().StringPtr(),
		Supply:     fieldpb.EnumClearableToProto(req.Supply),
		UnitCost:   apirequest.OptionalUnitPriceToProto(req.UnitCost),
		Sales:      updateSalesToProto(req.Sales.Value()),
		ClearSales: req.Sales.IsClear(),
		Material:   updateMaterialToProto(req.Material),
	}
	if !req.Description.IsUnset() {
		pbReq.UpdateDescription = true
		if v, ok := req.Description.Value(); ok {
			pbReq.Description = &v
		}
	}
	if ids, ok := req.AttributeIDs.Value(); ok {
		pbReq.AttributeIds = &pb.StringListPatch{Value: ids, Clear: len(ids) == 0}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateItemResponse, error) {
			return m.coreClient.UpdateItem(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return writtenItem(ctx, resp.Item)
}

func (m *itemSvcImpl) DeleteItem(ctx context.Context, req *DeleteItemRequest) (*apiresource.Item, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteItem(ctx, &pb.DeleteItemRequest{Id: req.ItemID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.Item{ID: req.ItemID}, nil
}

func (m *itemSvcImpl) BulkUpsertItems(ctx context.Context, req *BulkUpsertItemsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	rows := make([]*pb.UpsertItemInput, len(req.Items))
	for i, in := range req.Items {
		row := &pb.UpsertItemInput{
			Type:        string(in.Type),
			Sku:         in.SKU,
			Description: in.Description.Ptr(),
			Category:    &pb.ObjectIdentifier{Id: in.CategoryID},
			Status:      in.Status.Ptr().StringPtr(),
			Supply:      fieldpb.EnumClearableToProto(in.Supply),
			UnitCost:    apirequest.OptionalUnitPriceToProto(in.UnitCost),
			Sales:       updateSalesToProto(in.Sales.Value()),
			Material:    updateMaterialToProto(in.Material),
		}
		if attrs, ok := in.Attributes.Value(); ok {
			row.HasAttributes = true
			row.Attributes = make([]*pb.UpsertItemPropertyInput, len(attrs))
			for j, a := range attrs {
				row.Attributes[j] = &pb.UpsertItemPropertyInput{Name: a.Property, Value: a.Value}
			}
		}
		rows[i] = row
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.bulk_upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkUpsertItemsResponse, error) {
			return m.coreClient.BulkUpsertItems(ctx, &pb.BulkUpsertItemsRequest{Items: rows}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}

// writtenItem is the item a write saved, read back whole. A caller allowed to write items but not read them still gets
// the item it saved, as core answered the write with it.
func writtenItem(ctx context.Context, saved *pb.ItemInfo) (*apiresource.Item, *apierror.APIError) {
	item, apiErr := loadItemByID(ctx, saved.Id)
	if apierror.HasCode(apiErr, apierror.CodeInsufficientPermissions) {
		var categories []*pb.ItemCategoryInfo
		if saved.Category != nil {
			categories = []*pb.ItemCategoryInfo{saved.Category}
		}
		return resourceloaders.ItemsFromProto(ctx, []*pb.ItemInfo{saved}, categories, nil)[saved.Id].(*apiresource.Item), nil
	}
	return item, apiErr
}

func updateSalesToProto(sales UpdateItemSalesInput, ok bool) *pb.ItemSalesInput {
	if !ok {
		return nil
	}
	return &pb.ItemSalesInput{
		UnitPrice:        apirequest.OptionalUnitPriceToProto(sales.UnitPrice),
		ProductLineId:    fieldpb.StringClearableToProto(sales.ProductLineID),
		PortalVisibility: optionalEnumString(sales.PortalVisibility),
		LeadTime:         apirequest.QuantityFieldToProto(sales.LeadTime),
	}
}

func updateMaterialToProto(f field.Optional[UpdateItemMaterialInput]) *pb.ItemMaterialInput {
	material, ok := f.Value()
	if !ok {
		return nil
	}
	return &pb.ItemMaterialInput{
		OrderPoint: apirequest.QuantityFieldToProto(material.OrderPoint),
		LeadTime:   apirequest.QuantityFieldToProto(material.LeadTime),
	}
}

func optionalQuantityPatch(f field.Optional[apirequest.QuantityInput]) *pb.QuantityPatch {
	v, ok := f.Value()
	if !ok {
		return nil
	}
	return &pb.QuantityPatch{Value: &v.Value, UnitId: &v.UnitID}
}

func optionalEnumString[T ~string](f field.Optional[T]) *string {
	v, ok := f.Value()
	if !ok {
		return nil
	}
	s := string(v)
	return &s
}

func enumStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// RequirePreview9 answers a route that 1.0.forge removed only for 1.0.forge-preview.9 callers; anyone else gets the
// 404 a missing route gives, with replacement as the way forward.
func RequirePreview9(ctx context.Context, replacement string) *apierror.APIError {
	if v, ok := appctx.GetAPIVersionFromContext(ctx); ok && v.Equal(apiversion.V1_0_Forge_Preview9) {
		return nil
	}
	return apierror.NewNotFoundError("This route was removed in 1.0.forge. " + replacement)
}
