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
	// AgentClient (required) is the agent-service gRPC client.
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
		apiendpoint.From(&agentmemoryep.ListMemoriesEndpoint{}).WithService(inner, memorySvc),
		apiendpoint.From(&agentmemoryep.RetrieveMemoryEndpoint{}).WithService(inner, memorySvc),
		apiendpoint.From(&agentmemoryep.CreateMemoryEndpoint{}).WithService(inner, memorySvc),
		apiendpoint.From(&agentmemoryep.UpdateMemoryEndpoint{}).WithService(inner, memorySvc),
		apiendpoint.From(&agentmemoryep.DeleteMemoryEndpoint{}).WithService(inner, memorySvc),
	}

	return &AgentMemoriesEndpointGroup{inner}
}
