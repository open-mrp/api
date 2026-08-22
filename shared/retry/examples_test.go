package retry_test

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/shared/retry"
)

// ExampleWithBackoff shows the minimal configuration needed to retry an
// operation: pass a partially filled Config (or nil) and zero-value fields
// are replaced with production defaults.
func ExampleWithBackoff() {
	cfg := &retry.Config{
		MaxRetries:  2,
		InitialWait: 10 * time.Millisecond,
	}

	err := retry.WithBackoff(context.Background(), cfg, func() error {
		return nil // the operation to retry
	})
	fmt.Println(err)
	// Output: <nil>
}
