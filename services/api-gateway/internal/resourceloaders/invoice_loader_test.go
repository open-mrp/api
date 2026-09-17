package resourceloaders

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
)

// stubInvoiceClient answers GetInvoice from a fixed set of ids and returns not-found for everything
// else. Only GetInvoice is exercised; the embedded interface panics if anything else is called.
type stubInvoiceClient struct {
	pb.CoreServiceClient
	known map[string]*pb.InvoiceInfo
}

func (s stubInvoiceClient) GetInvoice(_ context.Context, req *pb.GetInvoiceRequest, _ ...grpc.CallOption) (*pb.GetInvoiceResponse, error) {
	if invoice, ok := s.known[req.Id]; ok {
		return &pb.GetInvoiceResponse{Invoice: invoice}, nil
	}
	return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewResourceNotFoundError("Resource not found."))
}

// Voiding a shipment deletes its invoice, so a shipments page read moments earlier can name an
// invoice that no longer exists. That one row's related.invoice goes null; the page must not 404.
func TestLoadInvoices_MissingInvoiceIsOmittedNotFatal(t *testing.T) {
	original := coreClient
	t.Cleanup(func() { coreClient = original })
	coreClient = stubInvoiceClient{known: map[string]*pb.InvoiceInfo{
		"iv_alive": {Id: "iv_alive", Number: "1001"},
	}}

	out, apiErr := LoadInvoices(context.Background(), []string{"iv_alive", "iv_deleted"})
	if apiErr != nil {
		t.Fatalf("LoadInvoices returned %v, want the surviving invoice and no error", apiErr)
	}
	if _, ok := out["iv_alive"]; !ok {
		t.Error("the invoice that still exists must be present")
	}
	if _, ok := out["iv_deleted"]; ok {
		t.Error("the deleted invoice must be absent rather than expanded")
	}
}
