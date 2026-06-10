package main

import (
	"context"
	"errors"
	"fmt"
	"os"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, errBadFlags) {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(1)
	}
}
