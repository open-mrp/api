package grpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"

	"google.golang.org/grpc/metadata"
)

const (
	IdentityHeader = "identity"
)

// GetIdentityFromMetadata extracts the identity from the metadata
func GetIdentityFromMetadata(md metadata.MD) (*types.Identity, *contracts.APIError) {
	data := md.Get(IdentityHeader)
	if len(data) == 0 {
		return nil, contracts.NewInternalError(errors.New("Identity not found in metadata"), "Identity not found in metadata") // #nosec G101
	} else if len(data) > 1 {
		return nil, contracts.NewInternalError(fmt.Errorf("Identity metadata malformed: %s", strings.Join(data, ", ")), "Identity metadata malformed") // #nosec G101
	}
	var identity types.Identity
	err := json.Unmarshal([]byte(data[0]), &identity)
	if err != nil {
		return nil, contracts.NewInternalError(err, "Identity metadata malformed")
	}
	return &identity, nil
}

// SetIdentityInMetadata sets the identity in the metadata
func SetIdentityInMetadata(md metadata.MD, identity *types.Identity) {
	jsonData, err := json.Marshal(identity)
	if err != nil {
		return
	}
	md.Set(IdentityHeader, string(jsonData))
}
