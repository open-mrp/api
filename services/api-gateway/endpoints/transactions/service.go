package transactionep

import (
	"context"
	"fmt"
	"strings"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TransactionSvc interface {
	ListTransactions(ctx context.Context, req *ListTransactionsRequest) (*apiresource.List[apiresource.TransactionSummary], *apierror.APIError)
	GetTransaction(ctx context.Context, req *RetrieveTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError)
	CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError)
	UpdateTransaction(ctx context.Context, req *UpdateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError)
	DeleteTransaction(ctx context.Context, req *DeleteTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError)
	ListAccountTransactions(ctx context.Context, req *ListAccountTransactionsRequest) (*apiresource.List[apiresource.TransactionDetail], *apierror.APIError)
	ListTransactionTypes(ctx context.Context, req *ListTransactionTypesRequest) (*apiresource.List[apiresource.TransactionType], *apierror.APIError)
	ListTransactionMethods(ctx context.Context, req *ListTransactionMethodsRequest) (*apiresource.List[apiresource.TransactionMethod], *apierror.APIError)
	ListAdjustmentTypes(ctx context.Context, req *ListAdjustmentTypesRequest) (*apiresource.List[apiresource.AdjustmentType], *apierror.APIError)
}

type TransactionSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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

	return TransactionListPresenter(ctx, resp), nil
}

