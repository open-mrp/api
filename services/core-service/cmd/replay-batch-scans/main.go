// Command replay-batch-scans re-drives core.batch_scanned_inventory inbox messages that failed
// permanently, one at a time, in the order they were originally scanned.
//
// A run of these messages had its inventory transaction aborted by Vitess when lock-wait pushed the
// transaction past its time limit, so nothing was applied: the inbox rows sit at status 'received'
// with processed_at NULL and attempts exhausted, while the payloads survive intact in message_outbox.
// The lock contention that caused the aborts is removed separately; this catches up the messages it
// stranded.
//
// It replays through the consumer's own inbox-dedup wrapper, which skips any message already marked
// processed, so this is idempotent and safe to re-run: a message applied by an earlier run is skipped
// on the next. Replaying is sequential and ordered by received_at because scans allocate FIFO, and
// applying them out of order would hand receipts to the wrong demand.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/replay-batch-scans --account <id> --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/replay-batch-scans --account <id>
//
// Flags:
//
//	--account        only replay this account's messages (required).
//	--dry-run        list what would replay, in order; make no writes.
//	--halt-on-error  stop at the first message that errors (default true); when false, log and continue.
//	--limit          replay at most this many messages after ordering and filtering (0 = all).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/service"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/env"

	amqp "github.com/rabbitmq/amqp091-go"
)

const replayHandler = "core.batch_scanned_inventory"

const replayRoutingKey = "core.event.batch_scanned"

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err) // #nosec G705 -- CLI stderr output, not web context
		os.Exit(1)
	}
}

type failedMessage struct {
	messageID  string
	receivedAt time.Time
	payload    []byte
	accountID  string
	batchID    string
	itemID     string
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	fs := flag.NewFlagSet("replay-batch-scans", flag.ContinueOnError)
	fs.SetOutput(stderr)
	account := fs.String("account", "", "only replay this account's messages (required)")
	dryRun := fs.Bool("dry-run", false, "list what would replay, in order; make no writes")
	haltOnError := fs.Bool("halt-on-error", true, "stop at the first message that errors")
	limit := fs.Int("limit", 0, "replay at most this many messages after ordering and filtering (0 = all)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	if *account == "" {
		return fmt.Errorf("--account is required")
	}

	dbURL := env.GetEnv("DB_URL", getenv)
	if dbURL == "" {
		return fmt.Errorf("DB_URL is required")
	}
	dsn, err := normalizeDSN(dbURL)
	if err != nil {
		return err
	}

	pool, err := db.NewDbPool(&db.Config{DBURI: dsn})
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	messages, err := loadFailedMessages(ctx, pool, *account, *limit)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Fprintf(stdout, "no permanently-failed batch_scanned messages for account %s; nothing to do\n", *account)
		return nil
	}

	if *dryRun {
		fmt.Fprintf(stdout, "would replay %d message(s) for account %s, in order:\n", len(messages), *account)
		fmt.Fprintf(stdout, "  %-4s  %-24s  %-30s  %-24s  %s\n", "#", "received_at", "message_id", "batch_id", "item_id")
		for i, m := range messages {
			fmt.Fprintf(stdout, "  %-4d  %-24s  %-30s  %-24s  %s\n",
				i+1, m.receivedAt.UTC().Format(time.RFC3339), m.messageID, m.batchID, m.itemID)
		}
		return nil
	}

	queries := sqlc.New(pool)
	repoFactory := repository.NewRepoFactory(queries)
	inboxRepo := repository.NewInboxRepo(queries)
	txManager := service.NewTransactionManager(pool, queries)
	consumer := event.NewBatchScannedConsumer(nil, inboxRepo, repoFactory, txManager)

	var applied, skipped, failed int
	var haltedAt string
	total := len(messages)
	for i, m := range messages {
		wasProcessed, err := messageProcessed(ctx, pool, m.messageID)
		if err != nil {
			return err
		}

		delivery := amqp.Delivery{
			Body:       m.payload,
			MessageId:  m.messageID,
			RoutingKey: replayRoutingKey,
		}
		replayErr := consumer.ReplayMessage(ctx, delivery)
		switch {
		case replayErr != nil:
			failed++
			fmt.Fprintf(stdout, "  [%d/%d] %s  error: %s\n", i+1, total, m.messageID, replayErr)
			if *haltOnError {
				haltedAt = m.messageID
				fmt.Fprintf(stdout, "\nhalted at %s; %d applied, %d skipped, %d failed of %d\n",
					haltedAt, applied, skipped, failed, total)
				return fmt.Errorf("halted on error at message %s: %w", haltedAt, replayErr)
			}
		case wasProcessed:
			skipped++
			fmt.Fprintf(stdout, "  [%d/%d] %s  skipped-already-processed\n", i+1, total, m.messageID)
		default:
			applied++
			fmt.Fprintf(stdout, "  [%d/%d] %s  applied\n", i+1, total, m.messageID)
		}
	}

	fmt.Fprintf(stdout, "\ndone: %d applied, %d skipped, %d failed of %d\n", applied, skipped, failed, total)
	if failed > 0 {
		return fmt.Errorf("%d message(s) failed; re-run to retry them", failed)
	}
	return nil
}

