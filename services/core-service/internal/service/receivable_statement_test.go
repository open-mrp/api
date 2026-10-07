package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	publishermock "github.com/open-mrp/api/services/core-service/internal/domain/mock/publisher"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"go.uber.org/mock/gomock"
)

type statementFixture struct {
	svc            domain.ReceivableSvc
	customerRepo   *repositorymock.MockCustomerRepo
	receivableRepo *repositorymock.MockReceivableRepo
	accountRepo    *repositorymock.MockAccountRepo
	idempotency    *mediatormock.MockIdempotencyMed
	publisher      *publishermock.MockNotificationPublisher
}

func newStatementFixture(t *testing.T) statementFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	f := statementFixture{
		customerRepo:   repositorymock.NewMockCustomerRepo(ctrl),
		receivableRepo: repositorymock.NewMockReceivableRepo(ctrl),
		accountRepo:    repositorymock.NewMockAccountRepo(ctrl),
		idempotency:    mediatormock.NewMockIdempotencyMed(ctrl),
		publisher:      publishermock.NewMockNotificationPublisher(ctrl),
	}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewCustomerRepo().Return(f.customerRepo).AnyTimes()
	repos.EXPECT().NewReceivableRepo().Return(f.receivableRepo).AnyTimes()
	repos.EXPECT().NewAccountRepo().Return(f.accountRepo).AnyTimes()
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: f.idempotency}).AnyTimes()
	f.svc = NewReceivableSvc(&ReceivableSvcConfig{
		Repos:                 repos,
		MediatorFactory:       meds,
		TxManager:             &stubTxManager{factory: repos},
		NotificationPublisher: f.publisher,
	})
	f.idempotency.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_1", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	return f
}

func statementCtx() context.Context {
	return invoiceActorCtx(types.IdentityRelationTypeInternal, invoiceSellerID, invoiceSellerID, "customers:read")
}

// A statement is only for one of the account's customers: any other account is a 404 and nothing is
// read or sent, as the dashboard refused it.
func TestEmailReceivables_RefusesAnAccountThatIsNotACustomer(t *testing.T) {
	t.Parallel()

	f := newStatementFixture(t)
	notFound := apierror.NewResourceNotFoundError("Customer not found.")
	f.customerRepo.EXPECT().Get(gomock.Any(), invoiceSellerID, "ac_stranger", gomock.Nil()).Return(nil, notFound)
	f.idempotency.EXPECT().CacheErrorResponse(gomock.Any(), "idk_1", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr })

	apiErr := f.svc.EmailReceivablesForCustomer(statementCtx(), domain.EmailReceivablesParams{
		CustomerAccountID: "ac_stranger",
		RecipientEmails:   []string{"ap@example.com"},
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

// The statement a customer is sent pads invoice and credit numbers and ages a credit from when its funds arrived.
func TestEmailReceivables_SendsThePaddedStatement(t *testing.T) {
	t.Parallel()

	f := newStatementFixture(t)
	received := time.Now().AddDate(0, 0, -45)
	f.customerRepo.EXPECT().Get(gomock.Any(), invoiceSellerID, invoiceCustomerID, gomock.Nil()).Return(&domain.Customer{ID: invoiceCustomerID}, nil)
	f.receivableRepo.EXPECT().ListAllByCustomer(gomock.Any(), invoiceSellerID, invoiceCustomerID, gomock.Nil()).
		Return([]domain.ReceivableEntry{{InvoiceID: "iv_1", InvoiceNumber: "1234", InvoicedAt: time.Now(), RemainingBalance: "100.00"}}, nil)
	f.receivableRepo.EXPECT().ListOpenCreditsByCustomer(gomock.Any(), invoiceSellerID, invoiceCustomerID).
		Return([]domain.OpenCredit{{ID: "tr_1", Number: "56", CreatedAt: time.Now(), FundsReceivedAt: &received, LeftoverAmount: "25.00"}}, nil)
	f.accountRepo.EXPECT().GetName(gomock.Any(), gomock.Any()).Return("Buyer Co", nil).AnyTimes()
	f.accountRepo.EXPECT().GetByID(gomock.Any(), invoiceSellerID).Return(nil, nil).AnyTimes()
	f.idempotency.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_1", gomock.Any()).Return(nil)

	var sent messaging.EmailSendData
	f.publisher.EXPECT().PublishSendEmail(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, data messaging.EmailSendData) *apierror.APIError {
			sent = data
			return nil
		})

	apiErr := f.svc.EmailReceivablesForCustomer(statementCtx(), domain.EmailReceivablesParams{
		CustomerAccountID: invoiceCustomerID,
		RecipientEmails:   []string{"ap@example.com"},
	})
	require.Nil(t, apiErr)

	require.NotNil(t, sent.AttachmentData)
	rows := statementRows(t, *sent.AttachmentData)
	require.Len(t, rows, 4, "header, invoice, credit, totals")
	assert.Equal(t, "001234", rows[1][0])
	assert.Equal(t, "Credit: 000056", rows[2][0])
	assert.Equal(t, received.Format("1/2/2006"), rows[2][2], "a credit is dated by when its funds arrived")
	assert.Equal(t, "", rows[2][3], "45 days old is not current")
	assert.NotEqual(t, "", rows[2][4], "45 days old is over 30 days")
}

func TestOpenCredit_AgesFromFundsReceivedElseCreated(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	received := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, received, domain.OpenCredit{CreatedAt: created, FundsReceivedAt: &received}.AgedFrom())
	assert.Equal(t, created, domain.OpenCredit{CreatedAt: created}.AgedFrom())
}

// statementRows reads the attached workbook back as text, a zero amount as the blank its currency format shows.
func statementRows(t *testing.T, attachment string) [][]string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(attachment)
	require.NoError(t, err)
	book, err := excelize.OpenReader(bytes.NewReader(raw))
	require.NoError(t, err)
	t.Cleanup(func() { _ = book.Close() })
	rows, err := book.GetRows(statementSheet, excelize.Options{RawCellValue: true})
	require.NoError(t, err)
	for _, row := range rows {
		for i, cell := range row {
			if cell == "0" {
				row[i] = ""
			}
		}
	}
	return rows
}
