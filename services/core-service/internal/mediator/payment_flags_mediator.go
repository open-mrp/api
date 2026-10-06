package mediator

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

type PaymentFlagsMedConfig struct {
	// Repos (required) is the repository factory; inside a transaction, the transaction's.
	Repos domain.RepoFactory
}

type paymentFlagsMedImpl struct {
	repos domain.RepoFactory
}

func NewPaymentFlagsMed(config *PaymentFlagsMedConfig) domain.PaymentFlagsMed {
	return &paymentFlagsMedImpl{repos: config.Repos}
}

func (m *paymentFlagsMedImpl) Enqueue(ctx context.Context, accountID string, transactionIDs, invoiceIDs []string) *apierror.APIError {
	transactionIDs, invoiceIDs = uniqueIDs(transactionIDs), uniqueIDs(invoiceIDs)
	if len(transactionIDs) == 0 && len(invoiceIDs) == 0 {
		return nil
	}
	payload, err := json.Marshal(domain.RecomputePaymentFlagsEvent{
		AccountID:      accountID,
		TransactionIDs: transactionIDs,
		InvoiceIDs:     invoiceIDs,
	})
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal recompute payment flags event.")
	}
	msg := contracts.AmqpMessage{Data: payload}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	if _, err := m.repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreCmdRecomputePaymentFlags),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreCmdRecomputePaymentFlags),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for recompute payment flags.")
	}
	return nil
}

func (m *paymentFlagsMedImpl) Recompute(ctx context.Context, accountID string, transactionIDs, invoiceIDs []string) *apierror.APIError {
	transactionIDs, invoiceIDs = uniqueIDs(transactionIDs), uniqueIDs(invoiceIDs)
	repo := m.repos.NewSettlementRepo()
	if apiErr := repo.LockPaymentFlagRows(ctx, accountID, transactionIDs, invoiceIDs); apiErr != nil {
		return apiErr
	}

	txTotals, apiErr := repo.GetTransactionAllocationTotals(ctx, accountID, transactionIDs)
	if apiErr != nil {
		return apiErr
	}
	var allocated, open []string
	for _, t := range txTotals {
		amount, err := decimal.NewFromString(t.Total)
		if err != nil {
			return apierror.NewInternalError(err, "Transaction amount is not a number.")
		}
		spent, err := decimal.NewFromString(t.Allocated)
		if err != nil {
			return apierror.NewInternalError(err, "Transaction allocations are not a number.")
		}
		if domain.TransactionFullyAllocated(amount, spent) {
			allocated = append(allocated, t.ID)
		} else {
			open = append(open, t.ID)
		}
	}
	if apiErr := repo.UpdateTransactionsFullyAllocated(ctx, accountID, allocated, true); apiErr != nil {
		return apiErr
	}
	if apiErr := repo.UpdateTransactionsFullyAllocated(ctx, accountID, open, false); apiErr != nil {
		return apiErr
	}

	invoiceTotals, apiErr := repo.GetInvoicePaymentTotals(ctx, accountID, invoiceIDs)
	if apiErr != nil {
		return apiErr
	}
	for _, inv := range invoiceTotals {
		invoiced, err := decimal.NewFromString(inv.Total)
		if err != nil {
			return apierror.NewInternalError(err, "Invoice total is not a number.")
		}
		paid, err := decimal.NewFromString(inv.Allocated)
		if err != nil {
			return apierror.NewInternalError(err, "Invoice allocations are not a number.")
		}
		isPaidInFull, isOverPaid := domain.InvoicePaymentFlagsFor(invoiced, paid)
		// What has been applied to the invoice decides the flag, over a value someone set by hand. When that
		// overturns their value, they are told, and the flag is no longer theirs.
		overturned := inv.MarkedByID != nil && inv.IsPaidInFull != isPaidInFull
		if apiErr := repo.UpdateInvoicePaymentStatus(ctx, accountID, inv.ID, isPaidInFull, isOverPaid, overturned); apiErr != nil {
			return apiErr
		}
		if overturned {
			if apiErr := m.notifyMarkOverturned(ctx, accountID, inv, isPaidInFull, invoiced.Sub(paid)); apiErr != nil {
				return apiErr
			}
		}
	}
	return nil
}

// notifyMarkOverturned tells the person who set an invoice's paid-in-full flag by hand that recalculating
// its payments set it the other way. The alert rides the outbox in the recalculation's transaction.
func (m *paymentFlagsMedImpl) notifyMarkOverturned(ctx context.Context, accountID string, inv domain.InvoicePaymentTotals, isPaidInFull bool, owed decimal.Decimal) *apierror.APIError {
	title := "Invoice " + inv.Number + " is no longer marked paid"
	body := "You marked it paid in full, but recalculating its payments found $" + owed.StringFixed(2) + " still owed."
	if isPaidInFull {
		title = "Invoice " + inv.Number + " is now marked paid"
		body = "You marked it unpaid, but recalculating its payments found it paid in full."
	}
	data := messaging.AlertFanoutData{
		AccountID:        accountID,
		Category:         string(constants.NotificationCategoryInvoicePaymentStatusChanged),
		Kind:             "alert",
		Title:            title,
		Body:             body,
		LinkResourceType: string(constants.ObjectTypeInvoice),
		LinkResourceID:   inv.ID,
		Priority:         string(constants.NotificationPriorityNormal),
		SenderType:       string(constants.NotificationSenderTypeSystem),
		RecipientUserIDs: []string{*inv.MarkedByID},
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal invoice payment status alert.")
	}
	msg := contracts.AmqpMessage{Data: dataJSON}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	if _, err := m.repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdFanout),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdFanout),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to enqueue invoice payment status alert.")
	}
	return nil
}

func uniqueIDs(ids []string) []string {
	out := slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return id == "" })
	slices.Sort(out)
	return slices.Compact(out)
}
