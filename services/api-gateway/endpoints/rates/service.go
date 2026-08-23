package rateep

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
)

type RateSvc interface {
	UpdateRate(ctx context.Context, req *UpdateRateRequest) (*apiresource.Rate, *apierror.APIError)
}

type RateSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type rateSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var rateSvcTracer = tracing.GetTracer("api-gateway.endpoints.rates.service")

func (c *RateSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("rate endpoint service: core client is required")
	}
	return nil
}

func NewRateSvc(config *RateSvcConfig) RateSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &rateSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *rateSvcImpl) UpdateRate(ctx context.Context, req *UpdateRateRequest) (*apiresource.Rate, *apierror.APIError) {
	pbReq := &pb.UpdateRateRequest{
		Id:                req.RateID,
		Value:             req.Value.Ptr(),
		NumeratorUnitId:   req.NumeratorUnitID.Ptr(),
		DenominatorUnitId: req.DenominatorUnitID.Ptr(),
		ObjectId:          req.ObjectID.Ptr(),
		ObjectType:        req.ObjectType.Ptr().StringPtr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, rateSvcTracer, "service.rates.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateRateResponse, error) {
			return m.coreClient.UpdateRate(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := rateFromProto(resp.Rate)
	stashRateMeta(meta, resp.Rate)
	return &result, nil
}

func rateFromProto(r *pb.RateInfo) apiresource.Rate {
	if r == nil {
		return apiresource.Rate{}
	}
	normalizedValue := apiresource.NormalizeRateValue(r.Value)

	return apiresource.Rate{
		ID:           r.Id,
		Object:       constants.ObjectTypeRate,
		Value:        normalizedValue,
		DisplayValue: apiresource.FormatRateDisplayValue(normalizedValue, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.DenominatorUnitAbbreviation),
		CreatedAt:    grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(r.UpdatedAt),
	}
}

func stashRateMeta(meta *resourcekit.LoadMeta, r *pb.RateInfo) {
	if r == nil {
		return
	}
	meta.Set(constants.ObjectTypeRate, r.Id, "numerator_unit", &apiresource.Unit{
		ID:                r.NumeratorUnitId,
		Object:            constants.ObjectTypeUnit,
		Name:              r.NumeratorUnitName,
		Abbreviation:      r.NumeratorUnitAbbreviation,
		Type:              constants.UnitType(r.NumeratorUnitType),
		RatioNumerator:    r.NumeratorUnitRatioNumerator,
		RatioDenominator:  r.NumeratorUnitRatioDenominator,
		OffsetNumerator:   r.NumeratorUnitOffsetNumerator,
		OffsetDenominator: r.NumeratorUnitOffsetDenominator,
		CreatedAt:         grpcutil.TimestampToTime(r.NumeratorUnitCreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(r.NumeratorUnitUpdatedAt),
	})
	meta.Set(constants.ObjectTypeRate, r.Id, "denominator_unit", &apiresource.Unit{
		ID:                r.DenominatorUnitId,
		Object:            constants.ObjectTypeUnit,
		Name:              r.DenominatorUnitName,
		Abbreviation:      r.DenominatorUnitAbbreviation,
		Type:              constants.UnitType(r.DenominatorUnitType),
		RatioNumerator:    r.DenominatorUnitRatioNumerator,
		RatioDenominator:  r.DenominatorUnitRatioDenominator,
		OffsetNumerator:   r.DenominatorUnitOffsetNumerator,
		OffsetDenominator: r.DenominatorUnitOffsetDenominator,
		CreatedAt:         grpcutil.TimestampToTime(r.DenominatorUnitCreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(r.DenominatorUnitUpdatedAt),
	})
}
