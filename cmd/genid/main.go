package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/augno/api/shared/id"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err) // #nosec G705 -- CLI stderr output, not web context
		os.Exit(1)
	}
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	if len(args) != 2 {
		fmt.Fprintf(stderr, "Usage: genid <prefix>\nExample: genid acbl\n")
		return errors.New("expected exactly one <prefix> argument")
	}

	prefix := id.IDPrefix(args[1])
	generated, apiErr := id.GenID(prefix, nil)
	if apiErr != nil {
		return fmt.Errorf("generating ID: %w", apiErr)
	}

	fmt.Fprint(stdout, generated)
	return nil
}
