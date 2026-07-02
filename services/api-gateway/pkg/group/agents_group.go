package httpgroup

import (
	"fmt"

	agentep "github.com/augno/api/services/api-gateway/endpoints/agents"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AgentsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AgentsEndpointGroupConfig struct {
	// AgentClient (required) is the agent-service gRPC client.
	AgentClient *grpcclient.AgentServiceClient
}

func (c *AgentsEndpointGroupConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agents endpoint group: agent client is required")
	}
	return nil
}

func (*AgentsEndpointGroup) Materialize(config *AgentsEndpointGroupConfig) *AgentsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	agentSvc := agentep.NewAgentSvc(&agentep.AgentSvcConfig{
		AgentClient: config.AgentClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Agent",
		Description:  "List, create, update, and delete agent definitions.",
		ResourceType: &apiresource.AgentDefinition{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&agentep.CreateAgentEndpoint{}).WithService(inner, agentSvc),
		apiendpoint.From(&agentep.ListAgentsEndpoint{}).WithService(inner, agentSvc),
		apiendpoint.From(&agentep.RetrieveAgentEndpoint{}).WithService(inner, agentSvc),
		apiendpoint.From(&agentep.UpdateAgentEndpoint{}).WithService(inner, agentSvc),
		apiendpoint.From(&agentep.DeleteAgentEndpoint{}).WithService(inner, agentSvc),
		apiendpoint.From(&agentep.UpdateAgentStatusEndpoint{}).WithService(inner, agentSvc),
	}

	return &AgentsEndpointGroup{inner}
}
