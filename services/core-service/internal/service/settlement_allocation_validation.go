package service

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

// validateSettlementAllocations checks that every transaction and invoice a settlement
// applies belongs to the account making the settlement.
//
// transaction_allocation carries no foreign keys, so nothing below this layer would object
// to an allocation naming a transaction that does not exist — or one belonging to another
// tenant. Either way the invoice ends up credited with money the account never received,
// and the row looks entirely ordinary afterwards.
//
// Each distinct ID is read once, so settling many lines against one invoice costs one read
// rather than one per line.
func validateSettlementAllocations(ctx context.Context, repos domain.RepoFactory, accountID string, allocations []domain.CreateSettlementAllocationParams) *apierror.APIError {
	transactionRepo := repos.NewTransactionRepo()
	invoiceRepo := repos.NewInvoiceRepo()

	seenTransactions := make(map[string]struct{}, len(allocations))
	seenInvoices := make(map[string]struct{}, len(allocations))

	for _, alloc := range allocations {
		if _, done := seenTransactions[alloc.TransactionID]; !done {
			seenTransactions[alloc.TransactionID] = struct{}{}
			if _, apiErr := transactionRepo.Get(ctx, accountID, alloc.TransactionID); apiErr != nil {
				return apiErr
			}
		}

		if _, done := seenInvoices[alloc.InvoiceID]; !done {
			seenInvoices[alloc.InvoiceID] = struct{}{}
			if _, apiErr := invoiceRepo.Get(ctx, domain.GetInvoiceParams{
				AccountID: accountID,
				InvoiceID: alloc.InvoiceID,
			}); apiErr != nil {
				return apiErr
			}
		}
	}

	return nil
}
