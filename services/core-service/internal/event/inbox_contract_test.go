package event

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The guarantees in this package are structural: a consumer that opens a transaction and does not
// commit its recovery point inside it is at-most-once, not exactly-once, and nothing at runtime says
// so — the message simply gets applied twice the next time a process dies at the wrong moment. These
// tests read the consumers themselves so a new one cannot quietly opt out.

// consumersExemptFromRecoveryPoint lists the consumers that open a transaction but deliberately
// commit no recovery point, with the reason. Adding a name here is a decision about duplicate work,
// so it should be argued in the consumer's own comment as well.
var consumersExemptFromRecoveryPoint = map[string]string{
	"allocate_open_issues_consumer.go": "pages across one transaction per issue, so there is no single transaction to commit into; allocation is convergent, so a repeat allocates nothing",
}

func consumerFiles(t *testing.T) []string {
	t.Helper()

	entries, err := filepath.Glob("*_consumer.go")
	if err != nil {
		t.Fatalf("globbing consumers: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no consumers found; this test is not looking where it thinks it is")
	}
	return entries
}

func TestEveryTransactionalConsumerCommitsARecoveryPoint(t *testing.T) {
	t.Parallel()

	for _, path := range consumerFiles(t) {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		src := string(body)

		if !strings.Contains(src, "txManager.WithTx") {
			continue
		}
		if reason, exempt := consumersExemptFromRecoveryPoint[filepath.Base(path)]; exempt {
			if !strings.Contains(src, "completeInboxRecord") && reason == "" {
				t.Errorf("%s is exempt with no reason recorded", path)
			}
			continue
		}
		if !strings.Contains(src, "completeInboxRecord") {
			t.Errorf("%s opens a transaction but commits no inbox recovery point: its work and the marker "+
				"saying the work happened can be split by a crash, and the message will be applied twice. "+
				"Call completeInboxRecord as the last statement inside the transaction, or add it to "+
				"consumersExemptFromRecoveryPoint with the reason.", path)
		}
	}
}

// A message a consumer refuses to process has to leave a terminal record. Returning nil acks the
// delivery and lets the wrapper mark it processed, so work that was dropped becomes indistinguishable
// from work that was applied — in the inbox, in the failure monitor, and to the replay commands.
func TestConsumersDoNotSilentlyDropMalformedMessages(t *testing.T) {
	t.Parallel()

	// Phrases that mark a branch as giving up on a message rather than reporting a failure.
	dropPhrases := []string{"dropping", "; skipping", "not syncable"}

	for _, path := range consumerFiles(t) {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		lines := strings.Split(string(body), "\n")

		for i, line := range lines {
			lower := strings.ToLower(line)
			matched := false
			for _, phrase := range dropPhrases {
				if strings.Contains(lower, phrase) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			// Look at what the branch actually does, stopping at the end of the branch so the
			// enclosing function's own `return nil` is not mistaken for the drop's.
			for j := i; j < len(lines) && j < i+4; j++ {
				if trimmed := strings.TrimSpace(lines[j]); trimmed == "}" || trimmed == "case" {
					break
				}
				if strings.TrimSpace(lines[j]) == "return nil" {
					t.Errorf("%s:%d drops a message with a bare `return nil`, which records it as processed. "+
						"Use Discard (or discardIfPermanent) so the drop is terminal and visible.", path, j+1)
					break
				}
			}
		}
	}
}

// Every consumer must go through the inbox wrapper, or it has no deduplication at all: a redelivery
// runs the handler again with nothing to stop it.
func TestEveryConsumerRegistersThroughTheInbox(t *testing.T) {
	t.Parallel()

	for _, path := range consumerFiles(t) {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		src := string(body)

		if !strings.Contains(src, "ConsumeMessages(") {
			continue
		}
		if !strings.Contains(src, "inboxConsumer.Wrap(") {
			t.Errorf("%s consumes a queue without the inbox wrapper, so redeliveries are undeduplicated", path)
		}
	}
}
