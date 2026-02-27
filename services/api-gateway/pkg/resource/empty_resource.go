package apiresource

// EmptyResource is an empty resource.
type EmptyResource struct {
}

var exampleRequest = &EmptyResource{}

func (*EmptyResource) SchemaExample() any {
	return exampleRequest
}
