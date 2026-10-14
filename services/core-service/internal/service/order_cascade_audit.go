package service

import (
	"context"
	"slices"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// Writes one record makes to another on its behalf (a ship stamping its order, an unissue tearing
// down a pick) are audited on the record written, so its own history shows them.

// publishSalesOrderChanges re-reads the order and publishes what changed since before.
func publishSalesOrderChanges(ctx context.Context, repos domain.RepoFactory, accountID string, before *domain.SalesOrder) *apierror.APIError {
	after, apiErr := repos.NewSalesOrderRepo().Get(ctx, accountID, before.ID)
	if apiErr != nil {
		return apiErr
	}
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionUpdate,
		ResourceType:     constants.ObjectTypeSalesOrder,
		ResourceID:       before.ID,
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   before.ID,
		Changes:          audit.ComputeChanges(before, after),
	})
}

// publishPurchaseOrderChanges re-reads the order and publishes what changed since before.
func publishPurchaseOrderChanges(ctx context.Context, repos domain.RepoFactory, accountID string, before *domain.PurchaseOrder) *apierror.APIError {
	after, apiErr := repos.NewPurchaseOrderRepo().Get(ctx, accountID, before.ID)
	if apiErr != nil {
		return apiErr
	}
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypePurchaseOrder,
		ResourceID:   before.ID,
		Changes:      audit.ComputeChanges(before, after),
	})
}

// markInvoiceEmailSent flags the invoice sent and audits the flag.
func markInvoiceEmailSent(ctx context.Context, repos domain.RepoFactory, accountID, invoiceID string) *apierror.APIError {
	repo := repos.NewInvoiceRepo()
	params := domain.GetInvoiceParams{AccountID: accountID, InvoiceID: invoiceID}
	before, apiErr := repo.Get(ctx, params)
	if apiErr != nil {
		return apiErr
	}
	if apiErr := repo.MarkEmailSent(ctx, accountID, invoiceID); apiErr != nil {
		return apiErr
	}
	after, apiErr := repo.Get(ctx, params)
	if apiErr != nil {
		return apiErr
	}
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionUpdate,
		ResourceType:     constants.ObjectTypeInvoice,
		ResourceID:       invoiceID,
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   after.OrderID,
		Changes:          audit.ComputeChanges(before, after),
	})
}

// markSalesOrderAcknowledgementSent flags the order's acknowledgement sent and audits the flag.
func markSalesOrderAcknowledgementSent(ctx context.Context, repos domain.RepoFactory, accountID, salesOrderID string) *apierror.APIError {
	before, apiErr := repos.NewSalesOrderRepo().Get(ctx, accountID, salesOrderID)
	if apiErr != nil {
		return apiErr
	}
	if apiErr := repos.NewSalesOrderRepo().MarkAcknowledgementSent(ctx, accountID, salesOrderID); apiErr != nil {
		return apiErr
	}
	return publishSalesOrderChanges(ctx, repos, accountID, before)
}

// markPurchaseOrderSubmissionSent flags the order's submission sent and audits the flag.
func markPurchaseOrderSubmissionSent(ctx context.Context, repos domain.RepoFactory, accountID, purchaseOrderID string) *apierror.APIError {
	before, apiErr := repos.NewPurchaseOrderRepo().Get(ctx, accountID, purchaseOrderID)
	if apiErr != nil {
		return apiErr
	}
	if apiErr := repos.NewPurchaseOrderRepo().MarkSubmissionSent(ctx, accountID, purchaseOrderID); apiErr != nil {
		return apiErr
	}
	return publishPurchaseOrderChanges(ctx, repos, accountID, before)
}

// publishPickChanges re-reads the pick and publishes what changed since before.
func publishPickChanges(ctx context.Context, repos domain.RepoFactory, accountID string, before *domain.Pick) *apierror.APIError {
	after, apiErr := repos.NewPickRepo().Get(ctx, accountID, before.ID)
	if apiErr != nil {
		return apiErr
	}
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionUpdate,
		ResourceType:     constants.ObjectTypePick,
		ResourceID:       before.ID,
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   before.SalesOrderID,
		Changes:          audit.ComputeChanges(before, after),
	})
}

