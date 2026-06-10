package httpgroup

import (
	"fmt"

	transactionep "github.com/augno/api/services/api-gateway/endpoints/transactions"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type TransactionsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type TransactionsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *TransactionsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("transactions endpoint group: core client is required")
	}
	return nil
}

func (*TransactionsEndpointGroup) Materialize(config *TransactionsEndpointGroupConfig) *TransactionsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	transactionSvc := transactionep.NewTransactionSvc(&transactionep.TransactionSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Transactions",
		Description:  "Create, view, update, and delete transactions.",
		ResourceType: &apiresource.TransactionDetail{},
	}

	listTransactionsEndpoint := apiendpoint.From(&transactionep.ListTransactionsEndpoint{}).WithService(inner, transactionSvc)
	getTransactionEndpoint := apiendpoint.From(&transactionep.RetrieveTransactionEndpoint{}).WithService(inner, transactionSvc)
	createTransactionEndpoint := apiendpoint.From(&transactionep.CreateTransactionEndpoint{}).WithService(inner, transactionSvc)
	updateTransactionEndpoint := apiendpoint.From(&transactionep.UpdateTransactionEndpoint{}).WithService(inner, transactionSvc)
	deleteTransactionEndpoint := apiendpoint.From(&transactionep.DeleteTransactionEndpoint{}).WithService(inner, transactionSvc)
	listAccountTransactionsEndpoint := apiendpoint.From(&transactionep.ListAccountTransactionsEndpoint{}).WithService(inner, transactionSvc)
	listTransactionTypesEndpoint := apiendpoint.From(&transactionep.ListTransactionTypesEndpoint{}).WithService(inner, transactionSvc)
	listTransactionMethodsEndpoint := apiendpoint.From(&transactionep.ListTransactionMethodsEndpoint{}).WithService(inner, transactionSvc)
	listAdjustmentTypesEndpoint := apiendpoint.From(&transactionep.ListAdjustmentTypesEndpoint{}).WithService(inner, transactionSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listTransactionsEndpoint,
		getTransactionEndpoint,
		createTransactionEndpoint,
		updateTransactionEndpoint,
		deleteTransactionEndpoint,
		listAccountTransactionsEndpoint,
		listTransactionTypesEndpoint,
		listTransactionMethodsEndpoint,
		listAdjustmentTypesEndpoint,
	}

	return &TransactionsEndpointGroup{inner}
}
