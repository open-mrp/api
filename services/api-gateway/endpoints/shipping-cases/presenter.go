package shippingcaseep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ShippingCasePresenter(sc *pb.ShippingCaseInfo) apiresource.ShippingCase {
	if sc == nil {
		return apiresource.ShippingCase{}
	}

	result := apiresource.ShippingCase{
		ID:             sc.Id,
		Object:         constants.ObjectTypeShippingCase,
		Number:         sc.Number,
		SSCC:           sc.Sscc,
		TrackingNumber: sc.TrackingNumber,
		FreightAmount: &apiresource.Quantity{
			ID:           sc.FreightAmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        sc.FreightAmountValue,
			DisplayValue: apiresource.FormatDisplayValue(sc.FreightAmountValue, sc.FreightAmountUnitAbbreviation, sc.FreightAmountUnitType),
			Unit: &apiresource.Unit{
				ID:                sc.FreightAmountUnitId,
				Object:            constants.ObjectTypeUnit,
				Name:              sc.FreightAmountUnitName,
				Abbreviation:      sc.FreightAmountUnitAbbreviation,
				Type:              constants.UnitType(sc.FreightAmountUnitType),
				RatioNumerator:    sc.FreightAmountUnitRatioNumerator,
				RatioDenominator:  sc.FreightAmountUnitRatioDenominator,
				OffsetNumerator:   sc.FreightAmountUnitOffsetNumerator,
				OffsetDenominator: sc.FreightAmountUnitOffsetDenominator,
				CreatedAt:         grpcutil.TimestampToTime(sc.FreightAmountUnitCreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(sc.FreightAmountUnitUpdatedAt),
			},
		},
		FreightWeight: &apiresource.Quantity{
			ID:           sc.FreightWeightId,
			Object:       constants.ObjectTypeQuantity,
			Value:        sc.FreightWeightValue,
			DisplayValue: apiresource.FormatDisplayValue(sc.FreightWeightValue, sc.FreightWeightUnitAbbreviation, sc.FreightWeightUnitType),
			Unit: &apiresource.Unit{
				ID:                sc.FreightWeightUnitId,
				Object:            constants.ObjectTypeUnit,
				Name:              sc.FreightWeightUnitName,
				Abbreviation:      sc.FreightWeightUnitAbbreviation,
				Type:              constants.UnitType(sc.FreightWeightUnitType),
				RatioNumerator:    sc.FreightWeightUnitRatioNumerator,
				RatioDenominator:  sc.FreightWeightUnitRatioDenominator,
				OffsetNumerator:   sc.FreightWeightUnitOffsetNumerator,
				OffsetDenominator: sc.FreightWeightUnitOffsetDenominator,
				CreatedAt:         grpcutil.TimestampToTime(sc.FreightWeightUnitCreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(sc.FreightWeightUnitUpdatedAt),
			},
		},
		Shipment: &apiresource.ShipmentDetail{
			ID:     sc.ShipmentId,
			Object: constants.ObjectTypeShipment,
			Number: sc.GetShipmentNumber(),
			Status: apiresource.ShipmentStatus{
				Code: sc.GetShipmentStatusCode(),
				Name: sc.GetShipmentStatusName(),
			},
			CreatedAt: grpcutil.TimestampToTime(sc.ShipmentCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(sc.ShipmentUpdatedAt),
		},
		CreatedAt: grpcutil.TimestampToTime(sc.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(sc.UpdatedAt),
	}

	carrier := &apiresource.Carrier{
		ID:     sc.CarrierId,
		Object: constants.ObjectTypeCarrier,
		Name:   sc.CarrierName,
	}
	if sc.CarrierIsPortalEnabled != nil && *sc.CarrierIsPortalEnabled {
		carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
	} else {
		carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
	}
	if sc.CarrierCreatedAt != nil {
		carrier.CreatedAt = sc.CarrierCreatedAt.AsTime()
	}
	if sc.CarrierUpdatedAt != nil {
		carrier.UpdatedAt = sc.CarrierUpdatedAt.AsTime()
	}
	result.Carrier = carrier

	if sc.ShippedAt != nil {
		t := grpcutil.TimestampToTime(sc.ShippedAt)
		result.ShippedAt = &t
	}

	return result
}
