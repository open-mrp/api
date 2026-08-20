package transactionallocationep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
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
	// CoreClient (required) is the core-service gRPC client.
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
		t, err := grpcutil.ParseEndDateString(*req.EndDate)
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

	return allocationEntryListFromProto(ctx, resp), nil
}

func (m *transactionAllocationSvcImpl) UpdateTransactionAllocation(ctx context.Context, req *UpdateTransactionAllocationRequest) (*apiresource.TransactionAllocation, *apierror.APIError) {
	pbReq := &pb.UpdateTransactionAllocationRequest{
		Id:     req.AllocationID,
		Amount: req.Amount.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, transactionAllocationSvcTracer, "service.transaction-allocations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateTransactionAllocationResponse, error) {
			return m.coreClient.UpdateTransactionAllocation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := transactionAllocationFromProto(resp.Allocation)
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
		t, err := grpcutil.ParseEndDateString(*req.EndDate)
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

	return openCreditListFromProto(ctx, resp), nil
}

func allocationEntryFromProto(d *pb.AllocationEntryInfo) apiresource.AllocationEntry {
	if d == nil {
		return apiresource.AllocationEntry{}
	}

	return apiresource.AllocationEntry{
		ID:            d.Id,
		Object:        constants.ObjectTypeAllocationEntry,
		Amount:        d.AmountValue,
		DisplayAmount: apiresource.FormatDisplayValue(d.AmountValue, d.AmountUnitAbbr, string(constants.UnitTypeCurrency)),
		// pb.AllocationEntryInfo does not carry a customer id, so ID stays null.
		Customer: &apiresource.AllocationCustomer{
			Object: constants.ObjectTypeAllocationCustomer,
			Name:   d.CustomerName,
			Number: d.CustomerNumber,
		},
		Transaction: &apiresource.AllocationTransaction{
			ID:             d.TransactionId,
			Object:         constants.ObjectTypeTransaction,
			Type:           d.TransactionType,
			Method:         d.TransactionMethod,
			AdjustmentType: d.AdjustmentType,
		},
		Invoice: &apiresource.AllocationInvoice{
			ID:     d.InvoiceId,
			Object: constants.ObjectTypeInvoiceSummary,
			Number: d.InvoiceNumber,
		},
		Note:      d.Note,
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
	}
}

func allocationEntryListFromProto(ctx context.Context, resp *pb.ListAllocationEntriesResponse) *apiresource.List[apiresource.AllocationEntry] {
	if resp == nil {
		return apiresource.NewList[apiresource.AllocationEntry](nil, apiresource.PageInfo{})
	}

	entries := make([]apiresource.AllocationEntry, len(resp.Entries))
	for i, d := range resp.Entries {
		entries[i] = allocationEntryFromProto(d)
	}

	return apiresource.NewList(entries, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
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
			Unit: &apiresource.Unit{
				ID:     a.AmountUnitId,
				Object: constants.ObjectTypeUnit,
			},
		},
		Note: a.Note,
		Transaction: &apiresource.TransactionDetail{
			ID:     a.TransactionId,
			Object: constants.ObjectTypeTransaction,
		},
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}

	if a.InvoiceId != nil && *a.InvoiceId != "" {
		invoiceNumber := ""
		if a.InvoiceNumber != nil {
			invoiceNumber = *a.InvoiceNumber
		}
		alloc.Invoice = &apiresource.AllocationInvoice{
			ID:     *a.InvoiceId,
			Object: constants.ObjectTypeInvoiceSummary,
			Number: invoiceNumber,
		}
	}

	return alloc
}

func openCreditEntryFromProto(d *pb.OpenCreditEntryInfo) apiresource.OpenCreditEntry {
	if d == nil {
		return apiresource.OpenCreditEntry{}
	}

	allocations := make([]apiresource.InvoiceAllocationEntry, len(d.InvoiceAllocations))
	for i, a := range d.InvoiceAllocations {
		allocations[i] = apiresource.InvoiceAllocationEntry{
			Object:        constants.ObjectTypeInvoiceAllocationEntry,
			InvoiceNumber: a.InvoiceNumber,
			Amount:        a.Amount,
		}
	}

	return apiresource.OpenCreditEntry{
		ID:              d.Id,
		Object:          constants.ObjectTypeOpenCreditEntry,
		Number:          d.Number,
		OriginalAmount:  d.OriginalAmount,
		AllocatedAmount: d.AllocatedAmount,
		LeftoverAmount:  d.LeftoverAmount,
		Customer: &apiresource.AllocationCustomer{
			ID:     &d.CustomerId,
			Object: constants.ObjectTypeAllocationCustomer,
			Name:   d.CustomerName,
			Number: d.CustomerNumber,
		},
		TransactionType:     d.TransactionType,
		TransactionMethod:   d.TransactionMethod,
		AdjustmentType:      d.AdjustmentType,
		ResponsibleUserName: d.ResponsibleUserName,
		Note:                d.Note,
		StripePaymentID:     d.StripePaymentId,
		InvoiceAllocations:  apiresource.NewList(allocations, apiresource.PageInfo{}),
		CreatedAt:           grpcutil.TimestampToTime(d.CreatedAt),
	}
}

func openCreditListFromProto(ctx context.Context, resp *pb.ListOpenCreditsResponse) *apiresource.List[apiresource.OpenCreditEntry] {
	if resp == nil {
		return apiresource.NewList[apiresource.OpenCreditEntry](nil, apiresource.PageInfo{})
	}

	entries := make([]apiresource.OpenCreditEntry, len(resp.Entries))
	for i, d := range resp.Entries {
		entries[i] = openCreditEntryFromProto(d)
	}

	var pi apiresource.PageInfo
	if resp.PageInfo != nil {
		pi = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}

	return apiresource.NewList(entries, pi)
}
