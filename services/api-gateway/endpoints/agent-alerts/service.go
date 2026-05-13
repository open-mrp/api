package agentalertep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AgentAlertSvc interface {
	ListAlerts(ctx context.Context, req *ListAlertsRequest) (*apiresource.List[apiresource.AgentAlert], *apierror.APIError)
	GetAlert(ctx context.Context, req *RetrieveAlertRequest) (*apiresource.AgentAlert, *apierror.APIError)
	AcknowledgeAlert(ctx context.Context, req *AcknowledgeAlertRequest) (*apiresource.AgentAlert, *apierror.APIError)
}

type AgentAlertSvcConfig struct {
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

	return AgentAlertListPresenter(resp), nil
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

	result := AgentAlertPresenter(resp.Alert)
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

	result := AgentAlertPresenter(resp.Alert)
	return &result, nil
}
