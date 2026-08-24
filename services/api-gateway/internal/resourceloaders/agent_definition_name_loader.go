package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apierror "github.com/open-mrp/api/shared/errors"
	agentpb "github.com/open-mrp/api/shared/proto/agent"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var agentDefinitionNameLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.agent_definition_name")

// AgentDefinitionName is the minimal display info for an agent participant actor.
type AgentDefinitionName struct {
	Name string
	Slug string
}

// LoadAgentDefinitionNames resolves agent definition ids (agdf_) to their display name + slug via
// AgentService. There is no batch RPC, so it fetches each unique id with GetAgentDefinition (the set is small — agents per conversation are few). Best-effort: an unresolved id is omitted rather than failing the request.
func LoadAgentDefinitionNames(ctx context.Context, ids []string) (map[string]AgentDefinitionName, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make(map[string]AgentDefinitionName, len(ids))
	for _, id := range ids {
		agentID := id
		resp, apiErr := grpcutil.CallRPC(ctx, agentDefinitionNameLoaderTracer, "loader.agent_definitions.name", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*agentpb.GetAgentDefinitionResponse, error) {
				return agentClient.GetAgentDefinition(ctx, &agentpb.GetAgentDefinitionRequest{AgentDefinitionId: agentID}, opts...)
			})
		if apiErr != nil || resp == nil || resp.Agent == nil {
			continue // best-effort: skip ids that don't resolve
		}
		out[agentID] = AgentDefinitionName{Name: resp.Agent.Name, Slug: resp.Agent.Slug}
	}
	return out, nil
}

// AgentDefinitionExists reports whether an agent definition id resolves through AgentService, for callers that need to validate a caller-supplied reference before storing it. A missing agent is (false, nil); any other failure is returned so an AgentService outage surfaces as itself rather than as a rejected request.
func AgentDefinitionExists(ctx context.Context, id string) (bool, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, agentDefinitionNameLoaderTracer, "loader.agent_definitions.exists", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*agentpb.GetAgentDefinitionResponse, error) {
			return agentClient.GetAgentDefinition(ctx, &agentpb.GetAgentDefinitionRequest{AgentDefinitionId: id}, opts...)
		})
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return false, nil
		}
		return false, apiErr
	}
	return resp != nil && resp.Agent != nil, nil
}
