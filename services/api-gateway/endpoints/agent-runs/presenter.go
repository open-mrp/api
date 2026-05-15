package agentrunep

import (
	"encoding/json"

	agentep "github.com/augno/api/services/api-gateway/endpoints/agents"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
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

	if len(r.Actions) > 0 {
		actions := make([]apiresource.AgentAction, len(r.Actions))
		for i, a := range r.Actions {
			actions[i] = AgentActionPresenter(a, r.Id)
		}
		run.Actions = apiresource.NewList(actions, apiresource.PageInfo{})
	}

	if r.Definition != nil {
		var agentRole *agentep.ResolvedRole
		if role != nil {
			agentRole = &agentep.ResolvedRole{
				Name:     role.Name,
				RoleType: role.RoleType,
			}
		}
		def := agentep.AgentDefinitionPresenter(r.Definition, agentRole)
		run.Definition = &def
	}

	if len(r.Steps) > 0 {
		steps := make([]apiresource.AgentRunStep, len(r.Steps))
		for i, s := range r.Steps {
			steps[i] = AgentRunStepPresenter(s)
		}
		run.Steps = apiresource.NewList(steps, apiresource.PageInfo{})
	}

	return run
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
	if a == nil {
		return apiresource.AgentAction{}
	}

	runID := a.AgentRunId
	if runID == "" {
		runID = agentRunID
	}

	run := &apiresource.AgentRun{
		ID:     runID,
		Object: constants.ObjectTypeAgentRun,
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

func AgentRunListPresenter(resp *pb.ListRunsResponse) *apiresource.List[apiresource.AgentRun] {
	if resp == nil {
		return apiresource.NewList[apiresource.AgentRun](nil, apiresource.PageInfo{})
	}

	runs := make([]apiresource.AgentRun, len(resp.Runs))
	for i, r := range resp.Runs {
		runs[i] = AgentRunPresenter(r)
	}

	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = apiresource.PageInfo{
			NextCursor:  resp.PageInfo.NextCursor,
			PrevCursor:  resp.PageInfo.PrevCursor,
			HasNextPage: resp.PageInfo.HasNextPage,
			HasPrevPage: resp.PageInfo.HasPrevPage,
		}
	}

	return apiresource.NewList(runs, pageInfo)
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
