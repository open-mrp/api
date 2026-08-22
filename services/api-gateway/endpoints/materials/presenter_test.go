package materialep

import (
	"testing"

	"github.com/open-mrp/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMaterialPresenter_PopulatesQuantityUnits(t *testing.T) {
	t.Parallel()

	now := timestamppb.Now()
	info := &pb.MaterialInfo{
		Id: "ml_01abc",
		OrderPoint: &pb.QuantityInfo{
			Id:               "qty_01op",
			Value:            "10.000000000000000000000000000000",
			UnitId:           "un_01kg",
			UnitName:         "Kilogram",
			UnitAbbreviation: "kg",
			UnitType:         "mass",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		LeadTime: &pb.QuantityInfo{
			Id:               "qty_01lt",
			Value:            "5.000000000000000000000000000000",
			UnitId:           "un_01day",
			UnitName:         "Day",
			UnitAbbreviation: "d",
			UnitType:         "time",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := MaterialPresenter(info)
	resourcetest.ValidateResourceStruct(t, "Material", result)
	resourcetest.ValidatePopulatedExpandableFields(t, "Material", result)
}
