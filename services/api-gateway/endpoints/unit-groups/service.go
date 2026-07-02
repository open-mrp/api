package unitgroupep

import (
	"context"
	"fmt"
	"strconv"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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
	// CoreClient (required) is the core-service gRPC client.
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
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
		Type:   req.Type.StringPtr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListUnitGroupsResponse, error) {
			return m.coreClient.ListUnitGroups(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.UnitGroups))
	for i, ug := range resp.UnitGroups {
		ids[i] = ug.Id
	}
	loaded, apiErr := resourceloaders.LoadUnitGroups(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.UnitGroup, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.UnitGroup)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *unitGroupSvcImpl) GetUnitGroup(ctx context.Context, req *RetrieveUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
	return loadUnitGroupByID(ctx, req.UnitGroupID)
}

func (m *unitGroupSvcImpl) CreateUnitGroup(ctx context.Context, req *CreateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
	associatedUnits := make([]*pb.CreateUnitGroupUnitParam, len(req.AssociatedUnits))
	for i, au := range req.AssociatedUnits {
		discountPct := "1"
		if v, ok := au.DiscountPercentage.Value(); ok {
			discountPct = strconv.FormatFloat(v, 'f', -1, 64)
		}
		discountFixed := "0"
		if v, ok := au.DiscountFixed.Value(); ok {
			discountFixed = strconv.FormatFloat(v, 'f', -1, 64)
		}
		isVisible := true
		if v, ok := au.CustomerPortalVisibility.Value(); ok {
			isVisible = v == constants.CustomerPortalVisibilityVisible
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
		Notes:           req.Notes.Ptr(),
		Type:            string(req.Type),
		BaseUnitId:      req.BaseUnitID,
		UnitConversions: associatedUnits,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateUnitGroupResponse, error) {
			return m.coreClient.CreateUnitGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadUnitGroupByID(ctx, resp.UnitGroup.Id)
}

func (m *unitGroupSvcImpl) UpdateUnitGroup(ctx context.Context, req *UpdateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
	pbReq := &pb.UpdateUnitGroupRequest{
		Id:         req.UnitGroupID,
		Name:       req.Name.Ptr(),
		BaseUnitId: req.BaseUnitID.Ptr(),
	}

	pbReq.Notes = field.StringClearableToProto(req.Notes)

	if associatedUnitsReq, ok := req.AssociatedUnits.Value(); ok {
		pbReq.UpdateUnitConversions = true
		associatedUnits := make([]*pb.CreateUnitGroupUnitParam, len(associatedUnitsReq))
		for i, au := range associatedUnitsReq {
			discountPct := "1"
			if v, ok := au.DiscountPercentage.Value(); ok {
				discountPct = strconv.FormatFloat(v, 'f', -1, 64)
			}
			discountFixed := "0"
			if v, ok := au.DiscountFixed.Value(); ok {
				discountFixed = strconv.FormatFloat(v, 'f', -1, 64)
			}
			isVisible := true
			if v, ok := au.CustomerPortalVisibility.Value(); ok {
				isVisible = v == constants.CustomerPortalVisibilityVisible
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

	return loadUnitGroupByID(ctx, resp.UnitGroup.Id)
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
	if v, ok := req.DiscountPercentage.Value(); ok {
		discountPct = strconv.FormatFloat(v, 'f', -1, 64)
	}
	discountFixed := "0"
	if v, ok := req.DiscountFixed.Value(); ok {
		discountFixed = strconv.FormatFloat(v, 'f', -1, 64)
	}
	isVisible := true
	if v, ok := req.CustomerPortalVisibility.Value(); ok {
		isVisible = v == constants.CustomerPortalVisibilityVisible
	}

	pbReq := &pb.UpsertUnitGroupUnitRequest{
		UnitGroupId:        req.UnitGroupID,
		UnitId:             req.UnitID,
		DiscountPercentage: discountPct,
		DiscountFixed:      discountFixed,
		IsVisible:          proto.Bool(isVisible),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.create_unit", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpsertUnitGroupUnitResponse, error) {
			return m.coreClient.UpsertUnitGroupUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadUnitGroupUnitByID(ctx, resp.UnitGroupUnit.Id)
}

func (m *unitGroupSvcImpl) UpdateUnitGroupUnit(ctx context.Context, req *UpdateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
	pbReq := &pb.UpsertUnitGroupUnitRequest{
		UnitGroupId:     req.UnitGroupID,
		UnitGroupUnitId: req.AssociatedUnitID,
	}

	if v, ok := req.UnitID.Value(); ok {
		pbReq.UnitId = v
	}
	if v, ok := req.DiscountPercentage.Value(); ok {
		pbReq.DiscountPercentage = strconv.FormatFloat(v, 'f', -1, 64)
	}
	if v, ok := req.DiscountFixed.Value(); ok {
		pbReq.DiscountFixed = strconv.FormatFloat(v, 'f', -1, 64)
	}
	if v, ok := req.CustomerPortalVisibility.Value(); ok {
		pbReq.IsVisible = proto.Bool(v == constants.CustomerPortalVisibilityVisible)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.update_unit", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpsertUnitGroupUnitResponse, error) {
			return m.coreClient.UpsertUnitGroupUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadUnitGroupUnitByID(ctx, resp.UnitGroupUnit.Id)
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
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitGroupSvcTracer, "service.unit_groups.list_units", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListUnitGroupUnitsResponse, error) {
			return m.coreClient.ListUnitGroupUnits(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.UnitGroupUnits))
	for i, u := range resp.UnitGroupUnits {
		ids[i] = u.Id
	}
	loaded, apiErr := resourceloaders.LoadUnitGroupUnits(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.UnitGroupUnit, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.UnitGroupUnit)))
		}
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (m *unitGroupSvcImpl) GetUnitGroupUnit(ctx context.Context, req *RetrieveUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
	return loadUnitGroupUnitByID(ctx, req.UnitGroupUnitID)
}

// loadUnitGroupByID wraps the single-ID load pattern used after every
// mutation and for the retrieve endpoint.
func loadUnitGroupByID(ctx context.Context, id string) (*apiresource.UnitGroup, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadUnitGroups(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Unit group not found.")
	}
	return v.(*apiresource.UnitGroup), nil
}

// loadUnitGroupUnitByID wraps the single-ID load pattern used after every
// mutation and for the retrieve endpoint.
func loadUnitGroupUnitByID(ctx context.Context, id string) (*apiresource.UnitGroupUnit, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadUnitGroupUnits(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Unit group unit not found.")
	}
	return v.(*apiresource.UnitGroupUnit), nil
}
