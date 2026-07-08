package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var createdByLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.created_by")

// LoadCreatedBySalesOrders resolves the creator of each sales order from its `create` audit event (via platform-service), returning one CreatedBy per order id. Orders with no create event (e.g. system/EDI-created) resolve to a system CreatedBy with no actor, so the field is always present once included.
func LoadCreatedBySalesOrders(ctx context.Context, orderIDs []string) (map[string]any, *apierror.APIError) {
	if len(orderIDs) == 0 {
		return nil, nil
	}

	resp, apiErr := grpcutil.CallRPC(ctx, createdByLoaderTracer, "loader.created_by.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetResourceCreatorsResponse, error) {
			return auditClient.BatchGetResourceCreators(ctx, &pb.BatchGetResourceCreatorsRequest{
				ResourceType: string(constants.ObjectTypeSalesOrder),
				ResourceIds:  orderIDs,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	agentNames := loadCreatorAgentNames(ctx, resp.Creators)

	out := make(map[string]any, len(orderIDs))
	for _, c := range resp.Creators {
		out[c.ResourceId] = createdByFromAuditActor(c.Actor, agentNames)
	}
	// Resources with no create audit event resolve to system (no human actor).
	for _, id := range orderIDs {
		if _, ok := out[id]; !ok {
			out[id] = apiresource.SystemCreatedBy()
		}
	}
	return out, nil
}

// loadCreatorAgentNames resolves display names for agent creators. Audit rows only carry names for users and API keys (agent definitions live in the agent-service, outside the audit store's reach), so agent actors are resolved here. Best-effort: an unresolved agent keeps a nil name rather than failing the include.
func loadCreatorAgentNames(ctx context.Context, creators []*pb.ResourceCreator) map[string]AgentDefinitionName {
	var agentIDs []string
	seen := make(map[string]struct{})
	for _, c := range creators {
		a := c.Actor
		if a == nil || constants.ActorType(a.ActorType) != constants.ActorTypeAgent || (a.Name != nil && *a.Name != "") {
			continue
		}
		if _, ok := seen[a.Id]; ok {
			continue
		}
		seen[a.Id] = struct{}{}
		agentIDs = append(agentIDs, a.Id)
	}
	if len(agentIDs) == 0 {
		return nil
	}

	names, apiErr := LoadAgentDefinitionNames(ctx, agentIDs)
	if apiErr != nil {
		return nil
	}
	return names
}

// createdByFromAuditActor maps a create-event actor to a CreatedBy. AuditActor.Type carries the relation (internal/customer/supplier); only internal/customer map to a named creator — anything else is presented as system rather than inventing a value. Agent actors get their display name from agentNames, since the audit store cannot resolve it.
func createdByFromAuditActor(a *pb.AuditActor, agentNames map[string]AgentDefinitionName) *apiresource.CreatedBy {
	if a == nil {
		return apiresource.SystemCreatedBy()
	}
	relation := constants.CreatedByRelation(a.Type)
	if relation != constants.CreatedByRelationInternal && relation != constants.CreatedByRelationCustomer {
		return apiresource.SystemCreatedBy()
	}
	name := a.Name
	if def, ok := agentNames[a.Id]; ok && (name == nil || *name == "") {
		name = &def.Name
	}
	actor := apiresource.NewActor(a.Id, constants.ActorType(a.ActorType), name, a.Handle)
	return apiresource.NewCreatedBy(relation, actor)
}
