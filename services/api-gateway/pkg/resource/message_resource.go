package apiresource

// MessageResource is a simple resource containing a message string.
type MessageResource struct {
	// A human-readable message.
	Message string `json:"message" validate:"required"`
}

var exampleMessageResource = &MessageResource{
	Message: "Operation completed successfully.",
}

func (*MessageResource) SchemaExample() any {
	return exampleMessageResource
}
