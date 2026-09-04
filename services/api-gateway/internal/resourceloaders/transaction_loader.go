package resourceloaders

import (
	"context"

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

var transactionLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.transaction")

// LoadTransactions fetches transactions by ID via GetTransaction and builds expandable
// TransactionDetail references with real header data. There is no batch RPC for transactions, so
// each ID is fetched individually — the same shape as LoadInvoices.
//
// The customer and responsible user are stashed as ids rather than resolved: they are the
// transaction's own expandables, and a caller reaching a transaction through an allocation asked
// for the transaction, not for everything hanging off it.
func LoadTransactions(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, transactionLoaderTracer, "loader.transactions.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetTransactionResponse, error) {
				return coreClient.GetTransaction(ctx, &pb.GetTransactionRequest{Id: id}, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}
		if resp.Transaction == nil {
			continue
		}
		out[resp.Transaction.Id] = transactionReferenceFromProto(meta, resp.Transaction)
	}
	return out, nil
}

func transactionReferenceFromProto(meta *resourcekit.LoadMeta, d *pb.TransactionInfo) *apiresource.TransactionDetail {
	tx := &apiresource.TransactionDetail{
		ID:               d.Id,
		Object:           constants.ObjectTypeTransaction,
		Number:           d.Number,
		Note:             d.Note,
		IsFullyAllocated: d.IsFullyAllocated,
		StripePaymentID:  d.StripePaymentId,
		AllocationCount:  d.AllocationCount,
		Amount: &apiresource.Quantity{
			ID:           d.AmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        d.AmountValue,
			DisplayValue: apiresource.FormatDisplayValue(d.AmountValue, d.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
			// Unit left nil: its id is stashed so `…transaction.amount.unit` resolves the real unit.
		},
		TransactionType: &apiresource.TransactionType{
			ID:     d.TransactionTypeId,
			Object: constants.ObjectTypeTransactionType,
			Name:   d.TransactionTypeName,
			Code:   constants.TransactionType(d.TransactionTypeCode),
		},
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}

	meta.Set(constants.ObjectTypeQuantity, d.AmountId, "unit_id", d.AmountUnitId)

	if d.TransactionMethodId != nil {
		tx.TransactionMethod = &apiresource.TransactionMethod{
			ID:     *d.TransactionMethodId,
			Object: constants.ObjectTypeTransactionMethod,
			Name:   d.GetTransactionMethodName(),
			Code:   constants.TransactionMethod(d.GetTransactionMethodCode()),
		}
	}
	if d.AdjustmentTypeId != nil {
		tx.AdjustmentType = &apiresource.AdjustmentType{
			ID:     *d.AdjustmentTypeId,
			Object: constants.ObjectTypeAdjustmentType,
			Name:   d.GetAdjustmentTypeName(),
			Code:   constants.AdjustmentType(d.GetAdjustmentTypeCode()),
		}
	}
	if d.CustomerId != nil && *d.CustomerId != "" {
		meta.Set(constants.ObjectTypeTransaction, tx.ID, "customer_id", *d.CustomerId)
	}
	if d.ResponsibleUserId != nil && *d.ResponsibleUserId != "" {
		meta.Set(constants.ObjectTypeTransaction, tx.ID, "responsible_user_id", *d.ResponsibleUserId)
	}
	return tx
}
