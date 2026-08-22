package httpgroup

import (
	"fmt"

	portaldomainep "github.com/open-mrp/api/services/api-gateway/endpoints/portal-domains"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type PortalDomainsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PortalDomainsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PortalDomainsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("portal domains endpoint group: core client is required")
	}
	return nil
}

func (*PortalDomainsEndpointGroup) Materialize(config *PortalDomainsEndpointGroupConfig) *PortalDomainsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	portalDomainSvc := portaldomainep.NewPortalDomainSvc(&portaldomainep.PortalDomainSvcConfig{
		PortalDomainClient: config.CoreClient.PortalDomain,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Portal Domains",
		Description:  "Connect a custom domain to the account's customer portal, verify its DNS, and resolve custom hosts to portal accounts.",
		ResourceType: &apiresource.PortalDomain{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&portaldomainep.CreatePortalDomainEndpoint{}).WithService(inner, portalDomainSvc),
		apiendpoint.From(&portaldomainep.ListPortalDomainsEndpoint{}).WithService(inner, portalDomainSvc),
		apiendpoint.From(&portaldomainep.GetPortalDomainEndpoint{}).WithService(inner, portalDomainSvc),
		apiendpoint.From(&portaldomainep.VerifyPortalDomainEndpoint{}).WithService(inner, portalDomainSvc),
		apiendpoint.From(&portaldomainep.DeletePortalDomainEndpoint{}).WithService(inner, portalDomainSvc),
		apiendpoint.From(&portaldomainep.ResolvePortalHostEndpoint{}).WithService(inner, portalDomainSvc),
	}

	return &PortalDomainsEndpointGroup{inner}
}
