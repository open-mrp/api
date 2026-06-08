package agentrunep

import (
	"context"
	"encoding/json"
	"time"

	agentep "github.com/augno/api/services/api-gateway/endpoints/agents"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/timeutil"
)

func AgentRunPresenter(r *pb.AgentRunInfo) apiresource.AgentRun {
	return AgentRunPresenterWithRole(r, nil)
}

func AgentRunPresenterWithRole(r *pb.AgentRunInfo, role *resolvedRole) apiresource.AgentRun {
	if r == nil {
		return apiresource.AgentRun{}
	}

	run := apiresource.AgentRun{
		ID:                r.Id,
		Object:            constants.ObjectTypeAgentRun,
		Status:            constants.AgentRunStatus(r.StatusCode),
		TriggerType:       constants.AgentTriggerType(r.TriggerType),
		ErrorMessage:      ptrStringOrNil(r.ErrorMessage),
		DurationMs:        ptrInt32OrNil(r.DurationMs),
		TotalInputTokens:  ptrInt64OrNil(r.TotalInputTokens),
		TotalOutputTokens: ptrInt64OrNil(r.TotalOutputTokens),
		TriggeredBy:       lightActorPresenter(r),
		StartedAt:         timeutil.TimestampToTimePtr(r.StartedAt),
		CompletedAt:       timeutil.TimestampToTimePtr(r.CompletedAt),
		CreatedAt:         timeutil.TimestampToTime(r.CreatedAt),
		UpdatedAt:         timeutil.TimestampToTime(r.UpdatedAt),
	}

	if r.Input != "" {
		run.Input = json.RawMessage(r.Input)
	}
	if r.Output != "" {
		run.Output = json.RawMessage(r.Output)
	}

	return run
}

func stashAgentRunMeta(ctx context.Context, run *apiresource.AgentRun, r *pb.AgentRunInfo, role *resolvedRole) {
	if r == nil {
		return
	}

	meta := resourcekit.GetLoadMeta(ctx)

	if len(r.Actions) > 0 {
		actions := make([]apiresource.AgentAction, len(r.Actions))
		for i, a := range r.Actions {
			actions[i] = agentActionPresenter(a, r.Id, timeutil.TimestampToTime(r.CreatedAt), timeutil.TimestampToTime(r.UpdatedAt))
		}
		meta.Set(constants.ObjectTypeAgentRun, run.ID, "actions", apiresource.NewList(actions, apiresource.PageInfo{}))
	}

	if r.Definition != nil {
		var agentRole *agentep.ResolvedRole
		if role != nil {
			agentRole = &agentep.ResolvedRole{
				Name:     role.Name,
				RoleType: role.RoleType,
			}
		}
		def := agentep.AgentDefinitionPresenter(r.Definition)
		meta.Set(constants.ObjectTypeAgentRun, run.ID, "definition", &def)
		agentep.StashAgentDefinitionMeta(meta, r.Definition, agentRole)
	}

	if len(r.Steps) > 0 {
		steps := make([]apiresource.AgentRunStep, len(r.Steps))
		for i, s := range r.Steps {
			steps[i] = AgentRunStepPresenter(s)
		}
		meta.Set(constants.ObjectTypeAgentRun, run.ID, "steps", apiresource.NewList(steps, apiresource.PageInfo{}))
	}
}

func AgentRunStepPresenter(s *pb.AgentRunStepInfo) apiresource.AgentRunStep {
	if s == nil {
		return apiresource.AgentRunStep{}
	}

	step := apiresource.AgentRunStep{
		ID:        s.Id,
		Object:    constants.ObjectTypeAgentRunStep,
		StepType:  s.StepType,
		Title:     s.Title,
		Content:   ptrStringOrNil(s.Content),
		Sequence:  s.Sequence,
		Actor:     lightStepActorPresenter(s),
		CreatedAt: timeutil.TimestampToTime(s.CreatedAt),
	}

	if s.DurationMs != 0 {
		d := s.DurationMs
		step.DurationMs = &d
	}

	if s.MetadataJson != "" && s.MetadataJson != "{}" {
		step.Metadata = json.RawMessage(s.MetadataJson)
	}

	return step
}

