package materialep

import (
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The presenter names each quantity's unit; the gateway loads the unit in full before the response is written.
func TestMaterialPresenter_QuantitiesNameTheirUnits(t *testing.T) {
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
	for name, tc := range map[string]struct {
		got        *apiresource.Quantity
		value, uid string
	}{
		"order_point": {result.OrderPoint, "10", "un_01kg"},
		"lead_time":   {result.LeadTime, "5", "un_01day"},
	} {
		if tc.got == nil {
			t.Fatalf("%s is nil", name)
		}
		if tc.got.Value != tc.value {
			t.Errorf("%s value = %q, want %q", name, tc.got.Value, tc.value)
		}
		if !apiresource.IsUnitStub(tc.got.Unit) || tc.got.Unit.ID != tc.uid {
			t.Errorf("%s unit = %+v, want a stub for %s", name, tc.got.Unit, tc.uid)
		}
	}
}
