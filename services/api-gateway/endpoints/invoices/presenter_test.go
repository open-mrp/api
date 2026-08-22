package invoiceep

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/api-gateway/pkg/resource/resourcetest"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func sampleInvoiceInfo() *pb.InvoiceInfo {
	now := timestamppb.Now()
	ptID := "pytm_01abc"
	ptName := "Net 30"
	ptActive := true
	addrName := "Acme Corp"
	custStatus := "active"
	custCommission := "net"

	return &pb.InvoiceInfo{
		Id:                       "inv_01abc",
		Number:                   "INV-0001",
		CustomerId:               "ac_01abc",
		CustomerName:             "Acme Corp",
		CustomerNumber:           "ACME001",
		CustomerStatusCode:       &custStatus,
		CustomerCommissionPolicy: &custCommission,
		CustomerIsEdiEnabled:     true,
		OrderId:                  "so_01abc",
		OrderNumber:              "SO-0001",
		LineCount:                1,
		BillingAddressId:         "addr_01abc",
		BillingAddressName:       &addrName,
		PriorityCode:             "normal",
		IsPaidInFull:             true,
		IsEdiSent:                true,
		HasBeenSent:              true,
		TotalInvoiced:            "100.00",
		PaymentTermId:            &ptID,
		PaymentTermName:          &ptName,
		PaymentTermIsActive:      &ptActive,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
}

func TestInvoicePresenter(t *testing.T) {
	t.Parallel()

	result := invoiceFromProto(context.Background(), sampleInvoiceInfo())
	resourcetest.ValidateExpandableStubs(t, "Invoice", result)
}

// Guards the collapse: the scalars that only the list projection used to carry must survive the
// single presenter that list, retrieve and update now share.
func TestInvoicePresenterCarriesListScalars(t *testing.T) {
	t.Parallel()

	result := invoiceFromProto(context.Background(), sampleInvoiceInfo())

	if result.LineCount != 1 {
		t.Errorf("LineCount = %d, want 1", result.LineCount)
	}
	if result.TotalInvoiced != "100.00" {
		t.Errorf("TotalInvoiced = %q, want %q", result.TotalInvoiced, "100.00")
	}
	if result.PriorityCode != constants.PriorityCodeNormal {
		t.Errorf("PriorityCode = %q, want %q", result.PriorityCode, constants.PriorityCodeNormal)
	}
	if !result.CustomerIsEdiEnabled {
		t.Error("CustomerIsEdiEnabled = false, want true")
	}
}

// Guards the bug this collapse fixes: an overpaid invoice must not report `paid` on the list or
// update paths just because they used to drop is_over_paid.
func TestInvoicePresenterReportsOverpaid(t *testing.T) {
	t.Parallel()

	info := sampleInvoiceInfo()
	info.IsOverPaid = true

	result := invoiceFromProto(context.Background(), info)

	if result.PaymentStatus != constants.InvoicePaymentStatusOverpaid {
		t.Errorf("PaymentStatus = %q, want %q", result.PaymentStatus, constants.InvoicePaymentStatusOverpaid)
	}
}
