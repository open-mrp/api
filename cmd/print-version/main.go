package main

import (
	"fmt"

	"github.com/augno/api/shared/version"
)

func main() {
	fmt.Print(version.Latest.Version)
}
