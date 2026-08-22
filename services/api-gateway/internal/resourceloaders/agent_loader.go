package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	agentpb "github.com/open-mrp/api/shared/proto/agent"
	"github.com/open-mrp/api/shared/timeutil"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var agentDefinitionLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.agent_definition")

// LoadAgentDefinitions fetches agent definitions by ID and builds base
// AgentDefinition references for expansion as a sub-resource (e.g. an email
// inbox's ?include=agent_config). Only the inline definition fields are
// populated; the definition's own expandable sub-objects (config, tools, role)
// are not stashed here, so deeper includes through this path resolve to null.
// AgentService exposes no batch get, so definitions are fetched one at a time;
// in practice each parent carries a single agent id and the resolver dedups
// across roots before calling this loader.
func LoadAgentDefinitions(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		agentID := id
		resp, apiErr := grpcutil.CallRPC(ctx, agentDefinitionLoaderTracer, "loader.agent_definitions.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*agentpb.GetAgentDefinitionResponse, error) {
				return agentClient.GetAgentDefinition(ctx, &agentpb.GetAgentDefinitionRequest{AgentDefinitionId: agentID}, opts...)
			})
		if apiErr != nil {
			if apierror.IsNotFound(apiErr) {
				continue
			}
			return nil, apiErr
		}
		if resp == nil || resp.Agent == nil {
			continue
		}
		out[agentID] = agentDefinitionBaseFromProto(resp.Agent)
	}
	return out, nil
}

// agentDefinitionBaseFromProto maps the inline fields of an agent definition
// proto to its API resource. It mirrors AgentDefinitionPresenter in the agents
// endpoint package, duplicated here because that package would otherwise form an
// import cycle with resourceloaders. Expandable sub-objects (config, tools,
// role) are intentionally left unset.
func agentDefinitionBaseFromProto(a *agentpb.AgentDefinitionInfo) *apiresource.AgentDefinition {
	accountStatus := constants.AgentAccountStatusInactive
	if a.AccountStatus != nil {
		accountStatus = constants.AgentAccountStatus(a.AccountStatus.StatusCode)
	}

	return &apiresource.AgentDefinition{
		ID:             a.Id,
		Object:         constants.ObjectTypeAgentDefinition,
		DefinitionType: constants.AgentDefinitionType(a.DefinitionType),
		CategoryCode:   a.CategoryCode,
		TriggerType:    constants.AgentTriggerType(a.TriggerType),
		Name:           a.Name,
		Slug:           a.Slug,
		Description:    a.Description,
		Editability:    constants.EditabilityFromBool(a.IsEditable),
		AccountStatus:  accountStatus,
		CreatedAt:      timeutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:      timeutil.TimestampToTime(a.UpdatedAt),
	}
}
