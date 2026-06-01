package quantityep

import (
	"testing"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
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

func TestUnitFromQuantityInfo_FallsBackToExpandableUnitStub(t *testing.T) {
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
	resourcetest.ValidateResourceStruct(t, "QuantityInfo.UnitFallback", wrapped)
	resourcetest.ValidatePopulatedExpandableFields(t, "QuantityInfo.UnitFallback", wrapped)
	if unit.Object != constants.ObjectTypeUnit {
		t.Fatalf("expected unit object type, got %q", unit.Object)
	}
	if unit.RatioNumerator != "1" || unit.RatioDenominator != "1" {
		t.Fatalf("expected fallback ratios 1/1, got %s/%s", unit.RatioNumerator, unit.RatioDenominator)
	}
	if unit.OffsetNumerator != "0" || unit.OffsetDenominator != "1" {
		t.Fatalf("expected fallback offsets 0/1, got %s/%s", unit.OffsetNumerator, unit.OffsetDenominator)
	}
	if !unit.CreatedAt.Equal(now.AsTime()) || !unit.UpdatedAt.Equal(now.AsTime()) {
		t.Fatalf("expected fallback timestamps to use quantity created_at, got created_at=%v updated_at=%v", unit.CreatedAt, unit.UpdatedAt)
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
	}

	result := quantityFromProto(q)
	result.Unit = UnitFromQuantityInfo(q)
	resourcetest.ValidateResourceStruct(t, "Quantity", result)
	resourcetest.ValidatePopulatedExpandableFields(t, "Quantity", result)
}
