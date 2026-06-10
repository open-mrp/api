package agentalertep

import (
	"context"
	"fmt"

	"encoding/json"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/timeutil"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AgentAlertSvc interface {
	ListAlerts(ctx context.Context, req *ListAlertsRequest) (*apiresource.List[apiresource.AgentAlert], *apierror.APIError)
	GetAlert(ctx context.Context, req *RetrieveAlertRequest) (*apiresource.AgentAlert, *apierror.APIError)
	AcknowledgeAlert(ctx context.Context, req *AcknowledgeAlertRequest) (*apiresource.AgentAlert, *apierror.APIError)
}

type AgentAlertSvcConfig struct {
	// AgentClient (required) is the agent-service gRPC client.
	AgentClient pb.AgentServiceClient
}

type agentAlertSvcImpl struct {
	agentClient pb.AgentServiceClient
}

var alertSvcTracer = tracing.GetTracer("api-gateway.endpoints.agent_alerts.service")

func (c *AgentAlertSvcConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent alert endpoint service: agent client is required")
	}
	return nil
}

func NewAgentAlertSvc(config *AgentAlertSvcConfig) AgentAlertSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &agentAlertSvcImpl{
		agentClient: config.AgentClient,
	}
}

func (m *agentAlertSvcImpl) ListAlerts(ctx context.Context, req *ListAlertsRequest) (*apiresource.List[apiresource.AgentAlert], *apierror.APIError) {
	pbReq := &pb.ListAgentAlertsRequest{
		Limit:  req.Limit,
		Cursor: req.Cursor,
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}
	if req.Severity != nil {
		pbReq.SeverityCode = string(*req.Severity)
	}
	if req.Status != nil {
		pbReq.StatusCode = string(*req.Status)
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, alertSvcTracer, "service.agent_alerts.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAgentAlertsResponse, error) {
			return m.agentClient.ListAgentAlerts(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	alerts := make([]apiresource.AgentAlert, len(resp.Alerts))
	for i, a := range resp.Alerts {
		alerts[i] = agentAlertFromProto(a)
		stashAgentAlertMeta(meta, a)
	}

	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}

	return apiresource.NewList(alerts, pageInfo), nil
}

func (m *agentAlertSvcImpl) GetAlert(ctx context.Context, req *RetrieveAlertRequest) (*apiresource.AgentAlert, *apierror.APIError) {
	pbReq := &pb.GetAgentAlertRequest{
		Id: req.AlertID,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, alertSvcTracer, "service.agent_alerts.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAgentAlertResponse, error) {
			return m.agentClient.GetAgentAlert(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := agentAlertFromProto(resp.Alert)
	stashAgentAlertMeta(meta, resp.Alert)
	return &result, nil
}

func (m *agentAlertSvcImpl) AcknowledgeAlert(ctx context.Context, req *AcknowledgeAlertRequest) (*apiresource.AgentAlert, *apierror.APIError) {
	pbReq := &pb.AcknowledgeAgentAlertRequest{
		Id: req.AlertID,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, alertSvcTracer, "service.agent_alerts.acknowledge", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AcknowledgeAgentAlertResponse, error) {
			return m.agentClient.AcknowledgeAgentAlert(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := agentAlertFromProto(resp.Alert)
	stashAgentAlertMeta(meta, resp.Alert)
	return &result, nil
}

func agentAlertFromProto(a *pb.AgentAlertInfo) apiresource.AgentAlert {
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

func stashAgentAlertMeta(meta *resourcekit.LoadMeta, a *pb.AgentAlertInfo) {
	if a == nil {
		return
	}

	if a.AgentRunId != "" {
		run := &apiresource.AgentRun{
			ID:          a.AgentRunId,
			Object:      constants.ObjectTypeAgentRun,
			Status:      constants.AgentRunStatus(a.GetRunStatusCode()),
			TriggerType: constants.AgentTriggerType(a.GetRunTriggerType()),
			CreatedAt:   timeutil.TimestampToTime(a.GetRunCreatedAt()),
			UpdatedAt:   timeutil.TimestampToTime(a.GetRunUpdatedAt()),
		}
		meta.Set(constants.ObjectTypeAgentAlert, a.Id, "run", run)
	}

	if a.AgentActionId != "" {
		action := &apiresource.AgentAction{
			ID:        a.AgentActionId,
			Object:    constants.ObjectTypeAgentAction,
			ToolSlug:  constants.ToolSlug(a.GetActionToolSlug()),
			Status:    constants.AgentActionStatus(a.GetActionStatusCode()),
			CreatedAt: timeutil.TimestampToTime(a.GetActionCreatedAt()),
			UpdatedAt: timeutil.TimestampToTime(a.GetActionUpdatedAt()),
		}
		if a.AgentRunId != "" {
			action.Run = &apiresource.AgentRun{
				ID:          a.AgentRunId,
				Object:      constants.ObjectTypeAgentRun,
				Status:      constants.AgentRunStatus(a.GetRunStatusCode()),
				TriggerType: constants.AgentTriggerType(a.GetRunTriggerType()),
				CreatedAt:   timeutil.TimestampToTime(a.GetRunCreatedAt()),
				UpdatedAt:   timeutil.TimestampToTime(a.GetRunUpdatedAt()),
			}
		}
		meta.Set(constants.ObjectTypeAgentAlert, a.Id, "action", action)
	}
}

func ptrStringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
