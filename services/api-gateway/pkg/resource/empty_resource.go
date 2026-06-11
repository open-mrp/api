package apiresource

// An empty object, used by endpoints that take no request parameters or return no response data.
type EmptyResource struct {
}

var exampleRequest = &EmptyResource{}

func (*EmptyResource) SchemaExample() any {
	return exampleRequest
}
