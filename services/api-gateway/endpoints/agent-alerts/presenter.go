package agentalertep

import (
	"encoding/json"

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

func AgentAlertListPresenter(resp *pb.ListAgentAlertsResponse) *apiresource.List[apiresource.AgentAlert] {
	if resp == nil {
		return apiresource.NewList[apiresource.AgentAlert](nil, apiresource.PageInfo{})
	}

	alerts := make([]apiresource.AgentAlert, len(resp.Alerts))
	for i, a := range resp.Alerts {
		alerts[i] = AgentAlertPresenter(a)
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

	return apiresource.NewList(alerts, pageInfo)
}

func ptrStringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
