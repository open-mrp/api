package httpgroup

import (
	"fmt"

	agenttoolep "github.com/augno/api/services/api-gateway/endpoints/agent-tools"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AgentToolsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AgentToolsEndpointGroupConfig struct {
	// AgentClient (required) is the agent-service gRPC client.
	AgentClient *grpcclient.AgentServiceClient
}

func (c *AgentToolsEndpointGroupConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent tools endpoint group: agent client is required")
	}
	return nil
}

func (*AgentToolsEndpointGroup) Materialize(config *AgentToolsEndpointGroupConfig) *AgentToolsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	toolSvc := agenttoolep.NewAgentToolSvc(&agenttoolep.AgentToolSvcConfig{
		AgentClient: config.AgentClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Agent Tools",
		Description:  "List available platform tools for agent configuration.",
		ResourceType: &apiresource.AvailableTool{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&agenttoolep.ListToolsEndpoint{}).WithService(inner, toolSvc),
		apiendpoint.From(&agenttoolep.ListToolGroupsEndpoint{}).WithService(inner, toolSvc),
	}

	return &AgentToolsEndpointGroup{inner}
}
