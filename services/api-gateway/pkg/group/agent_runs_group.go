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
	CoreClient  *grpcclient.CoreServiceClient
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

	runSvcCfg := &agentrunep.AgentRunSvcConfig{
		AgentClient: config.AgentClient.Client,
	}
	if config.CoreClient != nil {
		runSvcCfg.CoreClient = config.CoreClient.Client
	}
	runSvc := agentrunep.NewAgentRunSvc(runSvcCfg)

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Agent Runs",
		Description:  "List, retrieve, trigger, cancel, and continue agent runs.",
		ResourceType: &apiresource.AgentRun{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&agentrunep.ListRunsEndpoint{}).WithService(inner, runSvc),
		apiendpoint.From(&agentrunep.RetrieveRunEndpoint{}).WithService(inner, runSvc),
		apiendpoint.From(&agentrunep.TriggerRunEndpoint{}).WithService(inner, runSvc),
		apiendpoint.From(&agentrunep.CancelRunEndpoint{}).WithService(inner, runSvc),
		apiendpoint.From(&agentrunep.ContinueRunEndpoint{}).WithService(inner, runSvc),
	}

	return &AgentRunsEndpointGroup{inner}
}
