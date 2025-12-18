package apiresource

// We use `200 OK` with empty JSON objects instead of `204 No Content`.
//
// Unlike 204, this allows us to add response fields later without breaking
// backwards compatibility, since 204s cannot include a body per HTTP spec.
type EmptyResource struct {
}

var exampleRequest = &EmptyResource{}

func (*EmptyResource) SchemaExample() any {
	return exampleRequest
}
