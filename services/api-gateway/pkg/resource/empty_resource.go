package apiresource

// Empty resource.
type EmptyResource struct {
}

var exampleRequest = &EmptyResource{}

func (*EmptyResource) SchemaExample() any {
	return exampleRequest
}
