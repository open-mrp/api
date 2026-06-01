package agentalertep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/agent"
)

func TestAgentAlertPresenter(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	alert := &pb.AgentAlertInfo{
		Id:                      "agal_01abc",
		AccountId:               "ac_01abc",
		AgentRunId:              "agrun_01abc",
		AgentActionId:           "agact_01abc",
		SeverityCode:            "warning",
		StatusCode:              "open",
		Title:                   "Critical Stock Level",
		Message:                 "Raw Steel is below threshold.",
		MetadataJson:            `{"level":"critical"}`,
		AcknowledgedAt:          now,
		AcknowledgedBy:          "us_01abc",
		AcknowledgedByActorType: "user",
		AcknowledgedByActorName: "John Doe",
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	result := agentAlertFromProto(alert)
	resourcetest.ValidateResourceStruct(t, "AgentAlert", result)
}