// loadFailedMessages returns the permanently-failed batch_scanned messages for one account, oldest
// first, capped at limit. The inbox is joined to the outbox for the surviving payload, and the account
// and display fields are read from the decoded event because the inbox row does not carry them.
func loadFailedMessages(ctx context.Context, pool *sql.DB, account string, limit int) ([]failedMessage, error) {
	query := `
		SELECT i.message_id, i.received_at, o.payload
		FROM message_inbox i
		JOIN message_outbox o ON o.message_id = i.message_id
		WHERE i.handler = ?
		  AND i.status = 'received'
		  AND i.processed_at IS NULL
		ORDER BY i.received_at ASC`

	rows, err := pool.QueryContext(ctx, query, replayHandler)
	if err != nil {
		return nil, fmt.Errorf("listing failed messages: %w", err)
	}
	defer rows.Close()

	var messages []failedMessage
	for rows.Next() {
		var m failedMessage
		if err := rows.Scan(&m.messageID, &m.receivedAt, &m.payload); err != nil {
			return nil, fmt.Errorf("scanning failed message: %w", err)
		}

		evt, err := decodeEvent(m.payload)
		if err != nil {
			return nil, fmt.Errorf("decoding payload for message %s: %w", m.messageID, err)
		}
		m.accountID = evt.AccountID
		m.batchID = evt.BatchID
		m.itemID = evt.ItemID

		if m.accountID != account {
			continue
		}
		messages = append(messages, m)
		if limit > 0 && len(messages) == limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing failed messages: %w", err)
	}
	return messages, nil
}

// decodeEvent unwraps the AMQP envelope stored in the outbox payload and returns the batch-scanned
// event it carries. The envelope's data field is base64-encoded JSON, which json.Unmarshal into the
// []byte field decodes for us.
func decodeEvent(payload []byte) (domain.BatchScannedEvent, error) {
	var envelope contracts.AmqpMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return domain.BatchScannedEvent{}, fmt.Errorf("unmarshaling envelope: %w", err)
	}
	var evt domain.BatchScannedEvent
	if err := json.Unmarshal(envelope.Data, &evt); err != nil {
		return domain.BatchScannedEvent{}, fmt.Errorf("unmarshaling event: %w", err)
	}
	if evt.AccountID == "" && envelope.Identity != nil && envelope.Identity.Target != nil {
		evt.AccountID = envelope.Identity.Target.AccountID
	}
	return evt, nil
}

// messageProcessed reports whether the inbox already marks this message processed, which is how a
// replay tells an apply from a skip: the SELECT that built the run only saw unprocessed rows, so any
// row now processed was handled by an earlier run.
func messageProcessed(ctx context.Context, pool *sql.DB, messageID string) (bool, error) {
	var status string
	err := pool.QueryRowContext(ctx,
		"SELECT status FROM message_inbox WHERE message_id = ? AND handler = ?",
		messageID, replayHandler,
	).Scan(&status)
	if err != nil {
		return false, fmt.Errorf("reading inbox status for message %s: %w", messageID, err)
	}
	return status == "processed", nil
}

func normalizeDSN(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("DB_URL is empty")
	}
	if strings.Contains(raw, "@tcp(") || strings.Contains(raw, "@unix(") {
		return raw, nil
	}
	if !strings.HasPrefix(raw, "mysql://") {
		return "", fmt.Errorf("unrecognized DB_URL form; expected a mysql://... URL or a user:pass@tcp(host:port)/db DSN")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parsing DB_URL: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("DB_URL has no host")
	}
	port := u.Port()
	if port == "" {
		port = "3306"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	userInfo := u.User.Username()
	if pass, ok := u.User.Password(); ok && pass != "" {
		userInfo += ":" + pass
	}

	dsn := fmt.Sprintf("%s@tcp(%s:%s)/%s", userInfo, host, port, dbName)
	if host != "localhost" && host != "127.0.0.1" {
		dsn += "?tls=true"
	}
	return dsn, nil
}