// publishPickDeleted records a pick removed with its order's teardown.
func publishPickDeleted(ctx context.Context, repos domain.RepoFactory, pick *domain.Pick) *apierror.APIError {
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           constants.AuditActionDelete,
		ResourceType:     constants.ObjectTypePick,
		ResourceID:       pick.ID,
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   pick.SalesOrderID,
		Changes:          audit.ComputeChanges(pick, (*domain.Pick)(nil)),
	})
}

// publishPickLinesDeleted records pick lines removed by a cascade; read them before the delete.
func publishPickLinesDeleted(ctx context.Context, repos domain.RepoFactory, salesOrderID string, lines []*domain.PickLine) *apierror.APIError {
	for _, line := range lines {
		if apiErr := audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionDelete,
			ResourceType:     constants.ObjectTypePickLine,
			ResourceID:       line.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   salesOrderID,
			Changes:          audit.ComputeChanges(line, (*domain.PickLine)(nil)),
		}); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// orderPickLines returns the order's pick and the lines of it that match keep; a nil pick means the order has none.
func orderPickLines(ctx context.Context, repos domain.RepoFactory, accountID, salesOrderID string, keep func(*domain.PickLine) bool) (*domain.Pick, []*domain.PickLine, *apierror.APIError) {
	pickID, apiErr := repos.NewSalesOrderRepo().GetPickID(ctx, salesOrderID)
	if apiErr != nil || pickID == nil {
		return nil, nil, apiErr
	}
	return pickWithLines(ctx, repos, accountID, *pickID, keep)
}

// pickWithLines returns the pick and those of its lines that match keep.
func pickWithLines(ctx context.Context, repos domain.RepoFactory, accountID, pickID string, keep func(*domain.PickLine) bool) (*domain.Pick, []*domain.PickLine, *apierror.APIError) {
	pick, apiErr := repos.NewPickRepo().Get(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	lines, apiErr := repos.NewPickRepo().GetLines(ctx, pickID)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	return pick, slices.DeleteFunc(lines, func(l *domain.PickLine) bool { return !keep(l) }), nil
}

// publishShippingCaseChanges re-reads the shipment's cases and publishes each one's changes since before.
func publishShippingCaseChanges(ctx context.Context, repos domain.RepoFactory, shipmentID, salesOrderID string, before []*domain.ShippingCase) *apierror.APIError {
	after, apiErr := repos.NewShippingCaseRepo().ListByShipment(ctx, shipmentID)
	if apiErr != nil {
		return apiErr
	}
	prior := make(map[string]*domain.ShippingCase, len(before))
	for _, c := range before {
		prior[c.ID] = c
	}
	for _, c := range after {
		old, ok := prior[c.ID]
		if !ok {
			continue
		}
		if apiErr := audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionUpdate,
			ResourceType:     constants.ObjectTypeShippingCase,
			ResourceID:       c.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   salesOrderID,
			Changes:          audit.ComputeChanges(old, c),
		}); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// publishReceivingOrderLinesCreated records receiving lines that appeared on the order since before.
func publishReceivingOrderLinesCreated(ctx context.Context, repos domain.RepoFactory, receivingOrderID, purchaseOrderID string, before []*domain.ReceivingOrderLine) *apierror.APIError {
	after, apiErr := repos.NewReceivingOrderRepo().ListLines(ctx, receivingOrderID)
	if apiErr != nil {
		return apiErr
	}
	for _, line := range after {
		if slices.ContainsFunc(before, func(b *domain.ReceivingOrderLine) bool { return b.ID == line.ID }) {
			continue
		}
		if apiErr := publishReceivingOrderLine(ctx, repos, constants.AuditActionCreate, purchaseOrderID, nil, line); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// publishReceivingOrderLine records a receiving line written or removed on its purchase order's behalf.
func publishReceivingOrderLine(ctx context.Context, repos domain.RepoFactory, action constants.AuditAction, purchaseOrderID string, before, after *domain.ReceivingOrderLine) *apierror.APIError {
	resourceID := ""
	if after != nil {
		resourceID = after.ID
	} else if before != nil {
		resourceID = before.ID
	}
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           action,
		ResourceType:     constants.ObjectTypeReceivingOrderLine,
		ResourceID:       resourceID,
		RootResourceType: constants.ObjectTypePurchaseOrder,
		RootResourceID:   purchaseOrderID,
		Changes:          audit.ComputeChanges(before, after),
	})
}

// sortedCopy returns values sorted, so a contact list's diff ignores the order rows came back in.
func sortedCopy(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	if out == nil {
		out = []string{}
	}
	return out
}

// appendListChange appends a change for field when the two sorted lists differ.
func appendListChange(changes []audit.FieldChange, field string, before, after []string) []audit.FieldChange {
	b, a := sortedCopy(before), sortedCopy(after)
	if slices.Equal(b, a) {
		return changes
	}
	return append(changes, audit.NewFieldChange(field, b, a))
}

// publishReceivingOrder records a receiving order raised or removed with its purchase order's status.
func publishReceivingOrder(ctx context.Context, repos domain.RepoFactory, action constants.AuditAction, before, after *domain.ReceivingOrder) *apierror.APIError {
	order := after
	if order == nil {
		order = before
	}
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           action,
		ResourceType:     constants.ObjectTypeReceivingOrder,
		ResourceID:       order.ID,
		RootResourceType: constants.ObjectTypePurchaseOrder,
		RootResourceID:   order.PurchaseOrderID,
		Changes:          audit.ComputeChanges(before, after),
	})
}

