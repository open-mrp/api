package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var invoiceLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.invoice")

// LoadInvoices fetches invoices by ID via GetInvoice and builds expandable
// Invoice references with real header data. There is no batch RPC for invoices,
// so each ID is fetched individually. Nested sub-resources (lines, allocations,
// customer, …) are their own expandable relations and are not populated here.
func LoadInvoices(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, invoiceLoaderTracer, "loader.invoices.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetInvoiceResponse, error) {
				return coreClient.GetInvoice(ctx, &pb.GetInvoiceRequest{Id: id}, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}
		if resp.Invoice == nil {
			continue
		}
		out[resp.Invoice.Id] = invoiceReferenceFromProto(resp.Invoice)
	}
	return out, nil
}

func invoiceReferenceFromProto(d *pb.InvoiceInfo) *apiresource.Invoice {
	paymentStatus := constants.InvoicePaymentStatusUnpaid
	switch {
	case d.IsOverPaid:
		paymentStatus = constants.InvoicePaymentStatusOverpaid
	case d.IsPaidInFull:
		paymentStatus = constants.InvoicePaymentStatusPaid
	}
	return &apiresource.Invoice{
		ID:                   d.Id,
		Object:               constants.ObjectTypeInvoice,
		Number:               d.Number,
		Note:                 d.Note,
		PaymentStatus:        paymentStatus,
		IsEdiSent:            d.IsEdiSent,
		HasBeenSent:          d.HasBeenSent,
		AcceptsInvoiceEmails: d.AcceptsInvoiceEmails,
		CreatedAt:            grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(d.UpdatedAt),
	}
}
