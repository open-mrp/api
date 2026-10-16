package supplieritemep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SupplierItemSvc interface {
	ListSupplierItems(ctx context.Context, req *ListSupplierItemsRequest) (*apiresource.List[apiresource.SupplierItem], *apierror.APIError)
	GetSupplierItem(ctx context.Context, req *RetrieveSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError)
	CreateSupplierItem(ctx context.Context, req *CreateSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError)
	UpdateSupplierItem(ctx context.Context, req *UpdateSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError)
	DeleteSupplierItem(ctx context.Context, req *DeleteSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError)
}

type SupplierItemSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

func (c *SupplierItemSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("supplier item endpoint service: core client is required")
	}
	return nil
}

type supplierItemSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var supplierItemSvcTracer = tracing.GetTracer("api-gateway.endpoints.supplier-items.service")

// supplierItemIncludeFields are the include keys every supplier item endpoint accepts.
var supplierItemIncludeFields = []string{"supplier", "item"}

func NewSupplierItemSvc(config *SupplierItemSvcConfig) SupplierItemSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &supplierItemSvcImpl{coreClient: config.CoreClient}
}

func (s *supplierItemSvcImpl) ListSupplierItems(ctx context.Context, req *ListSupplierItemsRequest) (*apiresource.List[apiresource.SupplierItem], *apierror.APIError) {
	if apiErr := req.CreatedRange.Validate(); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.ListSupplierItemsRequest{
		SupplierId: req.SupplierID,
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		Query:      req.Query,
		IsActive:   constants.IsActiveFilter(req.Statuses),
		ItemIds:    req.ItemIDs,
	}
	if req.CreatedAfter != nil {
		pbReq.CreatedAfter = timestamppb.New(*req.CreatedAfter)
	}
	if req.CreatedBefore != nil {
		pbReq.CreatedBefore = timestamppb.New(*req.CreatedBefore)
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierItemSvcTracer, "service.supplier_items.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSupplierItemsResponse, error) {
			return s.coreClient.ListSupplierItems(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.SupplierItem, len(resp.SupplierItems))
	for i, item := range resp.SupplierItems {
		items[i] = *resourceloaders.SupplierItemFromProto(ctx, item)
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (s *supplierItemSvcImpl) GetSupplierItem(ctx context.Context, req *RetrieveSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError) {
	pbReq := &pb.GetSupplierItemRequest{SupplierId: req.SupplierID, Id: req.SupplierItemID}
	return s.call(ctx, "service.supplier_items.get", func(ctx context.Context, opts ...grpc.CallOption) (*pb.SupplierItemResponse, error) {
		return s.coreClient.GetSupplierItem(ctx, pbReq, opts...)
	})
}

func (s *supplierItemSvcImpl) CreateSupplierItem(ctx context.Context, req *CreateSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError) {
	pbReq := &pb.CreateSupplierItemRequest{
		SupplierId:          req.SupplierID,
		ItemId:              req.ItemID,
		SupplierSku:         req.SupplierSKU,
		SupplierDescription: req.SupplierDescription.Ptr(),
	}
	if v, ok := req.Status.Value(); ok {
		status := string(v)
		pbReq.Status = &status
	}
	return s.call(ctx, "service.supplier_items.create", func(ctx context.Context, opts ...grpc.CallOption) (*pb.SupplierItemResponse, error) {
		return s.coreClient.CreateSupplierItem(ctx, pbReq, opts...)
	})
}

func (s *supplierItemSvcImpl) UpdateSupplierItem(ctx context.Context, req *UpdateSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError) {
	pbReq := &pb.UpdateSupplierItemRequest{
		SupplierId:          req.SupplierID,
		Id:                  req.SupplierItemID,
		SupplierSku:         req.SupplierSKU.Ptr(),
		SupplierDescription: fieldpb.StringClearableToProto(req.SupplierDescription),
	}
	if v, ok := req.Status.Value(); ok {
		status := string(v)
		pbReq.Status = &status
	}
	return s.call(ctx, "service.supplier_items.update", func(ctx context.Context, opts ...grpc.CallOption) (*pb.SupplierItemResponse, error) {
		return s.coreClient.UpdateSupplierItem(ctx, pbReq, opts...)
	})
}

func (s *supplierItemSvcImpl) DeleteSupplierItem(ctx context.Context, req *DeleteSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError) {
	pbReq := &pb.DeleteSupplierItemRequest{SupplierId: req.SupplierID, Id: req.SupplierItemID}
	return s.call(ctx, "service.supplier_items.delete", func(ctx context.Context, opts ...grpc.CallOption) (*pb.SupplierItemResponse, error) {
		return s.coreClient.DeleteSupplierItem(ctx, pbReq, opts...)
	})
}

func (s *supplierItemSvcImpl) call(ctx context.Context, name string, fn func(context.Context, ...grpc.CallOption) (*pb.SupplierItemResponse, error)) (*apiresource.SupplierItem, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, supplierItemSvcTracer, name, domain.ServiceName, fn)
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.SupplierItemFromProto(ctx, resp.SupplierItem), nil
}
