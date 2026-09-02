// Command replay-allocate-open-issues re-drives core.allocate_open_issues inbox messages that failed permanently, oldest first.
//
// These messages carry a page of one item's open demand. A run of them died in vtgate on `FOR UPDATE OF`, a parse error, so every per-issue transaction rolled back whole and the pages left their issues open: the inbox rows sit at status 'received' with processed_at NULL and attempts exhausted, while the payloads survive in message_outbox. There is nothing to unwind, only demand that was never offered the stock it can draw on.
//
// Replay goes through the consumer's own inbox-dedup wrapper, which skips any message already marked processed, so this is idempotent and safe to re-run. A replayed full page enqueues its continuation to the outbox exactly as a live delivery does, so a chain longer than one page finishes through the running service rather than here.
//
// Deploy the fix first. Replaying against a service that still has the broken query fails every message again and burns the run.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/replay-allocate-open-issues --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/replay-allocate-open-issues --account ac_...
//
// Flags:
//
//	--account        only replay this account's messages (default: all).
//	--item           only replay this item's messages (default: all).
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
	"sort"
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

const replayHandler = "core.allocate_open_issues"

const replayRoutingKey = "core.cmd.allocate_open_issues"

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err) // #nosec G705 -- CLI stderr output, not web context
		os.Exit(1)
	}
}

// failedMessage is one dead inbox row with the request it carried.
type failedMessage struct {
	messageID  string
	receivedAt time.Time
	accountID  string
	itemID     string
	afterID    string
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	fs := flag.NewFlagSet("replay-allocate-open-issues", flag.ContinueOnError)
	fs.SetOutput(stderr)
	account := fs.String("account", "", "only replay this account's messages (default: all)")
	item := fs.String("item", "", "only replay this item's messages (default: all)")
	dryRun := fs.Bool("dry-run", false, "list what would replay, in order; make no writes")
	haltOnError := fs.Bool("halt-on-error", true, "stop at the first message that errors")
	limit := fs.Int("limit", 0, "replay at most this many messages after ordering and filtering (0 = all)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
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

	messages, payloads, unrecoverable, err := loadFailedMessages(ctx, pool, *account, *item, *limit)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Fprintf(stdout, "no permanently-failed allocate_open_issues messages%s; nothing to do\n", scopeSuffix(*account, *item))
		reportUnrecoverable(stdout, unrecoverable)
		return nil
	}

	if *dryRun {
		return report(stdout, messages, unrecoverable, *account, *item)
	}

	queries := sqlc.New(pool)
	repoFactory := repository.NewRepoFactory(queries)
	inboxRepo := repository.NewInboxRepo(queries)
	txManager := service.NewTransactionManager(pool, queries)
	consumer := event.NewAllocateOpenIssuesConsumer(nil, inboxRepo, repoFactory, txManager)

	var applied, skipped, failed int
	total := len(messages)
	for i, m := range messages {
		wasProcessed, err := messageProcessed(ctx, pool, m.messageID)
		if err != nil {
			return err
		}

		delivery := amqp.Delivery{
			Body:       payloads[m.messageID],
			MessageId:  m.messageID,
			RoutingKey: replayRoutingKey,
		}
		replayErr := consumer.ReplayMessage(ctx, delivery)
		switch {
		case replayErr != nil:
			failed++
			fmt.Fprintf(stdout, "  [%d/%d] %s  error: %s\n", i+1, total, m.messageID, replayErr)
			if *haltOnError {
				fmt.Fprintf(stdout, "\nhalted at %s; %d applied, %d skipped, %d failed of %d\n",
					m.messageID, applied, skipped, failed, total)
				return fmt.Errorf("halted on error at message %s: %w", m.messageID, replayErr)
			}
		case wasProcessed:
			skipped++
			fmt.Fprintf(stdout, "  [%d/%d] %s  skipped-already-processed\n", i+1, total, m.messageID)
		default:
			applied++
			fmt.Fprintf(stdout, "  [%d/%d] %s  applied  %s/%s\n", i+1, total, m.messageID, m.accountID, m.itemID)
		}
	}

	fmt.Fprintf(stdout, "\ndone: %d applied, %d skipped, %d failed of %d\n", applied, skipped, failed, total)
	reportUnrecoverable(stdout, unrecoverable)
	if failed > 0 {
		return fmt.Errorf("%d message(s) failed; re-run to retry them", failed)
	}
	return nil
}

