package shipmentep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestShipmentPresenter(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	carrierPortal := true
	slID := "crop_01abc"
	slName := "FedEx Ground"
	slPortal := true
	slToken := "fedex_ground"
	pickID := "pk_01abc"
	pickNumber := "PCK-0001"
	custStatus := "active"
	custCommission := "net"

	info := &pb.ShipmentInfo{
		Id:                          "sh_01abc",
		Number:                      "SHP-0001",
		StatusCode:                  "shipped",
		StatusName:                  "Shipped",
		SalesOrderId:                "so_01abc",
		SalesOrderNumber:            "SO-0001",
		CustomerId:                  "ac_01abc",
		CustomerName:                "Acme Corp",
		CustomerNumber:              "ACME001",
		CarrierId:                   "cr_01abc",
		CarrierName:                 "FedEx",
		CarrierIsPortalEnabled:      &carrierPortal,
		ServiceLevelId:              &slID,
		ServiceLevelName:            &slName,
		ServiceLevelIsPortalEnabled: &slPortal,
		ServiceLevelToken:           &slToken,
		PickId:                      &pickID,
		PickNumber:                  &pickNumber,
		CustomerStatusCode:          &custStatus,
		CustomerCommissionPolicy:    &custCommission,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}

	result := ShipmentPresenter(info)
	resourcetest.ValidateExpandableStubs(t, "ShipmentDetail", result)
}

func TestShipmentSummaryPresenter(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	carrierPortal := true
	slID := "crop_01abc"
	slName := "FedEx Ground"
	slPortal := true
	slToken := "fedex_ground"
	custStatus := "active"
	custCommission := "net"

	info := &pb.ShipmentSummaryInfo{
		Id:                          "sh_01abc",
		Number:                      "SHP-0001",
		StatusCode:                  "shipped",
		StatusName:                  "Shipped",
		SalesOrderId:                "so_01abc",
		SalesOrderNumber:            "SO-0001",
		CustomerId:                  "ac_01abc",
		CustomerName:                "Acme Corp",
		CustomerNumber:              "ACME001",
		CarrierId:                   "cr_01abc",
		CarrierName:                 "FedEx",
		CarrierIsPortalEnabled:      &carrierPortal,
		ServiceLevelId:              &slID,
		ServiceLevelName:            &slName,
		ServiceLevelIsPortalEnabled: &slPortal,
		ServiceLevelToken:           &slToken,
		CustomerStatusCode:          &custStatus,
		CustomerCommissionPolicy:    &custCommission,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}

	result := ShipmentSummaryPresenter(info)
	resourcetest.ValidateExpandableStubs(t, "ShipmentSummary", result)
}

func TestShippingCaseDetailPresenter(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	carrierPortal := true

	info := &pb.ShippingCaseDetailInfo{
		Id:                     "sc_01abc",
		Number:                 "SC-0001",
		CarrierId:              "cr_01abc",
		CarrierName:            "FedEx",
		CarrierIsPortalEnabled: &carrierPortal,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	result := ShippingCaseDetailPresenter(info)
	resourcetest.ValidateExpandableStubs(t, "ShippingCaseDetail", result)
}
