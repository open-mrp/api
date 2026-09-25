package s3

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type countingProvider struct{ retrievals atomic.Int32 }

func (p *countingProvider) Retrieve(context.Context) (aws.Credentials, error) {
	p.retrievals.Add(1)
	return aws.Credentials{}, nil
}

func TestKeepCredentialsFresh_RetrievesOnEachTickUntilCancelled(t *testing.T) {
	t.Parallel()

	provider := &countingProvider{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		keepCredentialsFresh(ctx, provider, 5*time.Millisecond)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for provider.retrievals.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := provider.retrievals.Load(); got < 3 {
		t.Fatalf("retrieved %d times, want the loop to keep refreshing", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("keepCredentialsFresh kept running after its context was cancelled")
	}
}
