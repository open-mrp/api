package httpgroup

import (
	"fmt"

	contactep "github.com/open-mrp/api/services/api-gateway/endpoints/contacts"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type ContactsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ContactsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ContactsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("contacts endpoint group: core client is required")
	}
	return nil
}

func (*ContactsEndpointGroup) Materialize(config *ContactsEndpointGroupConfig) *ContactsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	contactSvc := contactep.NewContactSvc(&contactep.ContactSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Contacts",
		Description:  "Look up the people you do business with.",
		ResourceType: &apiresource.ContactMatch{},
	}

	findByEmailEndpoint := apiendpoint.From(&contactep.FindContactByEmailEndpoint{}).WithService(inner, contactSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		findByEmailEndpoint,
	}

	return &ContactsEndpointGroup{inner}
}
