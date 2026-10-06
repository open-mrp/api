package versiontransforms

import (
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func downgradeAccountUserToPreview5(objectType constants.ObjectType, data map[string]any) map[string]any {
	return version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview5, objectType, data)
}

func preview6AccountUser(is_commission_eligible any) map[string]any {
	return map[string]any{
		"id":                     "account_user_1",
		"object":                 "account_user",
		"is_commission_eligible": is_commission_eligible,
		"last_used_at":           nil,
	}
}

func TestAccountUserPreview6To5_WithheldValueReadsAsTheZeroValue(t *testing.T) {
	got := downgradeAccountUserToPreview5(constants.ObjectType("account_user"), preview6AccountUser(nil))
	if want := preview6AccountUser(false); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAccountUserPreview6To5_VisibleValueIsLeftAlone(t *testing.T) {
	got := downgradeAccountUserToPreview5(constants.ObjectType("account_user"), preview6AccountUser(true))
	if want := preview6AccountUser(true); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// An account user rides inside other resources through includes, so it is downgraded under whatever object type the response is served as.
func TestAccountUserPreview6To5_ReachesListsAndEmbeddedAccountUsers(t *testing.T) {
	list := map[string]any{"object": "list", "data": []any{preview6AccountUser(nil), preview6AccountUser(true)}}
	got := downgradeAccountUserToPreview5(constants.ObjectType("account_user"), list)
	data := got["data"].([]any)
	if data[0].(map[string]any)["is_commission_eligible"] != false || data[1].(map[string]any)["is_commission_eligible"] != true {
		t.Errorf("list: got %v", data)
	}

	parent := map[string]any{"object": "customer_defaults", "sales_rep": preview6AccountUser(nil), "is_commission_eligible": nil}
	got = downgradeAccountUserToPreview5(constants.ObjectTypeSalesOrder, parent)
	embedded := got["sales_rep"].(map[string]any)
	if embedded["is_commission_eligible"] != false {
		t.Errorf("embedded: got %v", embedded)
	}
	if embedded["last_used_at"] != nil {
		t.Errorf("a field preview.5 already allowed to be null stays null, got %v", embedded["last_used_at"])
	}
	if got["is_commission_eligible"] != nil {
		t.Errorf("only an account user's is_commission_eligible is touched, got %v", got["is_commission_eligible"])
	}
}

func TestAccountUserPreview6To5_DataMissing(t *testing.T) {
	for _, in := range []map[string]any{
		{"object": "account_user", "id": "account_user_1"},
		{"object": "customer_defaults", "sales_rep": nil},
		{},
	} {
		want := map[string]any{}
		for k, v := range in {
			want[k] = v
		}
		got := downgradeAccountUserToPreview5(constants.ObjectTypeSalesOrder, in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestAccountUserPreview6To5_LatestIsUntouched(t *testing.T) {
	got := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectType("account_user"), preview6AccountUser(nil))
	if !reflect.DeepEqual(got, preview6AccountUser(nil)) {
		t.Errorf("a preview.6 response must pass through, got %v", got)
	}
}

func TestAccountUserPreview6To5_RequestsPassThrough(t *testing.T) {
	got := version.TransformRequest(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview6, constants.ObjectType("account_user"), map[string]any{"is_commission_eligible": true})
	if !reflect.DeepEqual(got, map[string]any{"is_commission_eligible": true}) {
		t.Errorf("requests did not change, got %v", got)
	}
}
