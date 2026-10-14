package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/open-mrp/api/shared/proto/core"
)

func TestCreatedRangeFromProto(t *testing.T) {
	t.Parallel()
	assert.Nil(t, createdRangeFromProto(nil).After)

	jan := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	r := createdRangeFromProto(&pb.CreatedRange{CreatedAfter: timestamppb.New(jan)})
	if assert.NotNil(t, r.After) {
		assert.True(t, r.After.Equal(jan))
	}
	assert.Nil(t, r.Before)
}
