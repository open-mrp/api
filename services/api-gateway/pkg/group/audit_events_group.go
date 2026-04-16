package httpgroup

import (
	"fmt"

	auditeventep "github.com/augno/api/services/api-gateway/endpoints/audit-events"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AuditEventsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AuditEventsEndpointGroupConfig struct {
	PlatformClient *grpcclient.PlatformServiceClient
}

func (c *AuditEventsEndpointGroupConfig) validate() error {
	if c.PlatformClient == nil {
		return fmt.Errorf("audit events endpoint group: platform client is required")
	}
	return nil
}

func (*AuditEventsEndpointGroup) Materialize(config *AuditEventsEndpointGroupConfig) *AuditEventsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	auditEventSvc := auditeventep.NewAuditEventSvc(&auditeventep.AuditEventSvcConfig{
		AuditClient: config.PlatformClient.AuditClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Audit Event Management",
		Description:  "List and retrieve audit events.",
		ResourceType: &apiresource.AuditEvent{},
	}

	listEndpoint := (&auditeventep.ListAuditEventsEndpoint{}).Materialize().WithService(inner, auditEventSvc)
	listResourceTypesEndpoint := (&auditeventep.ListAuditEventResourceTypesEndpoint{}).Materialize().WithService(inner, auditEventSvc)
	getEndpoint := (&auditeventep.GetAuditEventEndpoint{}).Materialize().WithService(inner, auditEventSvc)

	// Order matters: the router's catch-all picks the LAST matching route, so
	// the static /resource-types path must be registered after the /{id} wildcard.
	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		getEndpoint,
		listResourceTypesEndpoint,
	}

	return &AuditEventsEndpointGroup{inner}
}
