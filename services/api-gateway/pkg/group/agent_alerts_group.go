package httpgroup

import (
	"fmt"

	agentalertep "github.com/augno/api/services/api-gateway/endpoints/agent-alerts"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AgentAlertsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AgentAlertsEndpointGroupConfig struct {
	// AgentClient (required) is the agent-service gRPC client.
	AgentClient *grpcclient.AgentServiceClient
}

func (c *AgentAlertsEndpointGroupConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent alerts endpoint group: agent client is required")
	}
	return nil
}

func (*AgentAlertsEndpointGroup) Materialize(config *AgentAlertsEndpointGroupConfig) *AgentAlertsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	alertSvc := agentalertep.NewAgentAlertSvc(&agentalertep.AgentAlertSvcConfig{
		AgentClient: config.AgentClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Agent Alerts",
		Description:  "List, retrieve, and acknowledge agent alerts.",
		ResourceType: &apiresource.AgentAlert{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&agentalertep.ListAlertsEndpoint{}).WithService(inner, alertSvc),
		apiendpoint.From(&agentalertep.RetrieveAlertEndpoint{}).WithService(inner, alertSvc),
		apiendpoint.From(&agentalertep.AcknowledgeAlertEndpoint{}).WithService(inner, alertSvc),
	}

	return &AgentAlertsEndpointGroup{inner}
}