func (m *transactionSvcImpl) GetTransaction(ctx context.Context, req *RetrieveTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
	pbReq := &pb.GetTransactionRequest{
		Id:       req.TransactionID,
		Includes: resourcekit.FilterIncludes(ctx, "allocations"),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, transactionSvcTracer, "service.transactions.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetTransactionResponse, error) {
			return m.coreClient.GetTransaction(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	d := resp.Transaction
	if d == nil {
		return nil, apierror.NewResourceNotFoundError("Transaction not found.")
	}

	createdAt := grpcutil.TimestampToTime(d.CreatedAt)
	updatedAt := grpcutil.TimestampToTime(d.UpdatedAt)

	tx := &apiresource.TransactionDetail{
		ID:               d.Id,
		Object:           constants.ObjectTypeTransaction,
		Number:           d.Number,
		IsFullyAllocated: d.IsFullyAllocated,
		StripePaymentID:  d.StripePaymentId,
		AllocationCount:  d.AllocationCount,
		Note:             d.Note,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	tx.Amount = &apiresource.Quantity{
		ID:           d.AmountId,
		Object:       constants.ObjectTypeQuantity,
		Value:        d.AmountValue,
		DisplayValue: apiresource.FormatDisplayValue(d.AmountValue, d.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
		// Unit left nil: expandable, loaded with real data via ?include=; never fabricated.
	}

	tx.TransactionType = &apiresource.TransactionType{
		ID:     d.TransactionTypeId,
		Object: constants.ObjectTypeTransactionType,
		Name:   d.TransactionTypeName,
		Code:   constants.TransactionType(d.TransactionTypeCode),
	}

	if d.TransactionMethodId != nil {
		tx.TransactionMethod = &apiresource.TransactionMethod{
			ID:     *d.TransactionMethodId,
			Object: constants.ObjectTypeTransactionMethod,
		}
		if d.TransactionMethodName != nil {
			tx.TransactionMethod.Name = *d.TransactionMethodName
		}
		if d.TransactionMethodCode != nil {
			tx.TransactionMethod.Code = constants.TransactionMethod(*d.TransactionMethodCode)
		}
	}

	if d.AdjustmentTypeId != nil {
		tx.AdjustmentType = &apiresource.AdjustmentType{
			ID:        *d.AdjustmentTypeId,
			Object:    constants.ObjectTypeAdjustmentType,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		if d.AdjustmentTypeName != nil {
			tx.AdjustmentType.Name = *d.AdjustmentTypeName
		}
		if d.AdjustmentTypeCode != nil {
			tx.AdjustmentType.Code = constants.AdjustmentType(*d.AdjustmentTypeCode)
		}
	}

	meta := resourcekit.GetLoadMeta(ctx)

	// customer is an expandable reference: stash the FK id so LoadCustomers
	// fetches the real Customer on ?include=customer. Never fabricate.
	if d.CustomerId != nil && *d.CustomerId != "" {
		meta.Set(constants.ObjectTypeTransaction, tx.ID, "customer_id", *d.CustomerId)
	}

	if d.ResponsibleUserId != nil {
		user := &apiresource.AccountUser{
			ID:        *d.ResponsibleUserId,
			Object:    constants.ObjectTypeAccountUser,
			Name:      d.ResponsibleUserName,
			CreatedAt: grpcutil.TimestampToTime(d.ResponsibleUserCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(d.ResponsibleUserUpdatedAt),
		}
		if d.ResponsibleUserStatus != nil {
			user.Status = constants.AccountUserStatus(*d.ResponsibleUserStatus)
		} else {
			user.Status = constants.AccountUserStatusActive
		}
		meta.Set(constants.ObjectTypeTransaction, tx.ID, "responsible_user", user)
	}

	if d.Allocations != nil {
		allocations := make([]apiresource.TransactionAllocation, len(d.Allocations))
		for i, a := range d.Allocations {
			allocations[i] = TransactionAllocationPresenter(a)
		}
		meta.Set(constants.ObjectTypeTransaction, tx.ID, "allocations",
			apiresource.NewList(allocations, apiresource.PageInfo{}))
	}

	return tx, nil
}

func (m *transactionSvcImpl) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
	pbReq := &pb.CreateTransactionRequest{
		CustomerId:            req.CustomerID,
		TransactionTypeCode:   req.TransactionTypeCode,
		Amount:                req.Amount,
		TransactionMethodCode: req.TransactionMethodCode.Ptr(),
		AdjustmentTypeCode:    req.AdjustmentTypeCode.Ptr(),
		ResponsibleUserId:     req.ResponsibleUserID.Ptr(),
		Note:                  req.Note.Ptr(),
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
		Number:                 req.Number.Ptr(),
		Note:                   req.Note.Ptr(),
		Amount:                 req.Amount.Ptr(),
		TransactionMethodCode:  req.TransactionMethodCode.Ptr(),
		AdjustmentTypeCode:     req.AdjustmentTypeCode.Ptr(),
		ResponsibleUserId:      req.ResponsibleUserID.Ptr(),
		ClearResponsibleUser:   req.ClearResponsibleUser,
		ClearTransactionMethod: req.ClearTransactionMethod,
		ClearAdjustmentType:    req.ClearAdjustmentType,
		IsFullyAllocated:       req.IsFullyAllocated.Ptr(),
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

	return AccountTransactionListPresenter(ctx, resp), nil
}

// seedID composes a well-known seeded-row ID from a shared/id prefix constant
// so the prefix cannot drift from shared/id. The suffixes match the rows in
// shared/db/seed/0001_static_types.sql.
func seedID(prefix id.IDPrefix, suffix string) string {
	return string(prefix) + "_" + suffix
}

// staticTransactionTypes is the hardcoded list of transaction types.
var staticTransactionTypes = []apiresource.TransactionType{
	{
		ID:     seedID(id.TransactionTypeIDPrefix, "01seedpayment000000"),
		Object: constants.ObjectTypeTransactionType,
		Name:   "Payment",
		Code:   "payment",
	},
	{
		ID:     seedID(id.TransactionTypeIDPrefix, "01seedcreditmemo000"),
		Object: constants.ObjectTypeTransactionType,
		Name:   "Credit Memo",
		Code:   "credit_memo",
	},
	{
		ID:     seedID(id.TransactionTypeIDPrefix, "01seedadjustment000"),
		Object: constants.ObjectTypeTransactionType,
		Name:   "Adjustment",
		Code:   "adjustment",
	},
	{
		ID:     seedID(id.TransactionTypeIDPrefix, "01seedrebate0000000"),
		Object: constants.ObjectTypeTransactionType,
		Name:   "Rebate",
		Code:   "rebate",
	},
}

// staticTransactionMethods is the hardcoded list of transaction methods.
var staticTransactionMethods = []apiresource.TransactionMethod{
	{
		ID:     seedID(id.TransactionMethodIDPrefix, "01seedcash00000000"),
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Cash",
		Code:   "cash",
	},
	{
		ID:     seedID(id.TransactionMethodIDPrefix, "01seedcheck0000000"),
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Check",
		Code:   "check",
	},
	{
		ID:     seedID(id.TransactionMethodIDPrefix, "01seedcreditcard00"),
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Credit Card",
		Code:   "credit_card",
	},
	{
		ID:     seedID(id.TransactionMethodIDPrefix, "01seedgiftcard0000"),
		Object: constants.ObjectTypeTransactionMethod,
		Name:   "Gift Card",
		Code:   "gift_card",
	},
	{
		ID:     seedID(id.TransactionMethodIDPrefix, "01seedach000000000"),
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

	if resp == nil {
		return apiresource.NewList[apiresource.AdjustmentType](nil, apiresource.PageInfo{}), nil
	}

	adjustmentTypes := make([]apiresource.AdjustmentType, len(resp.AdjustmentTypes))
	for i, at := range resp.AdjustmentTypes {
		adjustmentTypes[i] = apiresource.AdjustmentType{
			ID:        at.Id,
			Object:    constants.ObjectTypeAdjustmentType,
			Name:      at.Name,
			Code:      constants.AdjustmentType(at.Code),
			CreatedAt: grpcutil.TimestampToTime(at.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(at.UpdatedAt),
		}
	}

	return apiresource.NewList(adjustmentTypes, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}
