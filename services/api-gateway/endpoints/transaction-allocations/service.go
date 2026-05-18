package transactionallocationep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TransactionAllocationSvc interface {
	ListAllocationEntries(ctx context.Context, req *ListAllocationEntriesRequest) (*apiresource.List[apiresource.AllocationEntry], *apierror.APIError)
	UpdateTransactionAllocation(ctx context.Context, req *UpdateTransactionAllocationRequest) (*apiresource.TransactionAllocation, *apierror.APIError)
	DeleteTransactionAllocation(ctx context.Context, req *DeleteTransactionAllocationRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListOpenCredits(ctx context.Context, req *ListOpenCreditsRequest) (*apiresource.List[apiresource.OpenCreditEntry], *apierror.APIError)
}

type TransactionAllocationSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type transactionAllocationSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var transactionAllocationSvcTracer = tracing.GetTracer("api-gateway.endpoints.transaction-allocations.service")

func (c *TransactionAllocationSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("transaction allocation endpoint service: core client is required")
	}
	return nil
}

func NewTransactionAllocationSvc(config *TransactionAllocationSvcConfig) TransactionAllocationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &transactionAllocationSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *transactionAllocationSvcImpl) ListAllocationEntries(ctx context.Context, req *ListAllocationEntriesRequest) (*apiresource.List[apiresource.AllocationEntry], *apierror.APIError) {
	pbReq := &pb.ListAllocationEntriesRequest{
		Cursor:          req.Cursor,
		Limit:           req.Limit,
		Query:           req.Query,
		TransactionType: req.TransactionType,
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

	resp, apiErr := grpcutil.CallRPC(ctx, transactionAllocationSvcTracer, "service.transaction-allocations.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAllocationEntriesResponse, error) {
			return m.coreClient.ListAllocationEntries(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AllocationEntryListPresenter(ctx, resp), nil
}

func (m *transactionAllocationSvcImpl) UpdateTransactionAllocation(ctx context.Context, req *UpdateTransactionAllocationRequest) (*apiresource.TransactionAllocation, *apierror.APIError) {
	pbReq := &pb.UpdateTransactionAllocationRequest{
		Id:     req.AllocationID,
		Amount: req.Amount,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, transactionAllocationSvcTracer, "service.transaction-allocations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateTransactionAllocationResponse, error) {
			return m.coreClient.UpdateTransactionAllocation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := TransactionAllocationPresenter(resp.Allocation)
	return &result, nil
}

func (m *transactionAllocationSvcImpl) DeleteTransactionAllocation(ctx context.Context, req *DeleteTransactionAllocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteTransactionAllocationRequest{
		Id: req.AllocationID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, transactionAllocationSvcTracer, "service.transaction-allocations.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteTransactionAllocationResponse, error) {
			return m.coreClient.DeleteTransactionAllocation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *transactionAllocationSvcImpl) ListOpenCredits(ctx context.Context, req *ListOpenCreditsRequest) (*apiresource.List[apiresource.OpenCreditEntry], *apierror.APIError) {
	pbReq := &pb.ListOpenCreditsRequest{
		CustomerIds: req.CustomerIDs,
		Limit:       req.Limit,
	}
	if req.Cursor != nil {
		pbReq.Cursor = req.Cursor
	}
	if req.Query != nil {
		pbReq.Query = req.Query
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

	resp, apiErr := grpcutil.CallRPC(ctx, transactionAllocationSvcTracer, "service.transaction-allocations.list-open-credits", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListOpenCreditsResponse, error) {
			return m.coreClient.ListOpenCredits(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return OpenCreditListPresenter(ctx, resp), nil
}
