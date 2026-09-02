package mediator

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// EnqueueAllocateOpenIssues creates outbox messages telling the system to allocate available stock to open demand for each item. Because the outbox entry is written in the same transaction, the allocation request only exists if the inventory change successfully commits.
func EnqueueAllocateOpenIssues(ctx context.Context, repos domain.RepoFactory, accountID string, itemIDs ...string) *apierror.APIError {
	unique := ledgerlock.SortedUnique(itemIDs)
	if len(unique) == 0 {
		return nil
	}

	outboxRepo := repos.NewOutboxRepo()
	for _, itemID := range unique {
		if apiErr := EnqueueAllocateOpenIssuesFrom(ctx, outboxRepo, accountID, itemID, time.Time{}, "", ""); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// EnqueueAllocateOpenIssuesFrom creates one outbox message telling the system to allocate stock for a specific item, optionally starting from a paging cursor. It also preserves request metadata and can reuse a `messageID` for continuation retries so the consumer can process them idempotently.
func EnqueueAllocateOpenIssuesFrom(ctx context.Context, outboxRepo messaging.OutboxRepo, accountID, itemID string, afterCreatedAt time.Time, afterID, messageID string) *apierror.APIError {
	payload, err := json.Marshal(domain.AllocateOpenIssuesEvent{
		AccountID:      accountID,
		ItemID:         itemID,
		AfterCreatedAt: afterCreatedAt,
		AfterID:        afterID,
	})
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal allocate open issues event.")
	}

	msg := contracts.AmqpMessage{Data: payload}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	if _, err := outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.CoreCmdAllocateOpenIssues),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreCmdAllocateOpenIssues),
		Payload:     msg,
		MessageID:   messageID,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for allocate open issues.")
	}
	return nil
}
