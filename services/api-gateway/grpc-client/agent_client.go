package grpcclient

import (
	"context"

	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/agent"
)

const agentServiceName = "agent-service"

type AgentServiceClient struct {
	Client   pb.AgentServiceClient
	grpcConn *contracts.GRPCClientConn
}

func NewAgentServiceClientWithURL(url string) (*AgentServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: agentServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &AgentServiceClient{
		Client:   pb.NewAgentServiceClient(grpcConn.Conn()),
		grpcConn: grpcConn,
	}, nil
}

func (c *AgentServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *AgentServiceClient) Close() error {
	return c.grpcConn.Close()
}
