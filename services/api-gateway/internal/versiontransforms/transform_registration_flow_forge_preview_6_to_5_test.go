package versiontransforms

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func upgradeRegistrationFlowBody(method, route string, body map[string]any) map[string]any {
	return version.TransformEndpointRequest(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview6,
		constants.ObjectTypeRegistrationFlow, method, route, body)
}

func TestRegistrationFlowPreview6To5_UpdateBodies(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]any
		want map[string]any
	}{
		{
			name: "a flagged list is sent",
			in:   map[string]any{"has_payment_term_ids": true, "payment_term_ids": []any{"pytm_1"}},
			want: map[string]any{"payment_term_ids": []any{"pytm_1"}},
		},
		{
			name: "a flagged list that was omitted cleared the options",
			in:   map[string]any{"has_customer_group_ids": true},
			want: map[string]any{"customer_group_ids": []any{}},
		},
		{
			name: "a flagged null list cleared the options",
			in:   map[string]any{"has_shipping_term_ids": true, "shipping_term_ids": nil},
			want: map[string]any{"shipping_term_ids": []any{}},
		},
		{
			name: "a list flagged false was ignored",
			in:   map[string]any{"has_payment_term_ids": false, "payment_term_ids": []any{"pytm_1"}},
			want: map[string]any{},
		},
		{
			name: "an unflagged list was ignored",
			in:   map[string]any{"name": "Wholesale", "customer_group_ids": []any{"acgp_1"}},
			want: map[string]any{"name": "Wholesale"},
		},
		{
			name: "a flag that is not a boolean is left for the decoder to reject",
			in:   map[string]any{"has_payment_term_ids": "yes"},
			want: map[string]any{"has_payment_term_ids": "yes"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := upgradeRegistrationFlowBody(http.MethodPatch, registrationFlowUpdateRoute, tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// Create never had the flags: its lists were always applied.
func TestRegistrationFlowPreview6To5_CreateIsUntouched(t *testing.T) {
	in := map[string]any{"name": "Wholesale", "customer_group_ids": []any{"acgp_1"}}
	got := upgradeRegistrationFlowBody(http.MethodPost, "/v1/sales/registration-flows", in)
	want := map[string]any{"name": "Wholesale", "customer_group_ids": []any{"acgp_1"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRegistrationFlowPreview6To5_LatestIsUntouched(t *testing.T) {
	in := map[string]any{"customer_group_ids": []any{"acgp_1"}}
	got := version.TransformEndpointRequest(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6,
		constants.ObjectTypeRegistrationFlow, http.MethodPatch, registrationFlowUpdateRoute, in)
	if !reflect.DeepEqual(got, map[string]any{"customer_group_ids": []any{"acgp_1"}}) {
		t.Errorf("a preview.6 body must pass through, got %v", got)
	}
}

func TestRegistrationFlowPreview6To5_ResponsesPassThrough(t *testing.T) {
	data := map[string]any{"object": "registration_flow", "id": "rgfw_1", "name": "Wholesale"}
	got := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview5, constants.ObjectTypeRegistrationFlow, data)
	if !reflect.DeepEqual(got, map[string]any{"object": "registration_flow", "id": "rgfw_1", "name": "Wholesale"}) {
		t.Errorf("responses did not change, got %v", got)
	}
}
