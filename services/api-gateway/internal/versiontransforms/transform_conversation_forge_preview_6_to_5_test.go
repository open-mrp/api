package versiontransforms

import (
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func downgradeConversationToPreview5(objectType constants.ObjectType, data map[string]any) map[string]any {
	return version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview5, objectType, data)
}

func preview6Conversation(legal_hold any) map[string]any {
	return map[string]any{
		"id":         "conversation_1",
		"object":     "conversation",
		"legal_hold": legal_hold,
		"assignee":   nil,
	}
}

func TestConversationPreview6To5_WithheldValueReadsAsTheZeroValue(t *testing.T) {
	got := downgradeConversationToPreview5(constants.ObjectType("conversation"), preview6Conversation(nil))
	if want := preview6Conversation(""); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestConversationPreview6To5_VisibleValueIsLeftAlone(t *testing.T) {
	got := downgradeConversationToPreview5(constants.ObjectType("conversation"), preview6Conversation("held"))
	if want := preview6Conversation("held"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A conversation rides inside other resources through includes, so it is downgraded under whatever object type the response is served as.
func TestConversationPreview6To5_ReachesListsAndEmbeddedConversations(t *testing.T) {
	list := map[string]any{"object": "list", "data": []any{preview6Conversation(nil), preview6Conversation("held")}}
	got := downgradeConversationToPreview5(constants.ObjectType("conversation"), list)
	data := got["data"].([]any)
	if data[0].(map[string]any)["legal_hold"] != "" || data[1].(map[string]any)["legal_hold"] != "held" {
		t.Errorf("list: got %v", data)
	}

	parent := map[string]any{"object": "support_case", "conversation": preview6Conversation(nil), "legal_hold": nil}
	got = downgradeConversationToPreview5(constants.ObjectTypeSalesOrder, parent)
	embedded := got["conversation"].(map[string]any)
	if embedded["legal_hold"] != "" {
		t.Errorf("embedded: got %v", embedded)
	}
	if embedded["assignee"] != nil {
		t.Errorf("a field preview.5 already allowed to be null stays null, got %v", embedded["assignee"])
	}
	if got["legal_hold"] != nil {
		t.Errorf("only a conversation's legal_hold is touched, got %v", got["legal_hold"])
	}
}

func TestConversationPreview6To5_DataMissing(t *testing.T) {
	for _, in := range []map[string]any{
		{"object": "conversation", "id": "conversation_1"},
		{"object": "support_case", "conversation": nil},
		{},
	} {
		want := map[string]any{}
		for k, v := range in {
			want[k] = v
		}
		got := downgradeConversationToPreview5(constants.ObjectTypeSalesOrder, in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestConversationPreview6To5_LatestIsUntouched(t *testing.T) {
	got := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectType("conversation"), preview6Conversation(nil))
	if !reflect.DeepEqual(got, preview6Conversation(nil)) {
		t.Errorf("a preview.6 response must pass through, got %v", got)
	}
}

func TestConversationPreview6To5_RequestsPassThrough(t *testing.T) {
	got := version.TransformRequest(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview6, constants.ObjectType("conversation"), map[string]any{"legal_hold": "held"})
	if !reflect.DeepEqual(got, map[string]any{"legal_hold": "held"}) {
		t.Errorf("requests did not change, got %v", got)
	}
}
