package versiontransforms

import (
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func commitmentPayload() map[string]any {
	return map[string]any{
		"object":                   "commitment",
		"promised_at":              "2026-08-22T00:00:00Z",
		"lead_time_override_days":  nil,
		"ship_by_override_date":    nil,
		"ship_by_date":             "2026-08-20T19:00:00Z",
		"lead_time_days":           float64(16),
		"lead_time_source":         "manual",
		"transit_days":             float64(3),
		"transit_source":           "carrier_lane",
		"calendar_adjustment_days": float64(2),
		"estimated_delivery_date":  "2026-08-25T00:00:00Z",
	}
}

func salesOrderPreview4Payload() map[string]any {
	return map[string]any{
		"id":         "so_123",
		"object":     "sales_order",
		"commitment": commitmentPayload(),
	}
}

func TestTransformCommitment_HoistsOntoTheSalesOrder(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	result := tr.Transform(constants.ObjectTypeSalesOrder, salesOrderPreview4Payload())

	if _, ok := result["commitment"]; ok {
		t.Error("preview.3 had no commitment sub-resource")
	}
	if result["ship_by_date"] != "2026-08-20T00:00:00Z" {
		t.Errorf("ship_by_date: got %v", result["ship_by_date"])
	}
	if result["ship_by_cutoff_at"] != "2026-08-20T19:00:00Z" {
		t.Errorf("ship_by_cutoff_at: got %v", result["ship_by_cutoff_at"])
	}
	// preview.3 carried the inputs on the order, not inside a commitment.
	if result["promised_at"] != "2026-08-22T00:00:00Z" {
		t.Errorf("promised_at must hoist, got %v", result["promised_at"])
	}
	if result["lead_time_source"] != "manual" || result["transit_source"] != "carrier_lane" {
		t.Errorf("sources: got %v, %v", result["lead_time_source"], result["transit_source"])
	}
	if result["calendar_adjustment_days"] != float64(2) {
		t.Errorf("calendar_adjustment_days: got %v", result["calendar_adjustment_days"])
	}
}

// preview.3 never showed an arrival estimate on an order, so hoisting one would invent a field.
func TestTransformCommitment_SalesOrderDoesNotGainTheArrivalEstimate(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	result := tr.Transform(constants.ObjectTypeSalesOrder, salesOrderPreview4Payload())

	if _, ok := result["estimated_delivery_date"]; ok {
		t.Error("an order carried no estimated_delivery_date in preview.3")
	}
}

// A pick carried five of the fields flat and never the cutoff or the calendar adjustment.
func TestTransformCommitment_PickGetsOnlyTheFieldsItCarried(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	result := tr.Transform(constants.ObjectTypePick, map[string]any{
		"id":         "pk_123",
		"object":     "pick",
		"commitment": commitmentPayload(),
	})

	if result["ship_by_date"] != "2026-08-20T00:00:00Z" || result["lead_time_days"] != float64(16) {
		t.Errorf("the pick's fields must hoist, with the ship-by back to a bare day: %v", result)
	}
	for _, absent := range []string{"ship_by_cutoff_at", "calendar_adjustment_days", "estimated_delivery_date", "lead_time_override_days"} {
		if _, ok := result[absent]; ok {
			t.Errorf("a pick never carried %s", absent)
		}
	}
}

func TestTransformCommitment_QuoteFlattensBackToEightFields(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	result := tr.Transform(constants.ObjectTypeSalesOrderCommitmentQuote, map[string]any{
		"object":     "sales_order_commitment_quote",
		"commitment": commitmentPayload(),
		"steps":      []any{map[string]any{"code": "basis", "date": "2026-08-22T00:00:00Z"}},
	})

	if result["estimated_delivery_date"] != "2026-08-25T00:00:00Z" {
		t.Errorf("the preview did report an arrival, got %v", result["estimated_delivery_date"])
	}
	steps, ok := result["steps"].([]any)
	if !ok || len(steps) != 1 {
		t.Errorf("steps sat beside the commitment in both versions, got %v", result["steps"])
	}
}

// preview.3 typed the preview's adjustment as a plain int, so a null would break a caller that read it as a number.
func TestTransformCommitment_QuoteAdjustmentNeverDowngradesToNull(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	result := tr.Transform(constants.ObjectTypeSalesOrderCommitmentQuote, map[string]any{
		"object":     "sales_order_commitment_quote",
		"commitment": map[string]any{"object": "commitment"},
	})

	if result["calendar_adjustment_days"] != 0 {
		t.Errorf("expected 0, got %v", result["calendar_adjustment_days"])
	}
}

// An order with nothing committed still has to answer with the preview.3 keys, as nulls.
func TestTransformCommitment_AbsentCommitmentStillRestoresTheKeys(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	result := tr.Transform(constants.ObjectTypeSalesOrder, map[string]any{
		"id":     "so_123",
		"object": "sales_order",
	})

	for _, key := range append(preview3CommitmentKeys[string(constants.ObjectTypeSalesOrder)], "ship_by_date", "ship_by_cutoff_at") {
		v, ok := result[key]
		if !ok {
			t.Errorf("%s must be present, as null", key)
		}
		if v != nil {
			t.Errorf("%s: got %v, want null", key, v)
		}
	}
}

// A pick nested in a list envelope, and an order nested under it, both have to be rewritten.
func TestTransformCommitment_WalksListsAndNestedResources(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	result := tr.Transform(constants.ObjectTypePick, map[string]any{
		"object": "list",
		"data": []any{
			map[string]any{
				"id":         "pk_123",
				"object":     "pick",
				"commitment": commitmentPayload(),
				"related": map[string]any{
					"object":      "pick_related",
					"sales_order": salesOrderPreview4Payload(),
				},
			},
		},
	})

	pick := result["data"].([]any)[0].(map[string]any)
	if _, ok := pick["commitment"]; ok {
		t.Error("the pick in the envelope was not downgraded")
	}
	nested := pick["related"].(map[string]any)["sales_order"].(map[string]any)
	if nested["ship_by_date"] != "2026-08-20T00:00:00Z" || nested["ship_by_cutoff_at"] != "2026-08-20T19:00:00Z" {
		t.Errorf("the nested order was not downgraded: %v", nested)
	}
}

func TestTransformCommitment_LeavesOtherObjectsAlone(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	result := tr.Transform(constants.ObjectTypeShipment, map[string]any{
		"id":         "shp_123",
		"object":     "shipment",
		"commitment": commitmentPayload(),
	})

	if _, ok := result["commitment"]; !ok {
		t.Error("a shipment carries no preview.3 commitment shape, so nothing should move")
	}
}

func TestTransformCommitment_RequestIsIdentity(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	in := map[string]any{"promised_at": "2026-08-22T00:00:00Z"}
	if got := tr.TransformRequest(constants.ObjectTypeSalesOrder, in); got["promised_at"] != in["promised_at"] {
		t.Errorf("the request shape did not change between the versions, got %v", got)
	}
}

func TestTransformCommitment_DefaultRegistryEndToEnd(t *testing.T) {
	t.Parallel()

	result := version.Transform(
		version.V1_0_Forge_Preview4,
		version.V1_0_Forge_Preview3,
		constants.ObjectTypeSalesOrder,
		salesOrderPreview4Payload(),
	)

	if result["lead_time_source"] != "manual" {
		t.Errorf("expected the default registry to apply the downgrade, got %v", result["lead_time_source"])
	}
}

func TestTransformCommitment_DowngradesTheRealSampleResources(t *testing.T) {
	t.Parallel()
	tr := &commitmentForgePreview4To3{}

	order := tr.Transform(constants.ObjectTypeSalesOrder, marshalResource(t, apiresource.SampleSalesOrder))
	if order["ship_by_date"] != "2026-08-20T00:00:00Z" || order["lead_time_source"] != "manual" {
		t.Errorf("sample order: got %v / %v", order["ship_by_date"], order["lead_time_source"])
	}
	if order["ship_by_cutoff_at"] != "2026-08-20T19:00:00Z" {
		t.Errorf("the sample's 3pm cutoff must split back out, got %v", order["ship_by_cutoff_at"])
	}

	pick := tr.Transform(constants.ObjectTypePick, marshalResource(t, apiresource.SamplePick))
	if pick["transit_source"] != "carrier_lane" {
		t.Errorf("sample pick: got %v", pick["transit_source"])
	}
	if _, ok := pick["ship_by_cutoff_at"]; ok {
		t.Error("the sample pick must not gain the cutoff")
	}
}

// The instant preview.4 reports is the deadline; preview.3 wanted the day and the time apart.
func TestTransformCommitment_SplitsShipByOnMidnight(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		shipBy     any
		wantDay    any
		wantCutoff any
	}{
		{"a cutoff splits into day and instant", "2026-08-20T19:00:00Z", "2026-08-20T00:00:00Z", "2026-08-20T19:00:00Z"},
		{"midnight means no cutoff was configured", "2026-08-20T00:00:00Z", "2026-08-20T00:00:00Z", nil},
		{"an uncommitted order has neither", nil, nil, nil},
		{"an unparseable date downgrades to nulls rather than a guess", "not-a-date", nil, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := &commitmentForgePreview4To3{}

			result := tr.Transform(constants.ObjectTypeSalesOrder, map[string]any{
				"object":     "sales_order",
				"commitment": map[string]any{"object": "commitment", "ship_by_date": tc.shipBy},
			})

			if result["ship_by_date"] != tc.wantDay {
				t.Errorf("ship_by_date: got %v, want %v", result["ship_by_date"], tc.wantDay)
			}
			if result["ship_by_cutoff_at"] != tc.wantCutoff {
				t.Errorf("ship_by_cutoff_at: got %v, want %v", result["ship_by_cutoff_at"], tc.wantCutoff)
			}
		})
	}
}
