package httpgroup

import (
	"fmt"

	transactionallocationep "github.com/augno/api/services/api-gateway/endpoints/transaction-allocations"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type TransactionAllocationsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type TransactionAllocationsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *TransactionAllocationsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("transaction allocations endpoint group: core client is required")
	}
	return nil
}

func (*TransactionAllocationsEndpointGroup) Materialize(config *TransactionAllocationsEndpointGroupConfig) *TransactionAllocationsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := transactionallocationep.NewTransactionAllocationSvc(&transactionallocationep.TransactionAllocationSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Transaction Allocations",
		Description:  "View, update, and delete transaction allocations, and list open credits.",
		ResourceType: &apiresource.AllocationEntry{},
	}

	listAllocationEntriesEndpoint := apiendpoint.From(&transactionallocationep.ListAllocationEntriesEndpoint{}).WithService(inner, svc)
	updateTransactionAllocationEndpoint := apiendpoint.From(&transactionallocationep.UpdateTransactionAllocationEndpoint{}).WithService(inner, svc)
	deleteTransactionAllocationEndpoint := apiendpoint.From(&transactionallocationep.DeleteTransactionAllocationEndpoint{}).WithService(inner, svc)
	listOpenCreditsEndpoint := apiendpoint.From(&transactionallocationep.ListOpenCreditsEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listAllocationEntriesEndpoint,
		updateTransactionAllocationEndpoint,
		deleteTransactionAllocationEndpoint,
		listOpenCreditsEndpoint,
	}

	return &TransactionAllocationsEndpointGroup{inner}
}
