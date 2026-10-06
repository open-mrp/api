package versiontransforms

import (
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func downgradeAccountGroupToPreview5(objectType constants.ObjectType, data map[string]any) map[string]any {
	return version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview5, objectType, data)
}

func preview6AccountGroup(commission_policy any) map[string]any {
	return map[string]any{
		"id":                "account_group_1",
		"object":            "account_group",
		"commission_policy": commission_policy,
		"description":       nil,
	}
}

func TestAccountGroupPreview6To5_WithheldValueReadsAsTheZeroValue(t *testing.T) {
	got := downgradeAccountGroupToPreview5(constants.ObjectType("account_group"), preview6AccountGroup(nil))
	if want := preview6AccountGroup(""); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAccountGroupPreview6To5_VisibleValueIsLeftAlone(t *testing.T) {
	got := downgradeAccountGroupToPreview5(constants.ObjectType("account_group"), preview6AccountGroup("commission_exempt"))
	if want := preview6AccountGroup("commission_exempt"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// An account group rides inside other resources through includes, so it is downgraded under whatever object type the response is served as.
func TestAccountGroupPreview6To5_ReachesListsAndEmbeddedAccountGroups(t *testing.T) {
	list := map[string]any{"object": "list", "data": []any{preview6AccountGroup(nil), preview6AccountGroup("commission_exempt")}}
	got := downgradeAccountGroupToPreview5(constants.ObjectType("account_group"), list)
	data := got["data"].([]any)
	if data[0].(map[string]any)["commission_policy"] != "" || data[1].(map[string]any)["commission_policy"] != "commission_exempt" {
		t.Errorf("list: got %v", data)
	}

	parent := map[string]any{"object": "account_group_product_line_access", "account_group": preview6AccountGroup(nil), "commission_policy": nil}
	got = downgradeAccountGroupToPreview5(constants.ObjectTypeSalesOrder, parent)
	embedded := got["account_group"].(map[string]any)
	if embedded["commission_policy"] != "" {
		t.Errorf("embedded: got %v", embedded)
	}
	if embedded["description"] != nil {
		t.Errorf("a field preview.5 already allowed to be null stays null, got %v", embedded["description"])
	}
	if got["commission_policy"] != nil {
		t.Errorf("only an account group's commission_policy is touched, got %v", got["commission_policy"])
	}
}

func TestAccountGroupPreview6To5_DataMissing(t *testing.T) {
	for _, in := range []map[string]any{
		{"object": "account_group", "id": "account_group_1"},
		{"object": "account_group_product_line_access", "account_group": nil},
		{},
	} {
		want := map[string]any{}
		for k, v := range in {
			want[k] = v
		}
		got := downgradeAccountGroupToPreview5(constants.ObjectTypeSalesOrder, in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestAccountGroupPreview6To5_LatestIsUntouched(t *testing.T) {
	got := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectType("account_group"), preview6AccountGroup(nil))
	if !reflect.DeepEqual(got, preview6AccountGroup(nil)) {
		t.Errorf("a preview.6 response must pass through, got %v", got)
	}
}

func TestAccountGroupPreview6To5_RequestsPassThrough(t *testing.T) {
	got := version.TransformRequest(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview6, constants.ObjectType("account_group"), map[string]any{"commission_policy": "commission_exempt"})
	if !reflect.DeepEqual(got, map[string]any{"commission_policy": "commission_exempt"}) {
		t.Errorf("requests did not change, got %v", got)
	}
}
