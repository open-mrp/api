package httpgroup

import (
	"fmt"

	propertyep "github.com/augno/api/services/api-gateway/endpoints/properties"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type PropertiesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PropertiesEndpointGroupConfig struct {
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
		Title:        "Properties Management",
		Description:  "List and manage properties and their attributes.",
		ResourceType: &apiresource.Property{},
	}

	listPropertiesEndpoint := (&propertyep.ListPropertiesEndpoint{}).Materialize().WithService(inner, propertySvc)
	getPropertyEndpoint := (&propertyep.GetPropertyEndpoint{}).Materialize().WithService(inner, propertySvc)
	createPropertyEndpoint := (&propertyep.CreatePropertyEndpoint{}).Materialize().WithService(inner, propertySvc)
	updatePropertyEndpoint := (&propertyep.UpdatePropertyEndpoint{}).Materialize().WithService(inner, propertySvc)
	deletePropertyEndpoint := (&propertyep.DeletePropertyEndpoint{}).Materialize().WithService(inner, propertySvc)
	listAttributesEndpoint := (&propertyep.ListAttributesEndpoint{}).Materialize().WithService(inner, propertySvc)
	getAttributeEndpoint := (&propertyep.GetAttributeEndpoint{}).Materialize().WithService(inner, propertySvc)
	createAttributeEndpoint := (&propertyep.CreateAttributeEndpoint{}).Materialize().WithService(inner, propertySvc)
	updateAttributeEndpoint := (&propertyep.UpdateAttributeEndpoint{}).Materialize().WithService(inner, propertySvc)
	deleteAttributeEndpoint := (&propertyep.DeleteAttributeEndpoint{}).Materialize().WithService(inner, propertySvc)

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
	}

	return &PropertiesEndpointGroup{inner}
}
