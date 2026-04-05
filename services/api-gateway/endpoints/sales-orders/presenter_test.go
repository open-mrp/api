package salesorderep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSalesOrderSummaryPresenter(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	priorityID := "pi_01abc"

	custStatus := "active"
	custCommission := "net"

	info := &pb.SalesOrderSummaryInfo{
		Id:                       "so_01abc",
		Number:                   "SO-0001",
		CustomerId:               "ac_01abc",
		CustomerName:             "Acme Corp",
		CustomerNumber:           "ACME001",
		CustomerStatusCode:       &custStatus,
		CustomerCommissionPolicy: &custCommission,
		StatusCode:               "open",
		StatusName:               "Open",
		TypeCode:                 "standard",
		TypeName:                 "Standard",
		PriorityCode:             "normal",
		PriorityName:             "Normal",
		PriorityId:               &priorityID,
		LineCount:                1,
		IsAcknowledgmentSent:     true,
		CustomerPoNumber:         strPtr("PO-123"),
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	result := SalesOrderSummaryPresenter(info)
	resourcetest.ValidateExpandableStubs(t, "SalesOrderSummary", result)
}

func TestSalesOrderDetailPresenter(t *testing.T) {
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
		ServiceLevelId:              &slID,
		ServiceLevelName:            &slName,
		ServiceLevelIsPortalEnabled: &slPortal,
		ServiceLevelToken:           &slToken,
		PaymentTermId:               &ptID,
		PaymentTermName:             &ptName,
		PaymentTermIsActive:         &ptActive,
		ShippingTermId:              &stID,
		ShippingTermName:            &stName,
		ShippingTermType:            &stType,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}

	result := SalesOrderDetailPresenter(info)
	resourcetest.ValidateExpandableStubs(t, "SalesOrderDetail", result)
}

func strPtr(s string) *string { return &s }
