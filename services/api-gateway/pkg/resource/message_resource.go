package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// A human-readable confirmation returned by operations that do not produce a resource.
type MessageResource struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=message"`
	// Plain-language summary of what the operation did.
	//
	// This text is meant for display only. Its wording can change at any time, so do not parse it or branch on its contents.
	Message string `json:"message" validate:"required"`
}

var SampleMessageResource = &MessageResource{
	Object:  constants.ObjectTypeMessage,
	Message: "Operation completed successfully.",
}

func (*MessageResource) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMessageResource)
}
