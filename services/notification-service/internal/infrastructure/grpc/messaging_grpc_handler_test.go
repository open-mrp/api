package grpc

import (
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeCountFromMetadata(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want *uint32
	}{
		{"nil", nil, nil},
		{"empty", json.RawMessage(""), nil},
		{"absent key", json.RawMessage(`{}`), nil},
		{"invalid json", json.RawMessage(`{`), nil},
		{"first change", json.RawMessage(`{"change_count":1}`), ptr(uint32(1))},
		{"rolled up", json.RawMessage(`{"change_count":5}`), ptr(uint32(5))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := changeCountFromMetadata(tt.raw)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *tt.want, *got)
		})
	}
}

func TestToProtoNotification_ChangeCount(t *testing.T) {
	withCount := toProtoNotification(&domain.Notification{
		ID:       "nf_1",
		Metadata: json.RawMessage(`{"change_count":5}`),
	})
	require.NotNil(t, withCount.ChangeCount)
	assert.Equal(t, uint32(5), *withCount.ChangeCount)

	noCount := toProtoNotification(&domain.Notification{ID: "nf_2"})
	assert.Nil(t, noCount.ChangeCount)
}

func ptr[T any](v T) *T { return &v }
