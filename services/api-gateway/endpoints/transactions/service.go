package transactionep

import (
	"context"
	"fmt"
	"strings"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TransactionSvc interface {
	ListTransactions(ctx context.Context, req *ListTransactionsRequest) (*apiresource.List[apiresource.TransactionSummary], *apierror.APIError)
	GetTransaction(ctx context.Context, req *GetTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError)
	CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError)
	UpdateTransaction(ctx context.Context, req *UpdateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError)
	DeleteTransaction(ctx context.Context, req *DeleteTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError)
	ListAccountTransactions(ctx context.Context, req *ListAccountTransactionsRequest) (*apiresource.List[apiresource.TransactionDetail], *apierror.APIError)
	ListTransactionTypes(ctx context.Context, req *ListTransactionTypesRequest) (*apiresource.List[apiresource.TransactionType], *apierror.APIError)
	ListTransactionMethods(ctx context.Context, req *ListTransactionMethodsRequest) (*apiresource.List[apiresource.TransactionMethod], *apierror.APIError)
	ListAdjustmentTypes(ctx context.Context, req *ListAdjustmentTypesRequest) (*apiresource.List[apiresource.AdjustmentType], *apierror.APIError)
}

type TransactionSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type transactionSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var transactionSvcTracer = tracing.GetTracer("api-gateway.endpoints.transactions.service")

func (c *TransactionSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("transaction endpoint service: core client is required")
	}
	return nil
}

