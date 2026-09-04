package event

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A consumer that skips the inbox wrapper has no deduplication at all: a redelivery runs the handler
// again with nothing to stop it. This reads the consumers themselves so a new one cannot quietly opt
// out of the guarantee the rest of them rely on.
// consumersExemptFromInbox lists consumers whose writes are idempotent on their own, with the reason.
var consumersExemptFromInbox = map[string]string{
	"agent_reply_patch_consumer.go": "each patch carries the full accumulated body, so the latest write wins and a repeat changes nothing",
}

func TestEveryConsumerRegistersThroughTheInbox(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob("*_consumer.go")
	if err != nil {
		t.Fatalf("globbing consumers: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no consumers found; this test is not looking where it thinks it is")
	}

	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		src := string(body)

		if !strings.Contains(src, "ConsumeMessages(") {
			continue
		}
		if _, exempt := consumersExemptFromInbox[filepath.Base(path)]; exempt {
			continue
		}
		if !strings.Contains(src, "inboxConsumer.Wrap(") {
			t.Errorf("%s consumes a queue without the inbox wrapper, so redeliveries are undeduplicated. "+
				"Wrap the handler, or add it to consumersExemptFromInbox with the reason its writes are "+
				"already idempotent.", path)
		}
	}
}

// A consumer that opens a transaction must commit its inbox recovery point inside it. Without one its
// work and the marker saying the work happened can be split by a crash, and the message is applied twice.
func TestEveryTransactionalConsumerCommitsARecoveryPoint(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("globbing: %v", err)
	}

	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		src := string(body)

		if !strings.Contains(src, "txManager.WithTx") {
			continue
		}
		if !strings.Contains(src, "completeInboxRecord") {
			t.Errorf("%s opens a transaction but commits no inbox recovery point", path)
		}
	}
}
