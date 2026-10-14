package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/apikit/apierror"
)

// validateSettlementAllocations checks that every transaction and invoice a settlement
// applies belongs to the account making the settlement, that money is only applied from
// transactions whose funds have arrived, and that each allocation names exactly one
// transaction: an existing one, or one of the settlement's new transactions.
//
// transaction_allocation carries no foreign keys, so nothing below this layer would object
// to an allocation naming a transaction that does not exist — or one belonging to another
// tenant. Either way the invoice ends up credited with money the account never received,
// and the row looks entirely ordinary afterwards.
//
// Each distinct ID is read once, so settling many lines against one invoice costs one read
// rather than one per line.
func validateSettlementAllocations(ctx context.Context, repos domain.RepoFactory, accountID string, params domain.CreateSettlementParams) *apierror.APIError {
	transactionRepo := repos.NewTransactionRepo()
	invoiceRepo := repos.NewInvoiceRepo()

	newTransactions := make(map[string]bool, len(params.NewTransactions))
	for _, nt := range params.NewTransactions {
		if nt.Key == "" {
			return apierror.NewFieldError("new_transactions", apierror.CodeInvalidFormat, "Each new transaction needs a key.")
		}
		if _, dup := newTransactions[nt.Key]; dup {
			return apierror.NewFieldError("new_transactions", apierror.CodeInvalidFormat, "New transaction keys must be unique.")
		}
		newTransactions[nt.Key] = false
		exists, apiErr := transactionRepo.CustomerExists(ctx, accountID, nt.CustomerID)
		if apiErr != nil {
			return apiErr
		}
		if !exists {
			return apierror.NewNotFoundError("Customer not found.")
		}
	}

	seenTransactions := make(map[string]struct{}, len(params.Allocations))
	seenInvoices := make(map[string]struct{}, len(params.Allocations))

	for _, alloc := range params.Allocations {
		switch {
		case (alloc.TransactionID == "") == (alloc.TransactionKey == ""):
			return apierror.NewFieldError("allocations", apierror.CodeInvalidFormat, "Each allocation names either a transaction_id or a transaction_key.")
		case alloc.TransactionKey != "":
			if _, ok := newTransactions[alloc.TransactionKey]; !ok {
				return apierror.NewFieldError("allocations", apierror.CodeInvalidFormat, "An allocation names a transaction_key that is not among new_transactions.")
			}
			newTransactions[alloc.TransactionKey] = true
		default:
			if _, done := seenTransactions[alloc.TransactionID]; !done {
				seenTransactions[alloc.TransactionID] = struct{}{}
				tx, apiErr := transactionRepo.Get(ctx, accountID, alloc.TransactionID)
				if apiErr != nil {
					return apiErr
				}
				if tx.FundsReceivedAt == nil {
					return apierror.NewFieldError("allocations", apierror.CodeInvalidFormat, "Cannot settle transactions that have not yet been received as cash.")
				}
			}
		}

		if _, err := decimal.NewFromString(alloc.Amount); err != nil {
			return apierror.NewFieldError("allocations", apierror.CodeInvalidFormat, "Allocation amount must be a number.")
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

	for _, used := range newTransactions {
		if !used {
			return apierror.NewFieldError("new_transactions", apierror.CodeInvalidFormat, "Every new transaction must be drawn on by an allocation.")
		}
	}

	return nil
}

// newSettlementTransactionAmounts sums, for each new transaction, the allocations drawn from it, and
// finds the first of them, whose date the transaction takes.
func newSettlementTransactionAmounts(allocs []domain.CreateSettlementAllocationParams) (map[string]decimal.Decimal, map[string]domain.CreateSettlementAllocationParams) {
	amounts := map[string]decimal.Decimal{}
	first := map[string]domain.CreateSettlementAllocationParams{}
	for _, a := range allocs {
		if a.TransactionKey == "" {
			continue
		}
		amounts[a.TransactionKey] = amounts[a.TransactionKey].Add(decimal.RequireFromString(a.Amount))
		if _, ok := first[a.TransactionKey]; !ok {
			first[a.TransactionKey] = a
		}
	}
	return amounts, first
}