func NewTransactionSvc(config *TransactionSvcConfig) TransactionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &transactionSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *transactionSvcImpl) ListTransactions(ctx context.Context, req *ListTransactionsRequest) (*apiresource.List[apiresource.TransactionSummary], *apierror.APIError) {
	pbReq := &pb.ListTransactionsRequest{
		Cursor:              req.Cursor,
		Limit:               req.Limit,
		Query:               req.Query,
		Status:              req.Status,
		TypeCodes:           req.TypeCodes,
		AdjustmentTypeCodes: req.AdjustmentTypeCodes,
		MethodCodes:         req.MethodCodes,
		CustomerIds:         req.CustomerIDs,
		CustomerGroupIds:    req.CustomerGroupIDs,
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

	resp, apiErr := grpcutil.CallRPC(ctx, transactionSvcTracer, "service.transactions.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListTransactionsResponse, error) {
			return m.coreClient.ListTransactions(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return TransactionListPresenter(resp), nil
}

func (m *transactionSvcImpl) GetTransaction(ctx context.Context, req *GetTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
	pbReq := &pb.GetTransactionRequest{
		Id:       req.TransactionID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, transactionSvcTracer, "service.transactions.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetTransactionResponse, error) {
			return m.coreClient.GetTransaction(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := TransactionDetailPresenter(resp.Transaction)
	return &result, nil
}

func (m *transactionSvcImpl) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
	pbReq := &pb.CreateTransactionRequest{
		CustomerId:            req.CustomerID,
		TransactionTypeCode:   req.TransactionTypeCode,
		Amount:                req.Amount,
		TransactionMethodCode: req.TransactionMethodCode,
		AdjustmentTypeCode:    req.AdjustmentTypeCode,
		ResponsibleUserId:     req.ResponsibleUserID,
		Note:                  req.Note,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, transactionSvcTracer, "service.transactions.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateTransactionResponse, error) {
			return m.coreClient.CreateTransaction(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := TransactionDetailPresenter(resp.Transaction)
	return &result, nil
}

func (m *transactionSvcImpl) UpdateTransaction(ctx context.Context, req *UpdateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
	pbReq := &pb.UpdateTransactionRequest{
		Id:                     req.TransactionID,
		Number:                 req.Number,
		Note:                   req.Note,
		Amount:                 req.Amount,
		TransactionMethodCode:  req.TransactionMethodCode,
		AdjustmentTypeCode:     req.AdjustmentTypeCode,
		ResponsibleUserId:      req.ResponsibleUserID,
		ClearResponsibleUser:   req.ClearResponsibleUser,
		ClearTransactionMethod: req.ClearTransactionMethod,
		ClearAdjustmentType:    req.ClearAdjustmentType,
		IsFullyAllocated:       req.IsFullyAllocated,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, transactionSvcTracer, "service.transactions.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateTransactionResponse, error) {
			return m.coreClient.UpdateTransaction(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := TransactionDetailPresenter(resp.Transaction)
	return &result, nil
}

func (m *transactionSvcImpl) DeleteTransaction(ctx context.Context, req *DeleteTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
	pbReq := &pb.DeleteTransactionRequest{
		Id: req.TransactionID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, transactionSvcTracer, "service.transactions.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteTransactionResponse, error) {
			return m.coreClient.DeleteTransaction(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := TransactionDetailPresenter(resp.Transaction)
	return &result, nil
}

func (m *transactionSvcImpl) ListAccountTransactions(ctx context.Context, req *ListAccountTransactionsRequest) (*apiresource.List[apiresource.TransactionDetail], *apierror.APIError) {
	// Default to including child accounts (matches legacy Dashboard behavior).
	includeChildAccounts := req.IncludeChildAccounts == nil || *req.IncludeChildAccounts

	pbReq := &pb.ListAccountTransactionsRequest{
		CustomerAccountId:    req.CustomerAccountID,
		Cursor:               req.Cursor,
		Limit:                req.Limit,
		Query:                req.Query,
		Status:               req.Status,
		Type:                 req.Type,
		IncludeChildAccounts: includeChildAccounts,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, transactionSvcTracer, "service.transactions.list_account", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountTransactionsResponse, error) {
			return m.coreClient.ListAccountTransactions(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AccountTransactionListPresenter(resp), nil
}

// staticTransactionTypes is the hardcoded list of transaction types.
var staticTransactionTypes = []apiresource.TransactionType{
	{
		ID:     "txtp_01seedpayment000000",
		Object: constants.ObjectTypeTransactionType,
		Name:   "Payment",
		Code:   "payment",
	},
	{
		ID:     "txtp_01seedcreditmemo000",
		Object: constants.ObjectTypeTransactionType,
		Name:   "Credit Memo",
		Code:   "credit_memo",
	},
	{
		ID:     "txtp_01seedadjustment000",
		Object: constants.ObjectTypeTransactionType,
		Name:   "Adjustment",
		Code:   "adjustment",
	},
	{
		ID:     "txtp_01seedrebate0000000",
		Object: constants.ObjectTypeTransactionType,
		Name:   "Rebate",
		Code:   "rebate",
	},
}

// staticTransactionMethods is the hardcoded list of transaction methods.
var staticTransactionMethods = []apiresource.TransactionMethod{
	{
		ID:     "txmd_01seedcash00000000",
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Cash",
		Code:   "cash",
	},
	{
		ID:     "txmd_01seedcheck0000000",
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Check",
		Code:   "check",
	},
	{
		ID:     "txmd_01seedcreditcard00",
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Credit Card",
		Code:   "credit_card",
	},
	{
		ID:     "txmd_01seedgiftcard0000",
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Gift Card",
		Code:   "gift_card",
	},
	{
		ID:     "txmd_01seedach000000000",
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Ach",
		Code:   "ach",
	},
}

func (m *transactionSvcImpl) ListTransactionTypes(_ context.Context, req *ListTransactionTypesRequest) (*apiresource.List[apiresource.TransactionType], *apierror.APIError) {
	if req.Cursor != nil && *req.Cursor != "" {
		return nil, apierror.NewValidationError("Invalid pagination cursor.")
	}

	results := staticTransactionTypes
	if req.Query != nil && *req.Query != "" {
		q := strings.ToLower(*req.Query)
		filtered := make([]apiresource.TransactionType, 0, len(staticTransactionTypes))
		for _, t := range staticTransactionTypes {
			if strings.Contains(strings.ToLower(t.Name), q) {
				filtered = append(filtered, t)
			}
		}
		results = filtered
	}
	if req.Limit > 0 && int(req.Limit) < len(results) {
		results = results[:req.Limit]
	}
	return apiresource.NewList(results, apiresource.PageInfo{}), nil
}

func (m *transactionSvcImpl) ListTransactionMethods(_ context.Context, req *ListTransactionMethodsRequest) (*apiresource.List[apiresource.TransactionMethod], *apierror.APIError) {
	if req.Cursor != nil && *req.Cursor != "" {
		return nil, apierror.NewValidationError("Invalid pagination cursor.")
	}

	results := staticTransactionMethods
	if req.Query != nil && *req.Query != "" {
		q := strings.ToLower(*req.Query)
		filtered := make([]apiresource.TransactionMethod, 0, len(staticTransactionMethods))
		for _, t := range staticTransactionMethods {
			if strings.Contains(strings.ToLower(t.Name), q) {
				filtered = append(filtered, t)
			}
		}
		results = filtered
	}
	if req.Limit > 0 && int(req.Limit) < len(results) {
		results = results[:req.Limit]
	}
	return apiresource.NewList(results, apiresource.PageInfo{}), nil
}

func (m *transactionSvcImpl) ListAdjustmentTypes(ctx context.Context, req *ListAdjustmentTypesRequest) (*apiresource.List[apiresource.AdjustmentType], *apierror.APIError) {
	pbReq := &pb.ListAdjustmentTypesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, transactionSvcTracer, "service.adjustment_types.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAdjustmentTypesResponse, error) {
			return m.coreClient.ListAdjustmentTypes(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AdjustmentTypeListPresenter(resp), nil
}
