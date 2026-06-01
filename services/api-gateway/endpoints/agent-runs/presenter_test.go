package agentrunep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/agent"
)

func TestAgentRunPresenter(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	run := &pb.AgentRunInfo{
		Id:                      "agrun_01abc",
		AccountId:               "ac_01abc",
		AgentDefinitionId:       "agdf_01abc",
		StatusCode:              "completed",
		TriggerType:             "manual",
		Input:                   `{"message":"Process order"}`,
		Output:                  `{"response":"Done"}`,
		ErrorMessage:            "some error",
		StartedAt:               now,
		CompletedAt:             now,
		DurationMs:              1250,
		TotalInputTokens:        500,
		TotalOutputTokens:       300,
		TriggeredByActorId:      "us_01abc",
		TriggeredByIdentityType: "user",
		TriggeredByActorName:    "John Doe",
		CreatedAt:               now,
		UpdatedAt:               now,
		Actions: []*pb.AgentActionInfo{
			{
				Id:                  "agact_01abc",
				AgentRunId:          "agrun_01abc",
				ToolSlug:            "search_products",
				StatusCode:          "executed",
				Label:               "Search",
				Description:         "Search products",
				Input:               `{"query":"steel"}`,
				Output:              `{"results":[]}`,
				EntityType:          "account",
				EntityId:            "ac_01abc",
				RequiresReview:      true,
				ReviewedAt:          now,
				ReviewedBy:          "us_01abc",
				ReviewedByActorType: "user",
				ReviewedByActorName: "John Doe",
				ExecutedAt:          now,
				CreatedAt:           now,
				UpdatedAt:           now,
			},
		},
		Steps: []*pb.AgentRunStepInfo{
			{
				Id:           "agrunev_01abc",
				AgentRunId:   "agrun_01abc",
				StepType:     "trigger_received",
				Title:        "Run triggered",
				Content:      "Process order #1234",
				Sequence:     1,
				DurationMs:   100,
				MetadataJson: `{"key":"value"}`,
				CreatedAt:    now,
				ActorId:      "us_01abc",
				ActorType:    "user",
			},
		},
		Definition: &pb.AgentDefinitionInfo{
			Id:             "agdf_01abc",
			Name:           "Email Order Agent",
			Slug:           "email_order",
			Description:    "Processes incoming emails.",
			DefinitionType: "system",
			CategoryCode:   "order_processing",
			TriggerType:    "event",
			IsEditable:     true,
			AccountStatus: &pb.AgentAccountStatusInfo{
				StatusCode: "active",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	result := AgentRunPresenter(run)
	resourcetest.ValidateResourceStruct(t, "AgentRun", result)
	resourcetest.ValidateExpandableStubs(t, "AgentRun", result)
}

func TestAgentActionPresenter(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	runStatusCode := "completed"
	runTriggerType := "manual"

	action := &pb.AgentActionInfo{
		Id:                  "agact_01abc",
		AgentRunId:          "agrun_01abc",
		RunStatusCode:       &runStatusCode,
		RunTriggerType:      &runTriggerType,
		ToolSlug:            "search_products",
		StatusCode:          "executed",
		Label:               "Search",
		Description:         "Search products",
		Input:               `{"query":"steel"}`,
		Output:              `{"results":[]}`,
		EntityType:          "account",
		EntityId:            "ac_01abc",
		RequiresReview:      true,
		ReviewedAt:          now,
		ReviewedBy:          "us_01abc",
		ReviewedByActorType: "user",
		ReviewedByActorName: "John Doe",
		ExecutedAt:          now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	result := AgentActionPresenter(action, "agrun_01abc")
	resourcetest.ValidateResourceStruct(t, "AgentAction", result)
	resourcetest.ValidateExpandableStubs(t, "AgentAction", result)
}

func TestAgentRunStepPresenter(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	step := &pb.AgentRunStepInfo{
		Id:           "agrunev_01abc",
		AgentRunId:   "agrun_01abc",
		StepType:     "trigger_received",
		Title:        "Run triggered",
		Content:      "Process order #1234",
		Sequence:     1,
		DurationMs:   100,
		MetadataJson: `{"key":"value"}`,
		CreatedAt:    now,
		ActorId:      "us_01abc",
		ActorType:    "user",
	}

	result := AgentRunStepPresenter(step)
	resourcetest.ValidateResourceStruct(t, "AgentRunStep", result)
}
