package tracing_test

import (
	"context"
	"os"

	"github.com/augno/api/shared/tracing"
)

// ExampleInitProvider shows the minimal call to install the global tracer
// provider at service startup. Configuration is resolved from OTEL_*
// environment variables with production defaults.
func ExampleInitProvider() {
	shutdown, err := tracing.InitProvider(context.Background(), "my-service", os.Getenv)
	if err != nil {
		panic(err)
	}
	defer tracing.DeferShutdown(shutdown)()
}
