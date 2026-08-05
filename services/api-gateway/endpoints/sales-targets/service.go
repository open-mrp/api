package salestargetep

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type SalesTargetSvc interface {
	ListSalesTargets(ctx context.Context, req *ListSalesTargetsRequest) (*apiresource.List[apiresource.SalesTarget], *apierror.APIError)
	CreateSalesTarget(ctx context.Context, req *CreateSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError)
	UpsertSalesTarget(ctx context.Context, req *UpsertSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError)
}

type SalesTargetSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type salesTargetSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var salesTargetSvcTracer = tracing.GetTracer("api-gateway.endpoints.sales_targets.service")

func (c *SalesTargetSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sales target endpoint service: core client is required")
	}
	return nil
}

func NewSalesTargetSvc(config *SalesTargetSvcConfig) SalesTargetSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &salesTargetSvcImpl{coreClient: config.CoreClient}
}

func (m *salesTargetSvcImpl) ListSalesTargets(ctx context.Context, req *ListSalesTargetsRequest) (*apiresource.List[apiresource.SalesTarget], *apierror.APIError) {
	if req.Cursor != nil && *req.Cursor != "" {
		return nil, apierror.NewValidationError("Invalid pagination cursor.")
	}

	pbReq := &pb.ListSalesTargetsRequest{
		SalesRepId: req.SalesRepID,
		Limit:      req.Limit,
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesTargetSvcTracer, "service.sales_targets.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesTargetsResponse, error) {
			return m.coreClient.ListSalesTargets(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.SalesTarget](nil, apiresource.PageInfo{}), nil
	}

	targets := make([]apiresource.SalesTarget, len(resp.SalesTargets))
	for i, st := range resp.SalesTargets {
		targets[i] = salesTargetFromProto(st)
	}

	return apiresource.NewList(targets, apiresource.PageInfo{}), nil
}

func (m *salesTargetSvcImpl) CreateSalesTarget(ctx context.Context, req *CreateSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError) {
	pbReq := &pb.CreateSalesTargetRequest{
		SalesRepId:   req.SalesRepID,
		StartDate:    req.StartDate.Format(time.RFC3339),
		EndDate:      req.EndDate.Format(time.RFC3339),
		AmountValue:  req.AmountValue,
		AmountUnitId: req.AmountUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesTargetSvcTracer, "service.sales_targets.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesTargetResponse, error) {
			return m.coreClient.CreateSalesTarget(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesTargetFromProto(resp.SalesTarget)
	return &result, nil
}

func (m *salesTargetSvcImpl) UpsertSalesTarget(ctx context.Context, req *UpsertSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError) {
	pbReq := &pb.UpsertSalesTargetRequest{
		TargetId:     req.TargetID,
		SalesRepId:   req.SalesRepID,
		StartDate:    req.StartDate.Format(time.RFC3339),
		EndDate:      req.EndDate.Format(time.RFC3339),
		AmountValue:  req.AmountValue,
		AmountUnitId: req.AmountUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesTargetSvcTracer, "service.sales_targets.upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpsertSalesTargetResponse, error) {
			return m.coreClient.UpsertSalesTarget(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesTargetFromProto(resp.SalesTarget)
	return &result, nil
}

// parseRFC3339 tolerates an unset timestamp rather than failing the whole read; the caller's own required-field validation is what surfaces a genuinely missing one.
func parseRFC3339(v string) time.Time {
	t, _ := time.Parse(time.RFC3339, v)
	return t
}

func salesTargetFromProto(st *pb.SalesTargetProto) apiresource.SalesTarget {
	if st == nil {
		return apiresource.SalesTarget{}
	}

	startAt, _ := time.Parse(time.RFC3339, st.StartDate)
	endAt, _ := time.Parse(time.RFC3339, st.EndDate)
	createdAt, _ := time.Parse(time.RFC3339, st.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, st.UpdatedAt)

	return apiresource.SalesTarget{
		ID:     st.Id,
		Object: constants.ObjectTypeSalesTarget,
		// A reference, not a stub record: the sales rep id identifies an account user, and a partially-filled AccountUser would serialize with zero timestamps that read as real values.
		SalesRep: apiresource.NewEntity(st.SalesRepId, constants.ObjectTypeAccountUser, nil, nil),
		Amount: &apiresource.Quantity{
			ID:           st.AmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        apiresource.NormalizeQuantityValue(st.AmountValue, st.AmountUnitType),
			DisplayValue: apiresource.FormatDisplayValue(st.AmountValue, st.AmountUnitAbbreviation, st.AmountUnitType),
			Unit: &apiresource.Unit{
				ID:                st.AmountUnitId,
				Object:            constants.ObjectTypeUnit,
				Name:              st.AmountUnitName,
				Abbreviation:      st.AmountUnitAbbreviation,
				Type:              constants.UnitType(st.AmountUnitType),
				RatioNumerator:    st.AmountUnitRatioNumerator,
				RatioDenominator:  st.AmountUnitRatioDenominator,
				OffsetNumerator:   st.AmountUnitOffsetNumerator,
				OffsetDenominator: st.AmountUnitOffsetDenominator,
				CreatedAt:         parseRFC3339(st.AmountUnitCreatedAt),
				UpdatedAt:         parseRFC3339(st.AmountUnitUpdatedAt),
			},
		},
		StartAt:   startAt,
		EndAt:     endAt,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
