package quantityep

import (
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resource/resourcetest"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUnitFromQuantityInfo_UsesUnitDetailWhenPresent(t *testing.T) {
	t.Parallel()

	now := timestamppb.Now()
	q := &pb.QuantityInfo{
		Id:               "qty_01abc",
		Value:            "12.500000000000000000000000000000",
		UnitId:           "un_01abc",
		UnitName:         "Kilogram",
		UnitAbbreviation: "kg",
		UnitType:         "mass",
		CreatedAt:        now,
		UpdatedAt:        now,
		UnitDetail: &pb.UnitInfo{
			Id:                "un_01abc",
			Name:              "Kilogram",
			Abbreviation:      "kg",
			Type:              "mass",
			RatioNumerator:    "1000",
			RatioDenominator:  "1",
			OffsetNumerator:   "0",
			OffsetDenominator: "1",
			IsBaseUnit:        false,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	unit := UnitFromQuantityInfo(q)
	if unit == nil {
		t.Fatal("expected unit")
	}

	wrapped := apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        "12.5",
		DisplayValue: "12.5 kg",
		Unit:         unit,
	}
	resourcetest.ValidateResourceStruct(t, "QuantityInfo", wrapped)
	resourcetest.ValidatePopulatedExpandableFields(t, "QuantityInfo", wrapped)
	if unit.RatioNumerator != "1000" {
		t.Fatalf("expected ratio_numerator 1000, got %q", unit.RatioNumerator)
	}
	if unit.CreatedAt.IsZero() {
		t.Fatal("expected created_at to be populated from unit_detail")
	}
}

func TestUnitFromQuantityInfo_ReturnsNilWhenNoUnitDetail(t *testing.T) {
	t.Parallel()

	now := timestamppb.Now()
	q := &pb.QuantityInfo{
		Id:               "qty_01abc",
		Value:            "12.500000000000000000000000000000",
		UnitId:           "un_01abc",
		UnitName:         "Kilogram",
		UnitAbbreviation: "kg",
		UnitType:         "mass",
		CreatedAt:        now,
		UpdatedAt:        timestamppb.Now(),
	}

	// No unit_detail on the proto: never fabricate a placeholder unit. The unit
	// id is stashed elsewhere so the real Unit loads via ?include=unit.
	if unit := UnitFromQuantityInfo(q); unit != nil {
		t.Fatalf("expected nil when proto carries no unit detail, got %+v", unit)
	}
}

func TestQuantityFromProto_PopulatesNestedUnitContract(t *testing.T) {
	t.Parallel()

	now := timestamppb.Now()
	q := &pb.QuantityInfo{
		Id:               "qty_01abc",
		Value:            "12.500000000000000000000000000000",
		UnitId:           "un_01abc",
		UnitName:         "Kilogram",
		UnitAbbreviation: "kg",
		UnitType:         "mass",
		CreatedAt:        now,
		UpdatedAt:        now,
		UnitDetail: &pb.UnitInfo{
			Id:                "un_01abc",
			Name:              "Kilogram",
			Abbreviation:      "kg",
			Type:              "mass",
			RatioNumerator:    "1000",
			RatioDenominator:  "1",
			OffsetNumerator:   "0",
			OffsetDenominator: "1",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	result := quantityFromProto(q)
	result.Unit = UnitFromQuantityInfo(q)
	resourcetest.ValidateResourceStruct(t, "Quantity", result)
	resourcetest.ValidatePopulatedExpandableFields(t, "Quantity", result)
}
