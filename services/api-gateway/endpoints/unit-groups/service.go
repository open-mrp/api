package unitgroupep

import (
	"context"
	"fmt"
	"strconv"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	ownerutil "github.com/augno/api/services/api-gateway/internal/owner"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UnitGroupSvc interface {
	ListUnitGroups(ctx context.Context, req *ListUnitGroupsRequest) (*apiresource.List[apiresource.UnitGroup], *apierror.APIError)
	GetUnitGroup(ctx context.Context, req *RetrieveUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError)
	CreateUnitGroup(ctx context.Context, req *CreateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError)
	UpdateUnitGroup(ctx context.Context, req *UpdateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError)
	DeleteUnitGroup(ctx context.Context, req *DeleteUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError)
	CreateUnitGroupUnit(ctx context.Context, req *CreateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError)
	UpdateUnitGroupUnit(ctx context.Context, req *UpdateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError)
	DeleteUnitGroupUnit(ctx context.Context, req *DeleteUnitGroupUnitRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListUnitGroupUnits(ctx context.Context, req *ListUnitGroupUnitsRequest) (*apiresource.List[apiresource.UnitGroupUnit], *apierror.APIError)
	GetUnitGroupUnit(ctx context.Context, req *RetrieveUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError)
}

type UnitGroupSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type unitGroupSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var unitGroupSvcTracer = tracing.GetTracer("api-gateway.endpoints.unit_groups.service")

func (c *UnitGroupSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("unit group endpoint service: core client is required")
	}
	return nil
}

func NewUnitGroupSvc(config *UnitGroupSvcConfig) UnitGroupSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &unitGroupSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *unitGroupSvcImpl) ListUnitGroups(ctx context.Context, req *ListUnitGroupsRequest) (*apiresource.List[apiresource.UnitGroup], *apierror.APIError) {
	pbReq := &pb.ListUnitGroupsRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Type:     req.Type.StringPtr(),
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListUnitGroupsResponse, error) {
			return m.coreClient.ListUnitGroups(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	var ownerAccount *apiresource.Account
	for _, ug := range resp.UnitGroups {
		if ug.AccountId != nil {
			ownerAccount = ownerutil.ResolveOwnerAccount(ctx, m.coreClient, ug.AccountId)
			break
		}
	}

	return UnitGroupListPresenter(ctx, resp, ownerAccount), nil
}

func (m *unitGroupSvcImpl) GetUnitGroup(ctx context.Context, req *RetrieveUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
	pbReq := &pb.GetUnitGroupRequest{
		Id:       req.UnitGroupID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetUnitGroupResponse, error) {
			return m.coreClient.GetUnitGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.UnitGroup.AccountId)
	result := UnitGroupPresenter(resp.UnitGroup, ownerAccount)
	return &result, nil
}

func (m *unitGroupSvcImpl) CreateUnitGroup(ctx context.Context, req *CreateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
	associatedUnits := make([]*pb.CreateUnitGroupUnitParam, len(req.AssociatedUnits))
	for i, au := range req.AssociatedUnits {
		discountPct := "1"
		if au.DiscountPercentage != nil {
			discountPct = strconv.FormatFloat(*au.DiscountPercentage, 'f', -1, 64)
		}
		discountFixed := "0"
		if au.DiscountFixed != nil {
			discountFixed = strconv.FormatFloat(*au.DiscountFixed, 'f', -1, 64)
		}
		isVisible := true
		if au.CustomerPortalVisibility != nil {
			isVisible = *au.CustomerPortalVisibility == constants.CustomerPortalVisibilityVisible
		}
		associatedUnits[i] = &pb.CreateUnitGroupUnitParam{
			UnitId:             au.UnitID,
			DiscountPercentage: discountPct,
			DiscountFixed:      discountFixed,
			IsVisible:          isVisible,
		}
	}

	pbReq := &pb.CreateUnitGroupRequest{
		Name:            req.Name,
		Notes:           req.Notes,
		Type:            string(req.Type),
		BaseUnitId:      req.BaseUnitID,
		UnitConversions: associatedUnits,
		Includes:        appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateUnitGroupResponse, error) {
			return m.coreClient.CreateUnitGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.UnitGroup.AccountId)
	result := UnitGroupPresenter(resp.UnitGroup, ownerAccount)
	return &result, nil
}

func (m *unitGroupSvcImpl) UpdateUnitGroup(ctx context.Context, req *UpdateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
	pbReq := &pb.UpdateUnitGroupRequest{
		Id:         req.UnitGroupID,
		Name:       req.Name,
		BaseUnitId: req.BaseUnitID,
		Includes:   appctx.GetRequestedIncludeKeys(ctx),
	}

	if req.Notes != nil {
		pbReq.UpdateNotes = true
		pbReq.Notes = req.Notes
	}

	if req.AssociatedUnits != nil {
		pbReq.UpdateUnitConversions = true
		associatedUnits := make([]*pb.CreateUnitGroupUnitParam, len(*req.AssociatedUnits))
		for i, au := range *req.AssociatedUnits {
			discountPct := "1"
			if au.DiscountPercentage != nil {
				discountPct = strconv.FormatFloat(*au.DiscountPercentage, 'f', -1, 64)
			}
			discountFixed := "0"
			if au.DiscountFixed != nil {
				discountFixed = strconv.FormatFloat(*au.DiscountFixed, 'f', -1, 64)
			}
			isVisible := true
			if au.CustomerPortalVisibility != nil {
				isVisible = *au.CustomerPortalVisibility == constants.CustomerPortalVisibilityVisible
			}
			associatedUnits[i] = &pb.CreateUnitGroupUnitParam{
				UnitId:             au.UnitID,
				DiscountPercentage: discountPct,
				DiscountFixed:      discountFixed,
				IsVisible:          isVisible,
			}
		}
		pbReq.UnitConversions = associatedUnits
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateUnitGroupResponse, error) {
			return m.coreClient.UpdateUnitGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.UnitGroup.AccountId)
	result := UnitGroupPresenter(resp.UnitGroup, ownerAccount)
	return &result, nil
}

func (m *unitGroupSvcImpl) DeleteUnitGroup(ctx context.Context, req *DeleteUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteUnitGroupRequest{
		Id: req.UnitGroupID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteUnitGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *unitGroupSvcImpl) CreateUnitGroupUnit(ctx context.Context, req *CreateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
	discountPct := "1"
	if req.DiscountPercentage != nil {
		discountPct = strconv.FormatFloat(*req.DiscountPercentage, 'f', -1, 64)
	}
	discountFixed := "0"
	if req.DiscountFixed != nil {
		discountFixed = strconv.FormatFloat(*req.DiscountFixed, 'f', -1, 64)
	}
	isVisible := true
	if req.CustomerPortalVisibility != nil {
		isVisible = *req.CustomerPortalVisibility == constants.CustomerPortalVisibilityVisible
	}

	pbReq := &pb.UpsertUnitGroupUnitRequest{
		UnitGroupId:        req.UnitGroupID,
		UnitId:             req.UnitID,
		DiscountPercentage: discountPct,
		DiscountFixed:      discountFixed,
		IsVisible:          isVisible,
		Includes:           appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.create_unit", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpsertUnitGroupUnitResponse, error) {
			return m.coreClient.UpsertUnitGroupUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := UnitGroupUnitPresenter(resp.UnitGroupUnit)
	return &result, nil
}

func (m *unitGroupSvcImpl) UpdateUnitGroupUnit(ctx context.Context, req *UpdateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
	pbReq := &pb.UpsertUnitGroupUnitRequest{
		UnitGroupId:     req.UnitGroupID,
		UnitGroupUnitId: req.AssociatedUnitID,
		Includes:        appctx.GetRequestedIncludeKeys(ctx),
	}

	if req.UnitID != nil {
		pbReq.UnitId = *req.UnitID
	}
	if req.DiscountPercentage != nil {
		pbReq.DiscountPercentage = strconv.FormatFloat(*req.DiscountPercentage, 'f', -1, 64)
	}
	if req.DiscountFixed != nil {
		pbReq.DiscountFixed = strconv.FormatFloat(*req.DiscountFixed, 'f', -1, 64)
	}
	if req.CustomerPortalVisibility != nil {
		pbReq.IsVisible = *req.CustomerPortalVisibility == constants.CustomerPortalVisibilityVisible
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.update_unit", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpsertUnitGroupUnitResponse, error) {
			return m.coreClient.UpsertUnitGroupUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := UnitGroupUnitPresenter(resp.UnitGroupUnit)
	return &result, nil
}

func (m *unitGroupSvcImpl) DeleteUnitGroupUnit(ctx context.Context, req *DeleteUnitGroupUnitRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteUnitGroupUnitRequest{
		UnitGroupId:     req.UnitGroupID,
		UnitGroupUnitId: req.AssociatedUnitID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.delete_unit", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteUnitGroupUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *unitGroupSvcImpl) ListUnitGroupUnits(ctx context.Context, req *ListUnitGroupUnitsRequest) (*apiresource.List[apiresource.UnitGroupUnit], *apierror.APIError) {
	pbReq := &pb.ListUnitGroupUnitsRequest{
		UnitGroupId: req.UnitGroupID,
		Includes:    appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.list_units", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListUnitGroupUnitsResponse, error) {
			return m.coreClient.ListUnitGroupUnits(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return UnitGroupUnitListPresenter(ctx, resp), nil
}

func (m *unitGroupSvcImpl) GetUnitGroupUnit(ctx context.Context, req *RetrieveUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
	pbReq := &pb.GetUnitGroupUnitRequest{
		UnitGroupId:     req.UnitGroupID,
		UnitGroupUnitId: req.UnitGroupUnitID,
		Includes:        appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.get_unit", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetUnitGroupUnitResponse, error) {
			return m.coreClient.GetUnitGroupUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := UnitGroupUnitPresenter(resp.UnitGroupUnit)
	return &result, nil
}
