package versiontransforms

import (
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func downgradeProductLineToPreview5(objectType constants.ObjectType, data map[string]any) map[string]any {
	return version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview5, objectType, data)
}

func preview6ProductLine(commission_policy any) map[string]any {
	return map[string]any{
		"id":                "product_line_1",
		"object":            "product_line",
		"commission_policy": commission_policy,
		"notes":             nil,
	}
}

func TestProductLinePreview6To5_WithheldValueReadsAsTheZeroValue(t *testing.T) {
	got := downgradeProductLineToPreview5(constants.ObjectType("product_line"), preview6ProductLine(nil))
	if want := preview6ProductLine(""); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestProductLinePreview6To5_VisibleValueIsLeftAlone(t *testing.T) {
	got := downgradeProductLineToPreview5(constants.ObjectType("product_line"), preview6ProductLine("commission_applied"))
	if want := preview6ProductLine("commission_applied"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A product line rides inside other resources through includes, so it is downgraded under whatever object type the response is served as.
func TestProductLinePreview6To5_ReachesListsAndEmbeddedProductLines(t *testing.T) {
	list := map[string]any{"object": "list", "data": []any{preview6ProductLine(nil), preview6ProductLine("commission_applied")}}
	got := downgradeProductLineToPreview5(constants.ObjectType("product_line"), list)
	data := got["data"].([]any)
	if data[0].(map[string]any)["commission_policy"] != "" || data[1].(map[string]any)["commission_policy"] != "commission_applied" {
		t.Errorf("list: got %v", data)
	}

	parent := map[string]any{"object": "product", "product_line": preview6ProductLine(nil), "commission_policy": nil}
	got = downgradeProductLineToPreview5(constants.ObjectTypeSalesOrder, parent)
	embedded := got["product_line"].(map[string]any)
	if embedded["commission_policy"] != "" {
		t.Errorf("embedded: got %v", embedded)
	}
	if embedded["notes"] != nil {
		t.Errorf("a field preview.5 already allowed to be null stays null, got %v", embedded["notes"])
	}
	if got["commission_policy"] != nil {
		t.Errorf("only a product line's commission_policy is touched, got %v", got["commission_policy"])
	}
}

func TestProductLinePreview6To5_DataMissing(t *testing.T) {
	for _, in := range []map[string]any{
		{"object": "product_line", "id": "product_line_1"},
		{"object": "product", "product_line": nil},
		{},
	} {
		want := map[string]any{}
		for k, v := range in {
			want[k] = v
		}
		got := downgradeProductLineToPreview5(constants.ObjectTypeSalesOrder, in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestProductLinePreview6To5_LatestIsUntouched(t *testing.T) {
	got := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectType("product_line"), preview6ProductLine(nil))
	if !reflect.DeepEqual(got, preview6ProductLine(nil)) {
		t.Errorf("a preview.6 response must pass through, got %v", got)
	}
}

func TestProductLinePreview6To5_RequestsPassThrough(t *testing.T) {
	got := version.TransformRequest(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview6, constants.ObjectType("product_line"), map[string]any{"commission_policy": "commission_applied"})
	if !reflect.DeepEqual(got, map[string]any{"commission_policy": "commission_applied"}) {
		t.Errorf("requests did not change, got %v", got)
	}
}
