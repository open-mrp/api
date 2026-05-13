package apiendpoint

import (
	"testing"
	"time"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
)

func TestValidateExpandableFields_RejectsIncompleteNestedUnit(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeShippingTerm,
		Fields:     []string{"flat_rate.unit"},
	})
	req := map[string]bool{"flat_rate.unit": true}

	st := apiresource.ShippingTerm{
		ID:        "shtm_x",
		Object:    constants.ObjectTypeShippingTerm,
		Name:      "Test",
		Type:      constants.ShippingTermTypeFlatRateFreight,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FlatRate: &apiresource.Quantity{
			ID:           "qu_x",
			Object:       constants.ObjectTypeQuantity,
			Value:        "1.00",
			DisplayValue: "$1.00",
			Unit: &apiresource.Unit{
				ID:           "un_x",
				Object:       constants.ObjectTypeUnit,
				Name:         "",
				Abbreviation: "$",
				Type:         constants.UnitTypeCurrency,
			},
		},
		FreeShippingServiceLevels: apiresource.NewList([]apiresource.ServiceLevel{}, apiresource.PageInfo{}),
	}

	if err := ValidateExpandableFields(&st, req, cfg); err == nil {
		t.Fatal("expected validation error for empty unit.name")
	}
}

func TestValidateExpandableFields_AcceptsCompleteNestedUnit(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeShippingTerm,
		Fields:     []string{"flat_rate.unit"},
	})
	req := map[string]bool{"flat_rate.unit": true}
	now := time.Now().UTC().Truncate(time.Second)

	st := apiresource.ShippingTerm{
		ID:        "shtm_x",
		Object:    constants.ObjectTypeShippingTerm,
		Name:      "Test",
		Type:      constants.ShippingTermTypeFlatRateFreight,
		CreatedAt: now,
		UpdatedAt: now,
		FlatRate: &apiresource.Quantity{
			ID:           "qu_x",
			Object:       constants.ObjectTypeQuantity,
			Value:        "1.00",
			DisplayValue: "$1.00",
			Unit: &apiresource.Unit{
				ID:                "un_x",
				Object:            constants.ObjectTypeUnit,
				Name:              "US Dollar",
				Abbreviation:      "$",
				Type:              constants.UnitTypeCurrency,
				RatioNumerator:    "1",
				RatioDenominator:  "1",
				OffsetNumerator:   "0",
				OffsetDenominator: "1",
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
		FreeShippingServiceLevels: apiresource.NewList([]apiresource.ServiceLevel{}, apiresource.PageInfo{}),
	}

	if err := ValidateExpandableFields(&st, req, cfg); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateExpandableFields_ListDataElements(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeShippingTerm,
		Fields:     []string{"free_shipping_service_levels"},
	})
	req := map[string]bool{"free_shipping_service_levels": true}
	now := time.Now().UTC().Truncate(time.Second)

	st := apiresource.ShippingTerm{
		ID:        "shtm_x",
		Object:    constants.ObjectTypeShippingTerm,
		Name:      "Test",
		Type:      constants.ShippingTermTypeCarrierRateFreight,
		CreatedAt: now,
		UpdatedAt: now,
		FreeShippingServiceLevels: apiresource.NewList([]apiresource.ServiceLevel{
			{
				ID:                       "crop_x",
				Object:                   constants.ObjectTypeServiceLevel,
				Name:                     "Ground",
				ServiceLevelToken:        "ground",
				CustomerPortalVisibility: constants.CustomerPortalVisibilityVisible,
				CreatedAt:                now,
				UpdatedAt:                now,
			},
		}, apiresource.PageInfo{}),
	}

	if err := ValidateExpandableFields(&st, req, cfg); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	st.FreeShippingServiceLevels.Data[0].Name = ""
	if err := ValidateExpandableFields(&st, req, cfg); err == nil {
		t.Fatal("expected validation error for empty service level name")
	}
}
