package httpgroup

import (
	"fmt"

	agentmemoryep "github.com/open-mrp/api/services/api-gateway/endpoints/agent-memories"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type AgentMemoriesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AgentMemoriesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client; memories are notes.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AgentMemoriesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("agent memories endpoint group: core client is required")
	}
	return nil
}

func (*AgentMemoriesEndpointGroup) Materialize(config *AgentMemoriesEndpointGroupConfig) *AgentMemoriesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	memorySvc := agentmemoryep.NewAgentMemorySvc(&agentmemoryep.AgentMemorySvcConfig{
		NoteClient: config.CoreClient.Note,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Agent Memories",
		Description:  "Agent memories, which are notes. These routes answer only 1.0.forge-preview.9 callers.",
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
