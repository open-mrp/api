package salesorderep

import (
	"context"
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSalesOrderPresenter(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	carrierID := "cr_01abc"
	carrierName := "FedEx"
	carrierPortal := true
	slID := "crop_01abc"
	slName := "FedEx Ground"
	slPortal := true
	ptID := "pytm_01abc"
	ptName := "Net 30"
	ptActive := true
	stID := "shtm_01abc"
	stName := "Prepaid"
	stType := "carrier_rate_freight"
	priorityID := "pi_01abc"

	slToken := "fedex_ground"
	custStatus := "active"
	custCommission := "net"

	info := &pb.SalesOrderInfo{
		Id:                          "so_01abc",
		Number:                      "SO-0001",
		CustomerId:                  "ac_01abc",
		CustomerName:                "Acme Corp",
		CustomerNumber:              "ACME001",
		CustomerStatusCode:          &custStatus,
		CustomerCommissionPolicy:    &custCommission,
		StatusCode:                  "open",
		StatusName:                  "Open",
		TypeCode:                    "standard",
		TypeName:                    "Standard",
		PriorityCode:                "normal",
		PriorityName:                "Normal",
		PriorityId:                  &priorityID,
		IsAcknowledgmentSent:        true,
		CarrierId:                   &carrierID,
		CarrierName:                 &carrierName,
		CarrierIsPortalEnabled:      &carrierPortal,
		CarrierCreatedAt:            now,
		CarrierUpdatedAt:            now,
		ServiceLevelId:              &slID,
		ServiceLevelName:            &slName,
		ServiceLevelIsPortalEnabled: &slPortal,
		ServiceLevelToken:           &slToken,
		ServiceLevelCreatedAt:       now,
		ServiceLevelUpdatedAt:       now,
		PaymentTermId:               &ptID,
		PaymentTermName:             &ptName,
		PaymentTermIsActive:         &ptActive,
		ShippingTermId:              &stID,
		ShippingTermName:            &stName,
		ShippingTermType:            &stType,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}

	result := salesOrderDetailFromProto(info)
	resourcetest.ValidateExpandableStubs(t, "SalesOrder", result)
}

// salesOrderLineDetailFromProto must NOT embed a product stub on the line. The product is expandable and is populated only when the caller requests ?include=lines.product; otherwise the line.product field must be null. (Before the fix it leaked a half-populated stub with empty type / portal_visibility.)
func TestSalesOrderLineDetailFromProto_ProductNotEmbedded(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	productID := "prod_01abc"
	line := &pb.SalesOrderLineInfo{
		Id:             "sol_01abc",
		LineItemNumber: 1,
		ProductSku:     "SKU-1",
		ProductId:      &productID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	got := salesOrderLineDetailFromProto(line)

	if got.Product != nil {
		t.Errorf("line.Product must be nil until ?include=lines.product is requested, got %+v", got.Product)
	}
	resourcetest.AssertExpandablesNil(t, "SalesOrderLine", got)
}

// stashSalesOrderLineMeta must record the product id so that ?include=lines.product can still populate the line's product after the gated (product-less) build.
func TestStashSalesOrderLineMeta_StashesProductID(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	productID := "prod_01abc"
	info := &pb.SalesOrderLineInfo{
		Id:             "sol_01abc",
		LineItemNumber: 1,
		ProductId:      &productID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	line := salesOrderLineDetailFromProto(info)

	ctx := resourcekit.WithLoadMeta(context.Background())
	meta := resourcekit.GetLoadMeta(ctx)
	stashSalesOrderLineMeta(meta, info, &line)

	if v, _ := meta.GetString(constants.ObjectTypeSalesOrderLine, "sol_01abc", "product_id"); v != productID {
		t.Errorf("product_id meta = %q, want %q", v, productID)
	}
}
