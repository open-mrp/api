package versiontransforms

import (
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func downgradeCustomerToPreview5(objectType constants.ObjectType, data map[string]any) map[string]any {
	return version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview5, objectType, data)
}

func preview6Customer(commission_policy any) map[string]any {
	return map[string]any{
		"id":                "customer_1",
		"object":            "customer",
		"commission_policy": commission_policy,
		"note":              nil,
	}
}

func TestCustomerPreview6To5_WithheldValueReadsAsTheZeroValue(t *testing.T) {
	got := downgradeCustomerToPreview5(constants.ObjectType("customer"), preview6Customer(nil))
	if want := preview6Customer(""); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCustomerPreview6To5_VisibleValueIsLeftAlone(t *testing.T) {
	got := downgradeCustomerToPreview5(constants.ObjectType("customer"), preview6Customer("commission_applied"))
	if want := preview6Customer("commission_applied"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A customer rides inside other resources through includes, so it is downgraded under whatever object type the response is served as.
func TestCustomerPreview6To5_ReachesListsAndEmbeddedCustomers(t *testing.T) {
	list := map[string]any{"object": "list", "data": []any{preview6Customer(nil), preview6Customer("commission_applied")}}
	got := downgradeCustomerToPreview5(constants.ObjectType("customer"), list)
	data := got["data"].([]any)
	if data[0].(map[string]any)["commission_policy"] != "" || data[1].(map[string]any)["commission_policy"] != "commission_applied" {
		t.Errorf("list: got %v", data)
	}

	parent := map[string]any{"object": "sales_order", "customer": preview6Customer(nil), "commission_policy": nil}
	got = downgradeCustomerToPreview5(constants.ObjectTypeSalesOrder, parent)
	embedded := got["customer"].(map[string]any)
	if embedded["commission_policy"] != "" {
		t.Errorf("embedded: got %v", embedded)
	}
	if embedded["note"] != nil {
		t.Errorf("a field preview.5 already allowed to be null stays null, got %v", embedded["note"])
	}
	if got["commission_policy"] != nil {
		t.Errorf("only a customer's commission_policy is touched, got %v", got["commission_policy"])
	}
}

func TestCustomerPreview6To5_DataMissing(t *testing.T) {
	for _, in := range []map[string]any{
		{"object": "customer", "id": "customer_1"},
		{"object": "sales_order", "customer": nil},
		{},
	} {
		want := map[string]any{}
		for k, v := range in {
			want[k] = v
		}
		got := downgradeCustomerToPreview5(constants.ObjectTypeSalesOrder, in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestCustomerPreview6To5_LatestIsUntouched(t *testing.T) {
	got := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectType("customer"), preview6Customer(nil))
	if !reflect.DeepEqual(got, preview6Customer(nil)) {
		t.Errorf("a preview.6 response must pass through, got %v", got)
	}
}

func TestCustomerPreview6To5_RequestsPassThrough(t *testing.T) {
	got := version.TransformRequest(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview6, constants.ObjectType("customer"), map[string]any{"commission_policy": "commission_applied"})
	if !reflect.DeepEqual(got, map[string]any{"commission_policy": "commission_applied"}) {
		t.Errorf("requests did not change, got %v", got)
	}
}
