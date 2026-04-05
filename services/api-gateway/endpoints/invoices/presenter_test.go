package invoiceep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestInvoiceSummaryPresenter(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	ptID := "pytm_01abc"
	ptName := "Net 30"
	ptActive := true
	addrName := "Acme Corp"

	custStatus := "active"
	custCommission := "net"

	info := &pb.InvoiceSummaryInfo{
		Id:                       "inv_01abc",
		Number:                   "INV-0001",
		CustomerId:               "ac_01abc",
		CustomerName:             "Acme Corp",
		CustomerNumber:           "ACME001",
		CustomerStatusCode:       &custStatus,
		CustomerCommissionPolicy: &custCommission,
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

	result := InvoiceSummaryPresenter(info)
	resourcetest.ValidateExpandableStubs(t, "InvoiceSummary", result)
}
