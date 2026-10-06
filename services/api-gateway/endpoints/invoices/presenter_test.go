package invoiceep

import (
	"context"
	"testing"

	// Registers the invoice definitions whose sub-fields these tests resolve.
	_ "github.com/open-mrp/api/services/api-gateway/internal/resourceregistry"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resource/resourcetest"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// An overpaid invoice reports `overpaid` whether or not it is marked paid, so the mark rides alongside.
func TestInvoicePresenterCarriesPaidMarkBesideOverpaid(t *testing.T) {
	t.Parallel()

	for _, marked := range []bool{true, false} {
		info := sampleInvoiceInfo()
		info.IsOverPaid = true
		info.IsPaidInFull = marked

		result := invoiceFromProto(context.Background(), info)

		assert.Equal(t, constants.InvoicePaymentStatusOverpaid, result.PaymentStatus)
		assert.Equal(t, marked, result.IsPaidInFull)
	}
}

func resolveInvoiceIncludes(t *testing.T, ctx context.Context, inv *apiresource.Invoice, keys ...string) {
	t.Helper()
	apiErr := resourcekit.ResolveIncludes(ctx, []any{inv}, constants.ObjectTypeInvoice, resourcekit.ParseIncludeTree(keys))
	require.Nil(t, apiErr)
}

// The billing address and payment term come in full with the invoice, so expanding them calls no
// loader (and needs none of the address or payment-term permissions a loader would check).
func TestInvoicePresenterEmbedsBillingAddressAndPaymentTerm(t *testing.T) {
	t.Parallel()

	street, phone := "1 Main St", "555-0100"
	info := sampleInvoiceInfo()
	info.BillingAddress = &pb.AddressInfo{
		Id: "addr_01abc", Name: "Acme Corp", Phone: &phone, IsDropShip: true,
		Geolocation: &pb.GeolocationInfo{Id: "geo_01abc", StreetLine_1: &street, Country: "US"},
		CreatedAt:   timestamppb.Now(), UpdatedAt: timestamppb.Now(),
	}
	info.PaymentTerm = &pb.PaymentTermInfo{Id: "pytm_01abc", Name: "Net 30", Status: "active", CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now()}

	ctx := resourcekit.WithLoadMeta(context.Background())
	inv := invoiceFromProto(ctx, info)
	assert.Nil(t, inv.BillingAddress, "expandable fields stay null until requested")
	assert.Nil(t, inv.PaymentTerm, "expandable fields stay null until requested")

	resolveInvoiceIncludes(t, ctx, &inv, "billing_address", "payment_term")

	require.NotNil(t, inv.BillingAddress)
	assert.Equal(t, "addr_01abc", inv.BillingAddress.ID)
	assert.Equal(t, constants.AddressTypeDropShip, inv.BillingAddress.Type)
	assert.Equal(t, &phone, inv.BillingAddress.Phone)
	require.NotNil(t, inv.BillingAddress.Geolocation)
	assert.Equal(t, "geo_01abc", inv.BillingAddress.Geolocation.ID)
	assert.Equal(t, &street, inv.BillingAddress.Geolocation.StreetLine1)

	require.NotNil(t, inv.PaymentTerm)
	assert.Equal(t, "pytm_01abc", inv.PaymentTerm.ID)
	assert.Equal(t, "Net 30", inv.PaymentTerm.Name)
	assert.Equal(t, constants.PaymentTermStatusActive, inv.PaymentTerm.Status)
	assert.Nil(t, inv.PaymentTerm.Owner, "the owner is its own include")
}

// An order with no payment term leaves the include null rather than inventing one.
func TestInvoicePresenterLeavesMissingPaymentTermNull(t *testing.T) {
	t.Parallel()

	info := sampleInvoiceInfo()
	info.PaymentTerm = nil

	ctx := resourcekit.WithLoadMeta(context.Background())
	inv := invoiceFromProto(ctx, info)
	resolveInvoiceIncludes(t, ctx, &inv, "payment_term")

	assert.Nil(t, inv.PaymentTerm)
}

// The line's order line carries its number and the description as sold, not only its SKU.
func TestInvoicePresenterOrderLineCarriesNumberAndDescription(t *testing.T) {
	t.Parallel()

	number, description, sku := int32(3), "Crew sock, black", "SCK-001"
	info := sampleInvoiceInfo()
	info.Lines = []*pb.InvoiceLineInfo{{
		Id: "ivln_01abc", OrderLineId: "sol_01abc", OrderLineItemSku: &sku,
		OrderLineItemNumber: &number, OrderLineDescription: &description,
		CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now(),
	}}

	ctx := resourcekit.WithLoadMeta(context.Background())
	inv := invoiceFromProto(ctx, info)
	resolveInvoiceIncludes(t, ctx, &inv, "lines", "lines.order_line")

	require.NotNil(t, inv.Lines)
	require.Len(t, inv.Lines.Data, 1)
	orderLine := inv.Lines.Data[0].OrderLine
	require.NotNil(t, orderLine)
	assert.Equal(t, int32(3), orderLine.LineItemNumber)
	assert.Equal(t, &description, orderLine.ProductDescription)
	assert.Equal(t, sku, orderLine.ProductSKU)
}

// Each allocation names the settlement that recorded it, and none when it was recorded outside one.
func TestInvoicePresenterAllocationCarriesSettlement(t *testing.T) {
	t.Parallel()

	settlementID, settlementNumber := "sl_01abc", "42"
	info := sampleInvoiceInfo()
	info.Allocations = []*pb.InvoiceAllocationInfo{
		{Id: "tral_settled", TransactionId: "tr_1", AmountId: "qt_1", AmountValue: "10", SettlementId: &settlementID, SettlementNumber: &settlementNumber},
		{Id: "tral_loose", TransactionId: "tr_2", AmountId: "qt_2", AmountValue: "5"},
	}

	ctx := resourcekit.WithLoadMeta(context.Background())
	inv := invoiceFromProto(ctx, info)
	resolveInvoiceIncludes(t, ctx, &inv, "allocations")

	require.NotNil(t, inv.Allocations)
	require.Len(t, inv.Allocations.Data, 2)
	settled := inv.Allocations.Data[0].Settlement
	require.NotNil(t, settled)
	assert.Equal(t, apiresource.AllocationSettlement{ID: settlementID, Object: constants.ObjectTypeSettlement, Number: settlementNumber}, *settled)
	assert.Nil(t, inv.Allocations.Data[1].Settlement)
}

// The settle list offers the order's billing address the same way, from the row core already read.
func TestInvoiceForPaymentPresenterEmbedsBillingAddress(t *testing.T) {
	t.Parallel()

	ctx := resourcekit.WithLoadMeta(context.Background())
	inv := invoiceForPaymentFromProto(ctx, &pb.InvoiceForPaymentInfo{
		Id: "inv_01abc", Number: "1001", CustomerId: "ac_01abc", InvoiceTotal: "10.00",
		BillingAddress: &pb.AddressInfo{Id: "addr_01abc", Name: "HQ", Geolocation: &pb.GeolocationInfo{Id: "geo_01abc", Country: "US"}},
	})
	assert.Nil(t, inv.BillingAddress, "expandable fields stay null until requested")

	apiErr := resourcekit.ResolveIncludes(ctx, []any{&inv}, constants.ObjectTypeInvoiceForPayment, resourcekit.ParseIncludeTree([]string{"billing_address"}))
	require.Nil(t, apiErr)

	require.NotNil(t, inv.BillingAddress)
	assert.Equal(t, "addr_01abc", inv.BillingAddress.ID)
	assert.Equal(t, "HQ", inv.BillingAddress.Name)
}
