package apiresource

// Message resource.
type MessageResource struct {
	// Human-readable message.
	Message string `json:"message" validate:"required"`
}

var exampleMessageResource = &MessageResource{
	Message: "Operation completed successfully.",
}

func (*MessageResource) SchemaExample() any {
	return exampleMessageResource
}
