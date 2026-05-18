package itemcategoryep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	ownerutil "github.com/augno/api/services/api-gateway/internal/owner"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ItemCategorySvc interface {
	ListItemCategories(ctx context.Context, req *ListItemCategoriesRequest) (*apiresource.List[apiresource.ItemCategory], *apierror.APIError)
	GetItemCategory(ctx context.Context, req *RetrieveItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError)
	CreateItemCategory(ctx context.Context, req *CreateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError)
	UpdateItemCategory(ctx context.Context, req *UpdateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError)
	DeleteItemCategory(ctx context.Context, req *DeleteItemCategoryRequest) (*apiresource.EmptyResource, *apierror.APIError)
	AddItemCategoryProperty(ctx context.Context, req *AddItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError)
	RemoveItemCategoryProperty(ctx context.Context, req *RemoveItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ChangeItemCategoryUnitGroup(ctx context.Context, req *ChangeItemCategoryUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ItemCategorySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type itemCategorySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var itemCategorySvcTracer = tracing.GetTracer("api-gateway.endpoints.item-categories.service")

func (c *ItemCategorySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("item category endpoint service: core client is required")
	}
	return nil
}

func NewItemCategorySvc(config *ItemCategorySvcConfig) ItemCategorySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &itemCategorySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *itemCategorySvcImpl) ListItemCategories(ctx context.Context, req *ListItemCategoriesRequest) (*apiresource.List[apiresource.ItemCategory], *apierror.APIError) {
	pbReq := &pb.ListItemCategoriesRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Type:     req.Type.StringPtr(),
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemCategorySvcTracer, "service.item-categories.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListItemCategoriesResponse, error) {
			return m.coreClient.ListItemCategories(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	var ownerAccount *apiresource.Account
	for _, ic := range resp.ItemCategories {
		if ic.AccountId != nil {
			ownerAccount = ownerutil.ResolveOwnerAccount(ctx, m.coreClient, ic.AccountId)
			break
		}
	}

	return ItemCategoryListPresenter(ctx, resp, ownerAccount), nil
}

func (m *itemCategorySvcImpl) GetItemCategory(ctx context.Context, req *RetrieveItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
	pbReq := &pb.GetItemCategoryRequest{
		Id:       req.ItemCategoryID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemCategorySvcTracer, "service.item-categories.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetItemCategoryResponse, error) {
			return m.coreClient.GetItemCategory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.ItemCategory.AccountId)
	result := ItemCategoryPresenter(resp.ItemCategory, ownerAccount)
	return &result, nil
}

func (m *itemCategorySvcImpl) CreateItemCategory(ctx context.Context, req *CreateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
	pbReq := &pb.CreateItemCategoryRequest{
		Name:        req.Name,
		Type:        string(req.Type),
		UnitGroupId: req.UnitGroupID,
		Includes:    appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemCategorySvcTracer, "service.item-categories.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateItemCategoryResponse, error) {
			return m.coreClient.CreateItemCategory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.ItemCategory.AccountId)
	result := ItemCategoryPresenter(resp.ItemCategory, ownerAccount)
	return &result, nil
}

func (m *itemCategorySvcImpl) UpdateItemCategory(ctx context.Context, req *UpdateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
	pbReq := &pb.UpdateItemCategoryRequest{
		Id:       req.ItemCategoryID,
		Name:     req.Name,
		Notes:    req.Notes,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, itemCategorySvcTracer, "service.item-categories.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateItemCategoryResponse, error) {
			return m.coreClient.UpdateItemCategory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.ItemCategory.AccountId)
	result := ItemCategoryPresenter(resp.ItemCategory, ownerAccount)
	return &result, nil
}

func (m *itemCategorySvcImpl) DeleteItemCategory(ctx context.Context, req *DeleteItemCategoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteItemCategoryRequest{
		Id: req.ItemCategoryID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, itemCategorySvcTracer, "service.item-categories.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteItemCategory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *itemCategorySvcImpl) AddItemCategoryProperty(ctx context.Context, req *AddItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.AddItemCategoryPropertyRequest{
		Id:         req.ItemCategoryID,
		PropertyId: req.PropertyID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, itemCategorySvcTracer, "service.item-categories.add-property", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.AddItemCategoryProperty(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *itemCategorySvcImpl) RemoveItemCategoryProperty(ctx context.Context, req *RemoveItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.RemoveItemCategoryPropertyRequest{
		Id:         req.ItemCategoryID,
		PropertyId: req.PropertyID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, itemCategorySvcTracer, "service.item-categories.remove-property", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.RemoveItemCategoryProperty(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *itemCategorySvcImpl) ChangeItemCategoryUnitGroup(ctx context.Context, req *ChangeItemCategoryUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.ChangeItemCategoryUnitGroupRequest{
		Id:          req.ItemCategoryID,
		UnitGroupId: req.UnitGroupID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, itemCategorySvcTracer, "service.item-categories.change-unit-group", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.ChangeItemCategoryUnitGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