func AgentActionPresenter(a *pb.AgentActionInfo, agentRunID string) apiresource.AgentAction {
	return agentActionPresenter(a, agentRunID, time.Time{}, time.Time{})
}

func agentActionPresenter(a *pb.AgentActionInfo, agentRunID string, runCreatedAt, runUpdatedAt time.Time) apiresource.AgentAction {
	if a == nil {
		return apiresource.AgentAction{}
	}

	runID := a.AgentRunId
	if runID == "" {
		runID = agentRunID
	}
	if runCreatedAt.IsZero() {
		runCreatedAt = timeutil.TimestampToTime(a.CreatedAt)
	}
	if runUpdatedAt.IsZero() {
		runUpdatedAt = timeutil.TimestampToTime(a.UpdatedAt)
	}

	run := &apiresource.AgentRun{
		ID:          runID,
		Object:      constants.ObjectTypeAgentRun,
		Status:      constants.AgentRunStatusPending,
		TriggerType: constants.AgentTriggerTypeManual,
		CreatedAt:   runCreatedAt,
		UpdatedAt:   runUpdatedAt,
	}
	if a.RunStatusCode != nil {
		run.Status = constants.AgentRunStatus(*a.RunStatusCode)
	}
	if a.RunTriggerType != nil {
		run.TriggerType = constants.AgentTriggerType(*a.RunTriggerType)
	}

	action := apiresource.AgentAction{
		ID:             a.Id,
		Object:         constants.ObjectTypeAgentAction,
		Run:            run,
		ToolSlug:       constants.ToolSlug(a.ToolSlug),
		Status:         constants.AgentActionStatus(a.StatusCode),
		Label:          ptrStringOrNil(a.Label),
		Description:    ptrStringOrNil(a.Description),
		ErrorMessage:   ptrStringOrNil(a.ErrorMessage),
		Entity:         entityPresenter(a.EntityType, a.EntityId),
		RequiresReview: a.RequiresReview,
		ReviewedBy:     reviewedByPresenter(a),
		ReviewedAt:     timeutil.TimestampToTimePtr(a.ReviewedAt),
		ExecutedAt:     timeutil.TimestampToTimePtr(a.ExecutedAt),
		CreatedAt:      timeutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:      timeutil.TimestampToTime(a.UpdatedAt),
	}

	if a.Input != "" {
		action.Input = json.RawMessage(a.Input)
	}
	if a.Output != "" {
		action.Output = json.RawMessage(a.Output)
	}

	return action
}

func lightStepActorPresenter(s *pb.AgentRunStepInfo) *apiresource.Actor {
	if s.ActorId == "" || s.ActorType == "" {
		return nil
	}
	return apiresource.NewActor(
		s.ActorId,
		constants.ActorType(s.ActorType),
		ptrStringOrNil(s.ActorName),
		nil,
	)
}

func lightActorPresenter(r *pb.AgentRunInfo) *apiresource.Actor {
	if r.TriggeredByActorId == "" || r.TriggeredByIdentityType == "" {
		return nil
	}
	return apiresource.NewActor(
		r.TriggeredByActorId,
		constants.ActorType(r.TriggeredByIdentityType),
		ptrStringOrNil(r.TriggeredByActorName),
		nil,
	)
}

func ptrStringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrInt32OrNil(v int32) *int32 {
	if v == 0 {
		return nil
	}
	return &v
}

func ptrInt64OrNil(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func entityPresenter(entityType, entityID string) *apiresource.Entity {
	if entityType == "" || entityID == "" {
		return nil
	}
	return apiresource.NewEntity(entityID, constants.ObjectType(entityType), nil, nil)
}

func reviewedByPresenter(a *pb.AgentActionInfo) *apiresource.Actor {
	if a.ReviewedBy == "" {
		return nil
	}
	return apiresource.NewActor(
		a.ReviewedBy,
		constants.ActorType(a.ReviewedByActorType),
		ptrStringOrNil(a.ReviewedByActorName),
		nil,
	)
}
