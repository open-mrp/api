package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&conversationForgePreview6To5{})
}

// conversationForgePreview6To5 downgrades conversations from 1.0.forge-preview.6 to 1.0.forge-preview.5, wherever one appears in a response.
//
// preview.6 withholds a conversation's `legal_hold` from customer and supplier portal users, so it became nullable where preview.5 always carried `held` or `released`. A preview.5 portal caller reads it as ""; the seller's own users read the real status in both versions. Requests did not change.
type conversationForgePreview6To5 struct{}

var conversationForgePreview6To5Withheld = map[string]any{"legal_hold": ""}

func (t *conversationForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *conversationForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *conversationForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return everyObjectType()
}

func (t *conversationForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	zeroWithheldIn(data, constants.ObjectTypeConversation, conversationForgePreview6To5Withheld)
	return data
}

func (t *conversationForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}
