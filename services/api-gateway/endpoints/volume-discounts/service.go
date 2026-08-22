package volumediscountep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
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
	// CoreClient (required) is the core-service sales gRPC client.
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
		Includes: resourcekit.FilterIncludes(ctx, volumeDiscountIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListVolumeDiscountsResponse, error) {
			return m.coreClient.ListVolumeDiscounts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return volumeDiscountListFromProto(ctx, resp), nil
}

func (m *volumeDiscountSvcImpl) GetVolumeDiscount(ctx context.Context, req *RetrieveVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
	pbReq := &pb.GetVolumeDiscountRequest{
		Id:       req.VolumeDiscountID,
		Includes: resourcekit.FilterIncludes(ctx, volumeDiscountIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetVolumeDiscountResponse, error) {
			return m.coreClient.GetVolumeDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := volumeDiscountFromProto(resp.VolumeDiscount)
	stashVolumeDiscountMeta(meta, resp.VolumeDiscount)
	return &result, nil
}

func (m *volumeDiscountSvcImpl) CreateVolumeDiscount(ctx context.Context, req *CreateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
	tiers := make([]*pb.CreateVolumeDiscountTierInput, len(req.Tiers))
	for i, t := range req.Tiers {
		tiers[i] = &pb.CreateVolumeDiscountTierInput{
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			ParentTierId:       t.ParentTierID.Ptr(),
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
		Includes:         resourcekit.FilterIncludes(ctx, volumeDiscountIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateVolumeDiscountResponse, error) {
			return m.coreClient.CreateVolumeDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := volumeDiscountFromProto(resp.VolumeDiscount)
	stashVolumeDiscountMeta(meta, resp.VolumeDiscount)
	return &result, nil
}

func (m *volumeDiscountSvcImpl) UpdateVolumeDiscount(ctx context.Context, req *UpdateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
	tiers := make([]*pb.UpdateVolumeDiscountTierInput, len(req.Tiers))
	for i, t := range req.Tiers {
		tiers[i] = &pb.UpdateVolumeDiscountTierInput{
			Id:                 t.ID.Ptr(),
			Name:               t.Name.Ptr(),
			DiscountPercentage: t.DiscountPercentage.Ptr(),
			Threshold:          t.Threshold.Ptr(),
			ParentTierId:       t.ParentTierID.Ptr(),
		}
	}

	pbReq := &pb.UpdateVolumeDiscountRequest{
		Id:                req.VolumeDiscountID,
		Name:              req.Name.Ptr(),
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
		Includes:          resourcekit.FilterIncludes(ctx, volumeDiscountIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, volumeDiscountSvcTracer, "service.volume_discounts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateVolumeDiscountResponse, error) {
			return m.coreClient.UpdateVolumeDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := volumeDiscountFromProto(resp.VolumeDiscount)
	stashVolumeDiscountMeta(meta, resp.VolumeDiscount)
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

var volumeDiscountIncludes = []string{"customer_groups", "product_lines", "categories", "attributes", "acceptable_units"}

func volumeDiscountFromProto(d *pb.VolumeDiscountInfo) apiresource.VolumeDiscount {
	if d == nil {
		return apiresource.VolumeDiscount{}
	}

	tiers := make([]apiresource.VolumeDiscountTier, len(d.Tiers))
	for i, t := range d.Tiers {
		tiers[i] = apiresource.VolumeDiscountTier{
			ID:                 t.Id,
			Object:             constants.ObjectTypeVolumeDiscountTier,
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			CreatedAt:          grpcutil.TimestampToTime(t.CreatedAt),
			UpdatedAt:          grpcutil.TimestampToTime(t.UpdatedAt),
		}
	}

	return apiresource.VolumeDiscount{
		ID:        d.Id,
		Object:    constants.ObjectTypeVolumeDiscount,
		Name:      d.Name,
		Tiers:     apiresource.NewList(tiers, apiresource.PageInfo{}),
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func stashVolumeDiscountMeta(meta *resourcekit.LoadMeta, d *pb.VolumeDiscountInfo) {
	if d == nil {
		return
	}

	customerGroups := make([]apiresource.AccountGroup, len(d.CustomerGroups))
	for i, cg := range d.CustomerGroups {
		customerGroups[i] = apiresource.AccountGroup{
			ID:        cg.AccountGroupId,
			Object:    constants.ObjectTypeAccountGroup,
			Name:      cg.Name,
			CreatedAt: grpcutil.TimestampToTime(cg.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(cg.UpdatedAt),
		}
		if cg.CommissionPolicy != nil {
			customerGroups[i].CommissionPolicy = constants.CommissionPolicy(*cg.CommissionPolicy)
		}
		if cg.FreightPolicy != nil {
			customerGroups[i].FreightPolicy = constants.FreightPolicy(*cg.FreightPolicy)
		}
		if cg.Type != nil {
			customerGroups[i].Type = constants.AccountGroupType(*cg.Type)
		}
	}
	meta.Set(constants.ObjectTypeVolumeDiscount, d.Id, "customer_groups",
		apiresource.NewList(customerGroups, apiresource.PageInfo{}))

	productLines := make([]apiresource.ProductLine, len(d.ProductLines))
	for i, pl := range d.ProductLines {
		productLines[i] = apiresource.ProductLine{
			ID:        pl.Id,
			Object:    constants.ObjectTypeProductLine,
			Name:      pl.Name,
			CreatedAt: grpcutil.TimestampToTime(pl.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(pl.UpdatedAt),
		}
		if pl.CommissionPolicy != nil {
			productLines[i].CommissionPolicy = constants.CommissionPolicy(*pl.CommissionPolicy)
		}
		if pl.FreightPolicy != nil {
			productLines[i].FreightPolicy = constants.FreightPolicy(*pl.FreightPolicy)
		}
	}
	meta.Set(constants.ObjectTypeVolumeDiscount, d.Id, "product_lines",
		apiresource.NewList(productLines, apiresource.PageInfo{}))

	categories := make([]apiresource.ItemCategory, len(d.Categories))
	for i, cat := range d.Categories {
		categories[i] = apiresource.ItemCategory{
			ID:        cat.Id,
			Object:    constants.ObjectTypeItemCategory,
			Name:      cat.Name,
			CreatedAt: grpcutil.TimestampToTime(cat.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(cat.UpdatedAt),
		}
		if cat.Type != nil {
			categories[i].Type = constants.ItemCategoryType(*cat.Type)
		}
	}
	meta.Set(constants.ObjectTypeVolumeDiscount, d.Id, "categories",
		apiresource.NewList(categories, apiresource.PageInfo{}))

	attributes := make([]apiresource.Attribute, len(d.Attributes))
	for i, attr := range d.Attributes {
		attributes[i] = apiresource.Attribute{
			ID:        attr.Id,
			Object:    constants.ObjectTypeAttribute,
			Value:     attr.Name,
			CreatedAt: grpcutil.TimestampToTime(attr.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(attr.UpdatedAt),
		}
		if attr.ColorCode != nil {
			attributes[i].ColorCode = constants.Color(*attr.ColorCode)
		}
	}
	meta.Set(constants.ObjectTypeVolumeDiscount, d.Id, "attributes",
		apiresource.NewList(attributes, apiresource.PageInfo{}))

	units := make([]apiresource.Unit, len(d.AcceptableUnits))
	for i, u := range d.AcceptableUnits {
		units[i] = apiresource.Unit{
			ID:                u.Id,
			Object:            constants.ObjectTypeUnit,
			Name:              u.Name,
			Abbreviation:      u.Abbreviation,
			Type:              constants.UnitType(u.Type),
			RatioNumerator:    u.RatioNumerator,
			RatioDenominator:  u.RatioDenominator,
			OffsetNumerator:   u.OffsetNumerator,
			OffsetDenominator: u.OffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(u.CreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(u.UpdatedAt),
		}
	}
	meta.Set(constants.ObjectTypeVolumeDiscount, d.Id, "acceptable_units",
		apiresource.NewList(units, apiresource.PageInfo{}))
}

func volumeDiscountListFromProto(ctx context.Context, resp *pb.ListVolumeDiscountsResponse) *apiresource.List[apiresource.VolumeDiscount] {
	if resp == nil {
		return apiresource.NewList[apiresource.VolumeDiscount](nil, apiresource.PageInfo{})
	}

	meta := resourcekit.GetLoadMeta(ctx)
	discounts := make([]apiresource.VolumeDiscount, len(resp.VolumeDiscounts))
	for i, d := range resp.VolumeDiscounts {
		discounts[i] = volumeDiscountFromProto(d)
		stashVolumeDiscountMeta(meta, d)
	}

	return apiresource.NewList(discounts, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
