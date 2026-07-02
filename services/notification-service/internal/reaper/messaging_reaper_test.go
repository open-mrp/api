package reaper

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	s3 "github.com/augno/api/shared/cloud/s3"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/assert"
)

// fakeReaperRepo is a hand-written MessagingReaperRepo recording attachment purge calls. The
// notification/announcement methods are inert no-ops (covered elsewhere).
type fakeReaperRepo struct {
	purgeable    []AttachmentPurgeRef
	deletedRows  []string
	deleteRowErr error
}

func (f *fakeReaperRepo) PurgeActionedNotifications(context.Context, int, int32) (int64, error) {
	return 0, nil
}
func (f *fakeReaperRepo) PurgeStaleNotifications(context.Context, int, int32) (int64, error) {
	return 0, nil
}
func (f *fakeReaperRepo) PurgeExpiredAnnouncements(context.Context, int, int32) (int64, error) {
	return 0, nil
}
func (f *fakeReaperRepo) PurgeOrphanedAnnouncementReceipts(context.Context, int32) (int64, error) {
	return 0, nil
}
func (f *fakeReaperRepo) PurgeTombstonedMessages(context.Context, int, int32) (int64, error) {
	return 0, nil
}
func (f *fakeReaperRepo) ListPurgeableMessageAttachments(context.Context, int, int32) ([]AttachmentPurgeRef, error) {
	return f.purgeable, nil
}
func (f *fakeReaperRepo) DeleteMessageAttachmentByID(_ context.Context, id string) error {
	if f.deleteRowErr != nil {
		return f.deleteRowErr
	}
	f.deletedRows = append(f.deletedRows, id)
	return nil
}

// stubDeleteStore implements s3.ObjectStore, recording Delete calls and optionally failing them.
type stubDeleteStore struct {
	deleted   []string
	deleteErr *apierror.APIError
}

func (s *stubDeleteStore) Upload(context.Context, string, string, io.Reader, string) *apierror.APIError {
	return nil
}
func (s *stubDeleteStore) GetPresignedURL(context.Context, string, string, time.Duration) (string, *apierror.APIError) {
	return "", nil
}
func (s *stubDeleteStore) GetPresignedPutURL(context.Context, string, string, string, time.Duration) (string, *apierror.APIError) {
	return "", nil
}
func (s *stubDeleteStore) FileExists(context.Context, string, string) (bool, *apierror.APIError) {
	return true, nil
}
func (s *stubDeleteStore) Get(context.Context, string, string) ([]byte, *apierror.APIError) {
	return nil, nil
}
func (s *stubDeleteStore) Delete(_ context.Context, _ string, key string) *apierror.APIError {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, key)
	return nil
}
func (s *stubDeleteStore) Copy(context.Context, string, string, string) *apierror.APIError {
	return nil
}

func newReaper(t *testing.T, repo MessagingReaperRepo, store s3.ObjectStore) *MessagingReaper {
	t.Helper()
	return &MessagingReaper{
		config:      *(&MessagingReaperConfig{ServiceName: "test"}).WithDefaults(),
		repo:        repo,
		objectStore: store,
		chatBucket:  "chat-bucket",
	}
}

func TestReapTombstonedAttachments_DeletesObjectThenRow(t *testing.T) {
	repo := &fakeReaperRepo{purgeable: []AttachmentPurgeRef{
		{ID: "mgah_1", S3Key: "chat/ac/cv/mgah_1/a.png"},
		{ID: "mgah_2", S3Key: "chat/ac/cv/mgah_2/b.png"},
	}}
	store := &stubDeleteStore{}
	r := newReaper(t, repo, store)

	r.reapTombstonedAttachments(context.Background())

	assert.Equal(t, []string{"chat/ac/cv/mgah_1/a.png", "chat/ac/cv/mgah_2/b.png"}, store.deleted, "each object deleted")
	assert.Equal(t, []string{"mgah_1", "mgah_2"}, repo.deletedRows, "each row deleted after its object")
}

func TestReapTombstonedAttachments_ObjectDeleteFailureLeavesRow(t *testing.T) {
	repo := &fakeReaperRepo{purgeable: []AttachmentPurgeRef{{ID: "mgah_1", S3Key: "chat/ac/cv/mgah_1/a.png"}}}
	store := &stubDeleteStore{deleteErr: apierror.NewInternalError(errors.New("s3 down"), "boom")}
	r := newReaper(t, repo, store)

	r.reapTombstonedAttachments(context.Background())

	assert.Empty(t, repo.deletedRows, "the row is retained so the object delete is retried next tick")
}

func TestReapTombstonedAttachments_NoS3KeySkipsObjectDelete(t *testing.T) {
	repo := &fakeReaperRepo{purgeable: []AttachmentPurgeRef{{ID: "mgah_link", S3Key: ""}}}
	store := &stubDeleteStore{}
	r := newReaper(t, repo, store)

	r.reapTombstonedAttachments(context.Background())

	assert.Empty(t, store.deleted, "link/resource attachments have no object to delete")
	assert.Equal(t, []string{"mgah_link"}, repo.deletedRows, "the row is still removed")
}

func TestReapTombstonedAttachments_NilObjectStoreDeletesRowsOnly(t *testing.T) {
	repo := &fakeReaperRepo{purgeable: []AttachmentPurgeRef{{ID: "mgah_1", S3Key: "chat/ac/cv/mgah_1/a.png"}}}
	r := newReaper(t, repo, nil)

	r.reapTombstonedAttachments(context.Background())

	assert.Equal(t, []string{"mgah_1"}, repo.deletedRows)
}
