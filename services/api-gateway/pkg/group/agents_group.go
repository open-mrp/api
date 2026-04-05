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
	AgentClient *grpcclient.AgentServiceClient
	CoreClient  *grpcclient.CoreServiceClient
}

func (c *AgentsEndpointGroupConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agents endpoint group: agent client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("agents endpoint group: core client is required")
	}
	return nil
}

func (*AgentsEndpointGroup) Materialize(config *AgentsEndpointGroupConfig) *AgentsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	agentSvc := agentep.NewAgentSvc(&agentep.AgentSvcConfig{
		AgentClient: config.AgentClient.Client,
		CoreClient:  config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Agent Management",
		Description:  "List, create, update, and delete agent definitions.",
		ResourceType: &apiresource.AgentDefinition{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		(&agentep.CreateAgentEndpoint{}).Materialize().WithService(inner, agentSvc),
		(&agentep.ListAgentsEndpoint{}).Materialize().WithService(inner, agentSvc),
		(&agentep.GetAgentEndpoint{}).Materialize().WithService(inner, agentSvc),
		(&agentep.UpdateAgentEndpoint{}).Materialize().WithService(inner, agentSvc),
		(&agentep.DeleteAgentEndpoint{}).Materialize().WithService(inner, agentSvc),
		(&agentep.UpdateAgentStatusEndpoint{}).Materialize().WithService(inner, agentSvc),
		(&agentep.ListUsageEndpoint{}).Materialize().WithService(inner, agentSvc),
	}

	return &AgentsEndpointGroup{inner}
}
