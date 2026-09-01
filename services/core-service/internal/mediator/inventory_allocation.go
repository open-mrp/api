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

// EnqueueAllocateOpenIssues asks for items' open demand to be covered, from inside the caller's
// transaction. It is the only way to request allocation.
//
// Allocation used to run inline at three call sites — stocking a receiving order, voiding a shipment,
// undoing a batch scan — walking every open issue for the item before the request could answer. That
// walk is unbounded: an item with a long tail of demand no receipt can cover pays for all of it on
// every stocking. Worse, it ran with the receipts already locked, which is the opposite order from
// the consumer doing the same work asynchronously, and that inversion is the deadlock that has
// survived every fix so far. Nothing in the caller's response depended on the result.
//
// The outbox row commits with the caller's transaction, so allocation is requested if and only if the
// stock movement that justifies it actually happened. Callers should kick the enqueuer AFTER that
// commit — see messaging.OutboxNotifier — or the request waits out the enqueuer's idle poll backoff.
//
// Ids are sorted and deduplicated so two callers touching the same items produce the same sequence of
// rows, which keeps the outbox readable and matches the order the ledger work itself will take them
// in.
func EnqueueAllocateOpenIssues(ctx context.Context, repos domain.RepoFactory, accountID string, itemIDs ...string) *apierror.APIError {
	unique := SortedUniqueIDs(itemIDs)
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

// EnqueueAllocateOpenIssuesFrom enqueues one allocation request, optionally resuming from a cursor.
//
// messageID is empty for everything that starts a chain, so the outbox mints a random one and each
// request is its own chain. Only the consumer's own paging continuation passes one, and only so that
// its retries are idempotent: a derived id keyed on the cursor alone would be identical for every
// request an item ever gets, and the inbox would dedupe all of them against the first.
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

// SortedUniqueIDs drops blanks, deduplicates and sorts.
//
// It is ledgerlock.SortedUnique under another name, deliberately: the ids an enqueue produces are the
// ids the ledger work will take locks on, and two orderings that can drift apart are two orderings
// that eventually will.
func SortedUniqueIDs(ids []string) []string {
	return ledgerlock.SortedUnique(ids)
}
