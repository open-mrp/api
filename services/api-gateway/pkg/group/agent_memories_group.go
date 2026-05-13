package httpgroup

import (
	"fmt"

	agentmemoryep "github.com/augno/api/services/api-gateway/endpoints/agent-memories"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AgentMemoriesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AgentMemoriesEndpointGroupConfig struct {
	AgentClient *grpcclient.AgentServiceClient
}

func (c *AgentMemoriesEndpointGroupConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent memories endpoint group: agent client is required")
	}
	return nil
}

func (*AgentMemoriesEndpointGroup) Materialize(config *AgentMemoriesEndpointGroupConfig) *AgentMemoriesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	memorySvc := agentmemoryep.NewAgentMemorySvc(&agentmemoryep.AgentMemorySvcConfig{
		AgentClient: config.AgentClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Agent Memories",
		Description:  "List, create, update, and delete agent memories.",
		ResourceType: &apiresource.AgentMemory{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		(&agentmemoryep.ListMemoriesEndpoint{}).Materialize().WithService(inner, memorySvc),
		(&agentmemoryep.RetrieveMemoryEndpoint{}).Materialize().WithService(inner, memorySvc),
		(&agentmemoryep.CreateMemoryEndpoint{}).Materialize().WithService(inner, memorySvc),
		(&agentmemoryep.UpdateMemoryEndpoint{}).Materialize().WithService(inner, memorySvc),
		(&agentmemoryep.DeleteMemoryEndpoint{}).Materialize().WithService(inner, memorySvc),
	}

	return &AgentMemoriesEndpointGroup{inner}
}
