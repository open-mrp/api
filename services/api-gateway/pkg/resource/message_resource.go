package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Message resource.
type MessageResource struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=message"`
	// Human-readable message.
	Message string `json:"message" validate:"required"`
}

var SampleMessageResource = &MessageResource{
	Object:  constants.ObjectTypeMessage,
	Message: "Operation completed successfully.",
}

func (*MessageResource) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMessageResource)
}
