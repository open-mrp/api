package id

import (
	"fmt"

	"github.com/augno/api/shared/contracts"
)

// GenID generates a new ID with the given prefix and length.
// If length is nil, it will default to 12.
func GenID(prefix IDPrefix, length *IDLength) (string, *contracts.APIError) {
	var useLength IDLength
	if length == nil {
		useLength = IDLength12
	} else {
		useLength = *length
	}

	baseID, err := genNanoID(useLength)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", prefix, baseID), nil
}
