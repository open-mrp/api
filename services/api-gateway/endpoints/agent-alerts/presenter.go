package agentalertep

import (
	"context"
	"encoding/json"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/timeutil"
)

func AgentAlertPresenter(a *pb.AgentAlertInfo) apiresource.AgentAlert {
	if a == nil {
		return apiresource.AgentAlert{}
	}

	alert := apiresource.AgentAlert{
		ID:        a.Id,
		Object:    constants.ObjectTypeAgentAlert,
		Severity:  constants.AgentAlertSeverity(a.SeverityCode),
		Status:    constants.AgentAlertStatus(a.StatusCode),
		Title:     a.Title,
		Message:   ptrStringOrNil(a.Message),
		CreatedAt: timeutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: timeutil.TimestampToTime(a.UpdatedAt),
	}

	// Attach stub Run and Action resources when the alert carries the
	// corresponding IDs. These stubs satisfy `?include=run` /
	// `?include=action`; the collapse middleware nulls them when not
	// requested.
	if a.AgentRunId != "" {
		alert.Run = &apiresource.AgentRun{
			ID:          a.AgentRunId,
			Object:      constants.ObjectTypeAgentRun,
			Status:      constants.AgentRunStatus(a.GetRunStatusCode()),
			TriggerType: constants.AgentTriggerType(a.GetRunTriggerType()),
			CreatedAt:   timeutil.TimestampToTime(a.GetRunCreatedAt()),
			UpdatedAt:   timeutil.TimestampToTime(a.GetRunUpdatedAt()),
		}
	}
	if a.AgentActionId != "" {
		alert.Action = &apiresource.AgentAction{
			ID:        a.AgentActionId,
			Object:    constants.ObjectTypeAgentAction,
			ToolSlug:  constants.ToolSlug(a.GetActionToolSlug()),
			Status:    constants.AgentActionStatus(a.GetActionStatusCode()),
			CreatedAt: timeutil.TimestampToTime(a.GetActionCreatedAt()),
			UpdatedAt: timeutil.TimestampToTime(a.GetActionUpdatedAt()),
		}
		if a.AgentRunId != "" {
			alert.Action.Run = alert.Run
		}
	}

	if a.AcknowledgedAt != "" {
		t := timeutil.TimestampToTime(a.AcknowledgedAt)
		alert.AcknowledgedAt = &t
	}
	if a.AcknowledgedBy != "" {
		alert.AcknowledgedBy = apiresource.NewActor(
			a.AcknowledgedBy,
			constants.ActorType(a.AcknowledgedByActorType),
			ptrStringOrNil(a.AcknowledgedByActorName),
			nil,
		)
	}
	if a.MetadataJson != "" && a.MetadataJson != "{}" {
		alert.Metadata = json.RawMessage(a.MetadataJson)
	}

	return alert
}

func AgentAlertListPresenter(ctx context.Context, resp *pb.ListAgentAlertsResponse) *apiresource.List[apiresource.AgentAlert] {
	if resp == nil {
		return apiresource.NewList[apiresource.AgentAlert](nil, apiresource.PageInfo{})
	}

	alerts := make([]apiresource.AgentAlert, len(resp.Alerts))
	for i, a := range resp.Alerts {
		alerts[i] = AgentAlertPresenter(a)
	}

	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}

	return apiresource.NewList(alerts, pageInfo)
}

func ptrStringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