// report is the dry run: the run in order, then the distinct items behind it, which is the number that says how much demand this is about to re-offer.
func report(stdout io.Writer, messages []failedMessage, unrecoverable []string, account, item string) error {
	fmt.Fprintf(stdout, "would replay %d message(s)%s, in order:\n", len(messages), scopeSuffix(account, item))
	fmt.Fprintf(stdout, "  %-4s  %-24s  %-38s  %-32s  %-32s  %s\n", "#", "received_at", "message_id", "account", "item", "cursor")
	items := map[string]int{}
	for i, m := range messages {
		cursor := m.afterID
		if cursor == "" {
			cursor = "(first page)"
		}
		fmt.Fprintf(stdout, "  %-4d  %-24s  %-38s  %-32s  %-32s  %s\n",
			i+1, m.receivedAt.UTC().Format(time.RFC3339), m.messageID, m.accountID, m.itemID, cursor)
		items[m.accountID+"/"+m.itemID]++
	}

	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Fprintf(stdout, "\n%d distinct (account, item) pair(s):\n", len(keys))
	for _, k := range keys {
		fmt.Fprintf(stdout, "  %-64s  %d page(s)\n", k, items[k])
	}
	reportUnrecoverable(stdout, unrecoverable)
	return nil
}

// reportUnrecoverable names the dead rows whose outbox payload has been pruned. They carry no account or item, so they are outside every filter and cannot be replayed: those items need their allocation re-requested instead.
func reportUnrecoverable(stdout io.Writer, ids []string) {
	if len(ids) == 0 {
		return
	}
	fmt.Fprintf(stdout, "\n%d message(s) have no outbox payload and cannot be replayed:\n", len(ids))
	for _, id := range ids {
		fmt.Fprintf(stdout, "  %s\n", id)
	}
}

// loadFailedMessages returns the permanently-failed allocate_open_issues messages, oldest first, capped at limit. The account, item and cursor come from the decoded event because the inbox row does not carry them, so the filters are applied after decoding.
func loadFailedMessages(ctx context.Context, pool *sql.DB, account, item string, limit int) ([]failedMessage, map[string][]byte, []string, error) {
	// LEFT JOIN, not JOIN: outbox rows are pruned after about a week, and a dead inbox row whose payload is gone has to be reported rather than silently dropped from the run.
	query := `
		SELECT i.message_id, i.received_at, o.payload
		FROM message_inbox i
		LEFT JOIN message_outbox o ON o.message_id = i.message_id
		WHERE i.handler = ?
		  AND i.status = 'received'
		  AND i.processed_at IS NULL
		ORDER BY i.received_at ASC`

	rows, err := pool.QueryContext(ctx, query, replayHandler)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("listing failed messages: %w", err)
	}
	defer rows.Close()

	var messages []failedMessage
	var unrecoverable []string
	payloads := map[string][]byte{}
	for rows.Next() {
		var m failedMessage
		var payload []byte
		if err := rows.Scan(&m.messageID, &m.receivedAt, &payload); err != nil {
			return nil, nil, nil, fmt.Errorf("scanning failed message: %w", err)
		}

		if payload == nil {
			unrecoverable = append(unrecoverable, m.messageID)
			continue
		}

		evt, err := decodeEvent(payload)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("decoding payload for message %s: %w", m.messageID, err)
		}
		m.accountID, m.itemID, m.afterID = evt.AccountID, evt.ItemID, evt.AfterID

		if account != "" && m.accountID != account {
			continue
		}
		if item != "" && m.itemID != item {
			continue
		}
		payloads[m.messageID] = payload
		messages = append(messages, m)
		if limit > 0 && len(messages) == limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("listing failed messages: %w", err)
	}
	return messages, payloads, unrecoverable, nil
}

// decodeEvent unwraps the AMQP envelope stored in the outbox payload and returns the allocation request it carries. The envelope's data field is base64-encoded JSON, which json.Unmarshal into the []byte field decodes for us.
func decodeEvent(payload []byte) (domain.AllocateOpenIssuesEvent, error) {
	var envelope contracts.AmqpMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return domain.AllocateOpenIssuesEvent{}, fmt.Errorf("unmarshaling envelope: %w", err)
	}
	var evt domain.AllocateOpenIssuesEvent
	if err := json.Unmarshal(envelope.Data, &evt); err != nil {
		return domain.AllocateOpenIssuesEvent{}, fmt.Errorf("unmarshaling event: %w", err)
	}
	if evt.AccountID == "" && envelope.Identity != nil && envelope.Identity.Target != nil {
		evt.AccountID = envelope.Identity.Target.AccountID
	}
	return evt, nil
}

// messageProcessed reports whether the inbox already marks this message processed, which is how a replay tells an apply from a skip: the SELECT that built the run only saw unprocessed rows, so any row now processed was handled by an earlier run.
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

func scopeSuffix(account, item string) string {
	switch {
	case account != "" && item != "":
		return fmt.Sprintf(" for account %s item %s", account, item)
	case account != "":
		return fmt.Sprintf(" for account %s", account)
	case item != "":
		return fmt.Sprintf(" for item %s", item)
	}
	return ""
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
