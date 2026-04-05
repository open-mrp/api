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

	listTransactionsEndpoint := (&transactionep.ListTransactionsEndpoint{}).Materialize().WithService(inner, transactionSvc)
	getTransactionEndpoint := (&transactionep.GetTransactionEndpoint{}).Materialize().WithService(inner, transactionSvc)
	createTransactionEndpoint := (&transactionep.CreateTransactionEndpoint{}).Materialize().WithService(inner, transactionSvc)
	updateTransactionEndpoint := (&transactionep.UpdateTransactionEndpoint{}).Materialize().WithService(inner, transactionSvc)
	deleteTransactionEndpoint := (&transactionep.DeleteTransactionEndpoint{}).Materialize().WithService(inner, transactionSvc)
	listAccountTransactionsEndpoint := (&transactionep.ListAccountTransactionsEndpoint{}).Materialize().WithService(inner, transactionSvc)
	listTransactionTypesEndpoint := (&transactionep.ListTransactionTypesEndpoint{}).Materialize().WithService(inner, transactionSvc)
	listTransactionMethodsEndpoint := (&transactionep.ListTransactionMethodsEndpoint{}).Materialize().WithService(inner, transactionSvc)
	listAdjustmentTypesEndpoint := (&transactionep.ListAdjustmentTypesEndpoint{}).Materialize().WithService(inner, transactionSvc)

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
