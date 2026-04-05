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
				ID:           sc.FreightAmountUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         sc.FreightAmountUnitName,
				Abbreviation: sc.FreightAmountUnitAbbreviation,
				Type:         constants.UnitType(sc.FreightAmountUnitType),
			},
		},
		FreightWeight: &apiresource.Quantity{
			ID:           sc.FreightWeightId,
			Object:       constants.ObjectTypeQuantity,
			Value:        sc.FreightWeightValue,
			DisplayValue: apiresource.FormatDisplayValue(sc.FreightWeightValue, sc.FreightWeightUnitAbbreviation, sc.FreightWeightUnitType),
			Unit: &apiresource.Unit{
				ID:           sc.FreightWeightUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         sc.FreightWeightUnitName,
				Abbreviation: sc.FreightWeightUnitAbbreviation,
				Type:         constants.UnitType(sc.FreightWeightUnitType),
			},
		},
		Shipment: &apiresource.ShipmentDetail{
			ID:     sc.ShipmentId,
			Object: constants.ObjectTypeShipment,
		},
		Carrier: &apiresource.Carrier{
			ID:     sc.CarrierId,
			Object: constants.ObjectTypeCarrier,
			Name:   sc.CarrierName,
		},
		CreatedAt: grpcutil.TimestampToTime(sc.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(sc.UpdatedAt),
	}

	if sc.ShippedAt != nil {
		t := grpcutil.TimestampToTime(sc.ShippedAt)
		result.ShippedAt = &t
	}

	return result
}
