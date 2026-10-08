package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/db/backfill"
	"golang.org/x/sync/errgroup"
)

// RequestLogPayloadBackfillName names the backfill that moves request-log bodies written before
// the gateway stored them in object storage.
const RequestLogPayloadBackfillName = "request_log_payloads"

// requestLogPayloadUploads bounds concurrent uploads within a batch. The uploads touch no database,
// so they only bound the pod's own memory and connections to S3.
const requestLogPayloadUploads = 8

// RequestLogPayloadBackfill moves request-log bodies still inline in request_log into the payloads
// bucket. Rows are immutable once written, so the bodies are read from the replica, keeping the bulk
// read off the primary's buffer pool; only the short primary-key update touches the primary.
type RequestLogPayloadBackfill struct {
	replica  *sqlc.Queries
	primary  *sqlc.Queries
	payloads *blobstore.Store
	// before bounds the walk. Logs written since the gateway began storing bodies in the bucket need
	// no move, and the bound lets the walk end instead of chasing new rows.
	before time.Time
}

func NewRequestLogPayloadBackfill(replica, primary *sqlc.Queries, payloads *blobstore.Store, before time.Time) *RequestLogPayloadBackfill {
	return &RequestLogPayloadBackfill{replica: replica, primary: primary, payloads: payloads, before: before}
}

// Batch moves the bodies of up to limit logs after cursor: upload each payload, then point the rows
// at their objects. An object always exists before a row names it, and every step is safe to repeat.
func (b *RequestLogPayloadBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	afterAt, afterID, err := parseRequestLogCursor(cursor)
	if err != nil {
		return "", 0, false, err
	}

	var page []sqlc.ListRequestLogIDsAfterRow
	if err := m.Time(func() error {
		var err error
		page, err = b.replica.ListRequestLogIDsAfter(ctx, sqlc.ListRequestLogIDsAfterParams{
			Before:          b.before,
			AfterOccurredAt: afterAt,
			AfterID:         afterID,
			Limit:           int32(limit), // #nosec G115 - bounded by the runner's MaxBatch
		})
		return err
	}); err != nil {
		return "", 0, false, fmt.Errorf("list request logs: %w", err)
	}
	if len(page) == 0 {
		return cursor, 0, true, nil
	}
	last := page[len(page)-1]
	next := formatRequestLogCursor(last.OccurredAt, last.ID)

	ids := make([]string, len(page))
	for i, row := range page {
		ids[i] = row.ID
	}

	var inline []sqlc.ListInlineRequestLogPayloadsRow
	if err := m.Time(func() error {
		var err error
		inline, err = b.replica.ListInlineRequestLogPayloads(ctx, ids)
		return err
	}); err != nil {
		return "", 0, false, fmt.Errorf("read request log payloads: %w", err)
	}

	var moved []string
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(requestLogPayloadUploads)
	for _, row := range inline {
		payload, ok := requestLogPayloadFromRow(row)
		if !ok {
			continue
		}
		moved = append(moved, row.ID)
		g.Go(func() error {
			if apiErr := b.payloads.Put(gctx, contracts.RequestLogPayloadKey(row.ID), payload); apiErr != nil {
				return apiErr
			}
			return nil
		})
	}
	// Any failed upload fails the batch before a row is touched, so the runner retries it whole.
	if err := g.Wait(); err != nil {
		return "", 0, false, fmt.Errorf("upload request log payloads: %w", err)
	}
	if len(moved) == 0 {
		return next, 0, false, nil
	}

	var updated int64
	if err := m.Time(func() error {
		var err error
		updated, err = b.primary.PointRequestLogsAtPayloads(ctx, moved)
		return err
	}); err != nil {
		return "", 0, false, fmt.Errorf("point request logs at payloads: %w", err)
	}
	return next, int(updated), false, nil
}

// requestLogPayloadFromRow builds the stored payload. A row whose bodies are only the {} the
// repository writes for an absent body has nothing worth moving and stays as it is: its columns
// are a few bytes, and an absent payload reads back the same way.
func requestLogPayloadFromRow(row sqlc.ListInlineRequestLogPayloadsRow) (contracts.RequestLogPayload, bool) {
	payload := contracts.RequestLogPayload{
		QueryJSON:        nonEmptyJSON(row.QueryJson),
		RequestBodyJSON:  nonEmptyJSON(row.RequestBodyJson),
		ResponseBodyJSON: nonEmptyJSON(row.ResponseBodyJson),
		StackTrace:       db.StringFromNullString(row.StackTrace),
	}
	return payload, payload != (contracts.RequestLogPayload{})
}

func nonEmptyJSON(v db.NullableRawMessage) *string {
	s := strings.TrimSpace(string(v))
	if s == "" || s == "{}" || s == "null" {
		return nil
	}
	return &s
}

// The cursor is the (occurred_at, id) of the last row walked, in the order the index stores them.
func formatRequestLogCursor(at time.Time, id string) string {
	return at.UTC().Format(time.RFC3339Nano) + "|" + id
}

func parseRequestLogCursor(cursor string) (time.Time, string, error) {
	// MySQL rejects Go's zero time; every log is younger than the epoch.
	if cursor == "" {
		return time.Unix(0, 0).UTC(), "", nil
	}
	at, id, ok := strings.Cut(cursor, "|")
	if !ok {
		return time.Time{}, "", fmt.Errorf("malformed request log backfill cursor %q", cursor)
	}
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("malformed request log backfill cursor %q: %w", cursor, err)
	}
	return t, id, nil
}
