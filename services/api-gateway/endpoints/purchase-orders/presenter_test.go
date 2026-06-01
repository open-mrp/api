package purchaseorderep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestPurchaseOrderSummaryPresenter(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	priorityID := "pi_01abc"

	info := &pb.PurchaseOrderSummaryInfo{
		Id:                   "po_01abc",
		Number:               "PO-0001",
		SupplierId:           "sp_01abc",
		SupplierName:         "Supplier Co",
		SupplierNumber:       "SUP001",
		StatusCode:           "open",
		StatusName:           "Open",
		TypeCode:             "standard",
		TypeName:             "Standard",
		PriorityCode:         "normal",
		PriorityName:         "Normal",
		PriorityId:           &priorityID,
		LineCount:            1,
		IsAcknowledgmentSent: true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	result := purchaseOrderSummaryFromProto(info)
	resourcetest.ValidateExpandableStubs(t, "PurchaseOrderSummary", result)
}

func TestPurchaseOrderDetailPresenter(t *testing.T) {
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

	info := &pb.PurchaseOrderInfo{
		Id:                          "po_01abc",
		Number:                      "PO-0001",
		SupplierId:                  "sp_01abc",
		SupplierName:                "Supplier Co",
		SupplierNumber:              "SUP001",
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

	result := purchaseOrderDetailFromProto(info)
	resourcetest.ValidateExpandableStubs(t, "PurchaseOrderDetail", result)
}
