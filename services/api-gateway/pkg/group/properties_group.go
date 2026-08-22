package httpgroup

import (
	"fmt"

	propertyep "github.com/open-mrp/api/services/api-gateway/endpoints/properties"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type PropertiesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PropertiesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PropertiesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("properties endpoint group: core client is required")
	}
	return nil
}

func (*PropertiesEndpointGroup) Materialize(config *PropertiesEndpointGroupConfig) *PropertiesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	propertySvc := propertyep.NewPropertySvc(&propertyep.PropertySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Properties",
		Description:  "List and manage properties and their attributes.",
		ResourceType: &apiresource.Property{},
	}

	listPropertiesEndpoint := apiendpoint.From(&propertyep.ListPropertiesEndpoint{}).WithService(inner, propertySvc)
	getPropertyEndpoint := apiendpoint.From(&propertyep.RetrievePropertyEndpoint{}).WithService(inner, propertySvc)
	createPropertyEndpoint := apiendpoint.From(&propertyep.CreatePropertyEndpoint{}).WithService(inner, propertySvc)
	updatePropertyEndpoint := apiendpoint.From(&propertyep.UpdatePropertyEndpoint{}).WithService(inner, propertySvc)
	deletePropertyEndpoint := apiendpoint.From(&propertyep.DeletePropertyEndpoint{}).WithService(inner, propertySvc)
	listAttributesEndpoint := apiendpoint.From(&propertyep.ListAttributesEndpoint{}).WithService(inner, propertySvc)
	getAttributeEndpoint := apiendpoint.From(&propertyep.RetrieveAttributeEndpoint{}).WithService(inner, propertySvc)
	createAttributeEndpoint := apiendpoint.From(&propertyep.CreateAttributeEndpoint{}).WithService(inner, propertySvc)
	updateAttributeEndpoint := apiendpoint.From(&propertyep.UpdateAttributeEndpoint{}).WithService(inner, propertySvc)
	deleteAttributeEndpoint := apiendpoint.From(&propertyep.DeleteAttributeEndpoint{}).WithService(inner, propertySvc)
	bulkUpsertPropertiesEndpoint := apiendpoint.From(&propertyep.BulkUpsertPropertiesEndpoint{}).WithService(inner, propertySvc)
	exportPropertiesEndpoint := apiendpoint.From(&propertyep.ExportPropertiesEndpoint{}).WithService(inner, propertySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listPropertiesEndpoint,
		getPropertyEndpoint,
		createPropertyEndpoint,
		updatePropertyEndpoint,
		deletePropertyEndpoint,
		listAttributesEndpoint,
		getAttributeEndpoint,
		createAttributeEndpoint,
		updateAttributeEndpoint,
		deleteAttributeEndpoint,
		bulkUpsertPropertiesEndpoint,
		exportPropertiesEndpoint,
	}

	return &PropertiesEndpointGroup{inner}
}
