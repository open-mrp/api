package settlementep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SettlementSvc interface {
	ListSettlements(ctx context.Context, req *ListSettlementsRequest) (*apiresource.List[apiresource.SettlementSummary], *apierror.APIError)
	GetSettlement(ctx context.Context, req *RetrieveSettlementRequest) (*apiresource.Settlement, *apierror.APIError)
	CreateSettlement(ctx context.Context, req *CreateSettlementRequest) (*apiresource.Settlement, *apierror.APIError)
	UpdateSettlement(ctx context.Context, req *UpdateSettlementRequest) (*apiresource.Settlement, *apierror.APIError)
	DeleteSettlement(ctx context.Context, req *DeleteSettlementRequest) (*apiresource.Settlement, *apierror.APIError)
}

type SettlementSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type settlementSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var settlementSvcTracer = tracing.GetTracer("api-gateway.endpoints.settlements.service")

func (c *SettlementSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("settlement endpoint service: core client is required")
	}
	return nil
}

func NewSettlementSvc(config *SettlementSvcConfig) SettlementSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &settlementSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *settlementSvcImpl) ListSettlements(ctx context.Context, req *ListSettlementsRequest) (*apiresource.List[apiresource.SettlementSummary], *apierror.APIError) {
	pbReq := &pb.ListSettlementsRequest{
		Cursor:         req.Cursor,
		Limit:          req.Limit,
		Query:          req.Query,
		TransactionIds: req.TransactionIDs,
		InvoiceIds:     req.InvoiceIDs,
	}

	if req.StartDate != nil {
		t, err := grpcutil.ParseDateString(*req.StartDate)
		if err == nil {
			pbReq.StartDate = timestamppb.New(t)
		}
	}
	if req.EndDate != nil {
		t, err := grpcutil.ParseDateString(*req.EndDate)
		if err == nil {
			pbReq.EndDate = timestamppb.New(t)
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, settlementSvcTracer, "service.settlements.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSettlementsResponse, error) {
			return m.coreClient.ListSettlements(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return settlementListFromProto(ctx, resp), nil
}

func (m *settlementSvcImpl) GetSettlement(ctx context.Context, req *RetrieveSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
	pbReq := &pb.GetSettlementRequest{
		Id:       req.SettlementID,
		Includes: resourcekit.FilterIncludes(ctx, settlementIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, settlementSvcTracer, "service.settlements.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSettlementResponse, error) {
			return m.coreClient.GetSettlement(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := settlementFromProto(resp.Settlement)
	stashSettlementMeta(meta, resp.Settlement)
	return &result, nil
}

func (m *settlementSvcImpl) CreateSettlement(ctx context.Context, req *CreateSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
	allocations := make([]*pb.CreateSettlementAllocationParam, len(req.Allocations))
	for i, a := range req.Allocations {
		allocations[i] = &pb.CreateSettlementAllocationParam{
			TransactionId: a.TransactionID,
			InvoiceId:     a.InvoiceID,
			Amount:        a.Amount,
			Note:          a.Note.Ptr(),
		}
	}

	pbReq := &pb.CreateSettlementRequest{
		ResponsibleUserId: req.ResponsibleUserID,
		Allocations:       allocations,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, settlementSvcTracer, "service.settlements.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSettlementResponse, error) {
			return m.coreClient.CreateSettlement(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := settlementFromProto(resp.Settlement)
	stashSettlementMeta(meta, resp.Settlement)
	return &result, nil
}

func (m *settlementSvcImpl) UpdateSettlement(ctx context.Context, req *UpdateSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
	pbReq := &pb.UpdateSettlementRequest{
		Id:                req.SettlementID,
		Number:            req.Number.Ptr(),
		Note:              req.Note.Ptr(),
		ResponsibleUserId: req.ResponsibleUserID.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, settlementSvcTracer, "service.settlements.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSettlementResponse, error) {
			return m.coreClient.UpdateSettlement(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := settlementFromProto(resp.Settlement)
	stashSettlementMeta(meta, resp.Settlement)
	return &result, nil
}

func (m *settlementSvcImpl) DeleteSettlement(ctx context.Context, req *DeleteSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
	pbReq := &pb.DeleteSettlementRequest{
		Id: req.SettlementID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, settlementSvcTracer, "service.settlements.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteSettlementResponse, error) {
			return m.coreClient.DeleteSettlement(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := settlementFromProto(resp.Settlement)
	stashSettlementMeta(meta, resp.Settlement)
	return &result, nil
}

var settlementIncludes = []string{"allocations"}

func settlementFromProto(d *pb.SettlementInfo) apiresource.Settlement {
	if d == nil {
		return apiresource.Settlement{}
	}

	return apiresource.Settlement{
		ID:        d.Id,
		Object:    constants.ObjectTypeSettlement,
		Number:    d.Number,
		Note:      d.Note,
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func stashSettlementMeta(meta *resourcekit.LoadMeta, d *pb.SettlementInfo) {
	if d == nil {
		return
	}

	if d.ResponsibleUserId != nil {
		meta.Set(constants.ObjectTypeSettlement, d.Id, "responsible_user_id", *d.ResponsibleUserId)
	}

	if d.Allocations != nil {
		allocations := make([]apiresource.TransactionAllocation, len(d.Allocations))
		for i, a := range d.Allocations {
			allocations[i] = transactionAllocationFromProto(a)
		}
		meta.Set(constants.ObjectTypeSettlement, d.Id, "allocations",
			apiresource.NewList(allocations, apiresource.PageInfo{}))
	}
}

func transactionAllocationFromProto(a *pb.TransactionAllocationInfo) apiresource.TransactionAllocation {
	if a == nil {
		return apiresource.TransactionAllocation{}
	}

	alloc := apiresource.TransactionAllocation{
		ID:     a.Id,
		Object: constants.ObjectTypeTransactionAllocation,
		Amount: &apiresource.Quantity{
			ID:           a.AmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        a.AmountValue,
			DisplayValue: apiresource.FormatDisplayValue(a.AmountValue, a.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
		},
		Note:      a.Note,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}

	return alloc
}

func settlementSummaryFromProto(d *pb.SettlementSummaryInfo) apiresource.SettlementSummary {
	if d == nil {
		return apiresource.SettlementSummary{}
	}

	return apiresource.SettlementSummary{
		ID:               d.Id,
		Object:           constants.ObjectTypeSettlementSummary,
		Number:           d.Number,
		AllocationCount:  d.AllocationCount,
		TotalPayments:    d.TotalPayments,
		TotalRebates:     d.TotalRebates,
		TotalAdjustments: d.TotalAdjustments,
		TotalCredits:     d.TotalCredits,
		InvoiceNumbers:   d.InvoiceNumbers,
		CustomerNames:    d.CustomerNames,
		CreatedAt:        grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func settlementListFromProto(ctx context.Context, resp *pb.ListSettlementsResponse) *apiresource.List[apiresource.SettlementSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.SettlementSummary](nil, apiresource.PageInfo{})
	}

	settlements := make([]apiresource.SettlementSummary, len(resp.Settlements))
	for i, d := range resp.Settlements {
		settlements[i] = settlementSummaryFromProto(d)
	}

	return apiresource.NewList(settlements, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
