package resourceloaders

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/timeutil"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var agentRunLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.agent_run")

// LoadAgentRuns fetches agent runs by ID and builds base AgentRun references for expansion as a
// sub-resource (e.g. a chat message's ?include=agent_run). Only the inline run fields are
// populated; the run's own expandable sub-objects (triggered_by, actions, definition, steps) are
// not stashed here, so deeper includes through this path resolve to null. The agent service exposes
// no batch get, so runs are fetched one at a time; in practice each parent carries at most one run
// id and the resolver dedups across roots before calling this loader.
func LoadAgentRuns(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, agentRunLoaderTracer, "loader.agent_runs.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRunResponse, error) {
				return agentClient.GetRun(ctx, &pb.GetRunRequest{AgentRunId: id}, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}
		if resp.Run == nil {
			continue
		}
		out[id] = agentRunBaseFromProto(resp.Run)
	}
	return out, nil
}

// agentRunBaseFromProto maps the inline fields of an agent run proto to its API resource. It mirrors
// the base presenter in the agent-runs endpoint package, duplicated here because that package
// imports resourceloaders (importing it back would be a cycle).
func agentRunBaseFromProto(r *pb.AgentRunInfo) *apiresource.AgentRun {
	run := &apiresource.AgentRun{
		ID:           r.Id,
		Object:       constants.ObjectTypeAgentRun,
		Status:       constants.AgentRunStatus(r.StatusCode),
		TriggerType:  constants.AgentTriggerType(r.TriggerType),
		ErrorMessage: ptrStringOrNilForRun(r.ErrorMessage),
		DurationMs:   ptrInt32OrNilForRun(r.DurationMs),
		StartedAt:    timeutil.TimestampToTimePtr(r.StartedAt),
		CompletedAt:  timeutil.TimestampToTimePtr(r.CompletedAt),
		CreatedAt:    timeutil.TimestampToTime(r.CreatedAt),
		UpdatedAt:    timeutil.TimestampToTime(r.UpdatedAt),
	}
	if r.Input != "" {
		run.Input = json.RawMessage(r.Input)
	}
	if r.Output != "" {
		run.Output = json.RawMessage(r.Output)
	}
	return run
}

func ptrStringOrNilForRun(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrInt32OrNilForRun(v int32) *int32 {
	if v == 0 {
		return nil
	}
	return &v
}
