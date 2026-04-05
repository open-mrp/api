package rateep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func RatePresenter(r *pb.RateInfo) apiresource.Rate {
	if r == nil {
		return apiresource.Rate{}
	}

	return apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  r.Value,
		NumeratorUnit: &apiresource.Unit{
			ID:     r.NumeratorUnitId,
			Object: constants.ObjectTypeUnit,
		},
		DenominatorUnit: &apiresource.Unit{
			ID:     r.DenominatorUnitId,
			Object: constants.ObjectTypeUnit,
		},
		DisplayValue: apiresource.FormatRateDisplayValue(r.Value, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.DenominatorUnitAbbreviation),
		CreatedAt:    grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(r.UpdatedAt),
	}
}