// setReceivingOrderCompletion runs mark, which completes or reopens the purchase order's receiving order, and records the change; an order without one is a no-op.
func setReceivingOrderCompletion(ctx context.Context, repos domain.RepoFactory, accountID, purchaseOrderID string, mark func() *apierror.APIError) *apierror.APIError {
	repo := repos.NewReceivingOrderRepo()
	receivingOrderID, apiErr := repo.GetByOrderID(ctx, purchaseOrderID)
	if apiErr != nil {
		return apiErr
	}
	if receivingOrderID == nil {
		return mark()
	}
	before, apiErr := repo.Get(ctx, accountID, *receivingOrderID)
	if apiErr != nil {
		return apiErr
	}
	if apiErr := mark(); apiErr != nil {
		return apiErr
	}
	after, apiErr := repo.Get(ctx, accountID, *receivingOrderID)
	if apiErr != nil {
		return apiErr
	}
	return publishReceivingOrder(ctx, repos, constants.AuditActionUpdate, before, after)
}

// salesOrderContacts reads the order's email recipients; an order with none reads as empty lists.
func salesOrderContacts(ctx context.Context, repo domain.SalesOrderRepo, salesOrderID string) (*domain.SalesOrderContacts, *apierror.APIError) {
	contacts, apiErr := repo.GetContactsByOrders(ctx, []string{salesOrderID})
	if apiErr != nil {
		return nil, apiErr
	}
	if c, ok := contacts[salesOrderID]; ok {
		return c, nil
	}
	return &domain.SalesOrderContacts{}, nil
}

// purchaseOrderContactUserIDs reads the account users the order's submission is sent to.
func purchaseOrderContactUserIDs(ctx context.Context, repo domain.PurchaseOrderRepo, purchaseOrderID string) ([]string, *apierror.APIError) {
	contacts, apiErr := repo.GetEmailContacts(ctx, purchaseOrderID)
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, 0, len(contacts))
	for _, c := range contacts {
		ids = append(ids, c.AccountUserID)
	}
	return ids, nil
}
