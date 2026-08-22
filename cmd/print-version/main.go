package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/open-mrp/api/shared/version"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func Run(
	ctx context.Context,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	_, err := fmt.Fprint(stdout, version.Latest.Version)
	return err
}
