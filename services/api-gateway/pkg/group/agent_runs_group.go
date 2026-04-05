package httpgroup

import (
	"fmt"

	agentrunep "github.com/augno/api/services/api-gateway/endpoints/agent-runs"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AgentRunsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AgentRunsEndpointGroupConfig struct {
	AgentClient *grpcclient.AgentServiceClient
}

func (c *AgentRunsEndpointGroupConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent runs endpoint group: agent client is required")
	}
	return nil
}

func (*AgentRunsEndpointGroup) Materialize(config *AgentRunsEndpointGroupConfig) *AgentRunsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	runSvc := agentrunep.NewAgentRunSvc(&agentrunep.AgentRunSvcConfig{
		AgentClient: config.AgentClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Agent Runs",
		Description:  "List, retrieve, trigger, cancel, and continue agent runs.",
		ResourceType: &apiresource.AgentRun{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		(&agentrunep.ListRunsEndpoint{}).Materialize().WithService(inner, runSvc),
		(&agentrunep.GetRunEndpoint{}).Materialize().WithService(inner, runSvc),
		(&agentrunep.TriggerRunEndpoint{}).Materialize().WithService(inner, runSvc),
		(&agentrunep.CancelRunEndpoint{}).Materialize().WithService(inner, runSvc),
		(&agentrunep.ContinueRunEndpoint{}).Materialize().WithService(inner, runSvc),
	}

	return &AgentRunsEndpointGroup{inner}
}
