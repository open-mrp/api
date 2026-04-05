package main

import (
	"fmt"
	"os"

	"github.com/augno/api/shared/id"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: genid <prefix>\nExample: genid acbl\n")
		os.Exit(1)
	}

	prefix := id.IDPrefix(os.Args[1])
	generated, apiErr := id.GenID(prefix, nil)
	if apiErr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", apiErr) // #nosec G705 -- CLI stderr output, not web context
		os.Exit(1)
	}

	fmt.Print(generated)
}
