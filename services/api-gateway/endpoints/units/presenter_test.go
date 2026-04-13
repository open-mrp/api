package unitep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUnitPresenter(t *testing.T) {
	t.Parallel()
	acctID := "ac_01abc"
	unit := &pb.UnitInfo{
		Id:                "unit_01abc",
		Name:              "Kilogram",
		Abbreviation:      "kg",
		Type:              "mass",
		RatioNumerator:    "1000.0",
		RatioDenominator:  "1.0",
		OffsetNumerator:   "0.0",
		OffsetDenominator: "1.0",
		IsBaseUnit:        true,
		IsInternal:        true,
		AccountId:         &acctID,
		CreatedAt:         timestamppb.Now(),
		UpdatedAt:         timestamppb.Now(),
	}

	result := UnitPresenter(unit, nil)
	resourcetest.ValidateResourceStruct(t, "Unit", result)
}
