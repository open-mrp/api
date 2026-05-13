package volumediscountep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type VolumeDiscountSvc interface {
	ListVolumeDiscounts(ctx context.Context, req *ListVolumeDiscountsRequest) (*apiresource.List[apiresource.VolumeDiscount], *apierror.APIError)
	GetVolumeDiscount(ctx context.Context, req *RetrieveVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError)
	CreateVolumeDiscount(ctx context.Context, req *CreateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError)
	UpdateVolumeDiscount(ctx context.Context, req *UpdateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError)
	DeleteVolumeDiscount(ctx context.Context, req *DeleteVolumeDiscountRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type VolumeDiscountSvcConfig struct {
	CoreClient pb.CoreSalesServiceClient
}

type volumeDiscountSvcImpl struct {
	coreClient pb.CoreSalesServiceClient
}

var volumeDiscountSvcTracer = tracing.GetTracer("api-gateway.endpoints.volume-discounts.service")

func (c *VolumeDiscountSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("volume discount endpoint service: core client is required")
	}
	return nil
}

func NewVolumeDiscountSvc(config *VolumeDiscountSvcConfig) VolumeDiscountSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &volumeDiscountSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *volumeDiscountSvcImpl) ListVolumeDiscounts(ctx context.Context, req *ListVolumeDiscountsRequest) (*apiresource.List[apiresource.VolumeDiscount], *apierror.APIError) {
	pbReq := &pb.ListVolumeDiscountsRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListVolumeDiscountsResponse, error) {
			return m.coreClient.ListVolumeDiscounts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return VolumeDiscountListPresenter(resp), nil
}

func (m *volumeDiscountSvcImpl) GetVolumeDiscount(ctx context.Context, req *RetrieveVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
	pbReq := &pb.GetVolumeDiscountRequest{
		Id:       req.VolumeDiscountID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetVolumeDiscountResponse, error) {
			return m.coreClient.GetVolumeDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := VolumeDiscountPresenter(resp.VolumeDiscount)
	return &result, nil
}

func (m *volumeDiscountSvcImpl) CreateVolumeDiscount(ctx context.Context, req *CreateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
	tiers := make([]*pb.CreateVolumeDiscountTierInput, len(req.Tiers))
	for i, t := range req.Tiers {
		tiers[i] = &pb.CreateVolumeDiscountTierInput{
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			ParentTierId:       t.ParentTierID,
		}
	}

	pbReq := &pb.CreateVolumeDiscountRequest{
		Name:             req.Name,
		Tiers:            tiers,
		CustomerGroupIds: req.CustomerGroupIDs,
		ProductLineIds:   req.ProductLineIDs,
		CategoryIds:      req.CategoryIDs,
		AttributeIds:     req.AttributeIDs,
		UnitIds:          req.UnitIDs,
		Includes:         appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateVolumeDiscountResponse, error) {
			return m.coreClient.CreateVolumeDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := VolumeDiscountPresenter(resp.VolumeDiscount)
	return &result, nil
}

func (m *volumeDiscountSvcImpl) UpdateVolumeDiscount(ctx context.Context, req *UpdateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
	tiers := make([]*pb.UpdateVolumeDiscountTierInput, len(req.Tiers))
	for i, t := range req.Tiers {
		tiers[i] = &pb.UpdateVolumeDiscountTierInput{
			Id:                 t.ID,
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			ParentTierId:       t.ParentTierID,
		}
	}

	pbReq := &pb.UpdateVolumeDiscountRequest{
		Id:                req.VolumeDiscountID,
		Name:              req.Name,
		Tiers:             tiers,
		CustomerGroupIds:  req.CustomerGroupIDs,
		ProductLineIds:    req.ProductLineIDs,
		CategoryIds:       req.CategoryIDs,
		AttributeIds:      req.AttributeIDs,
		UnitIds:           req.UnitIDs,
		HasTiers:          req.HasTiers,
		HasCustomerGroups: req.HasCustomerGroups,
		HasProductLines:   req.HasProductLines,
		HasCategories:     req.HasCategories,
		HasAttributes:     req.HasAttributes,
		HasUnits:          req.HasUnits,
		Includes:          appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateVolumeDiscountResponse, error) {
			return m.coreClient.UpdateVolumeDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := VolumeDiscountPresenter(resp.VolumeDiscount)
	return &result, nil
}

func (m *volumeDiscountSvcImpl) DeleteVolumeDiscount(ctx context.Context, req *DeleteVolumeDiscountRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteVolumeDiscountRequest{
		Id: req.VolumeDiscountID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteVolumeDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
