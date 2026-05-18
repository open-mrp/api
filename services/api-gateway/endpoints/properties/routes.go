package propertyep

// Canonical HTTP route templates for this package. Endpoint Materialize Route
// fields must use these constants; presenters build concrete paths via
// apiendpoint.ExpandRoute.
const (
	CatalogPropertiesRoute         = "/v1/catalog/properties"
	CatalogPropertyRoute           = "/v1/catalog/properties/{id}"
	CatalogPropertyAttributesRoute = "/v1/catalog/properties/{property_id}/attributes"
	CatalogPropertyAttributeRoute  = "/v1/catalog/properties/{property_id}/attributes/{id}"
)
