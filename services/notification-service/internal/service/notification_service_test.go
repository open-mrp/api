package service

import (
	"context"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	servicemock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/service"
	"github.com/open-mrp/api/services/notification-service/internal/email"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// stubTemplateRenderer is a stub implementation for tests
type stubTemplateRenderer struct{}

func (s *stubTemplateRenderer) RenderTemplate(ctx context.Context, templateID constants.EmailTemplate, params map[string]any) (string, *apierror.APIError) {
	return "<html>Rendered template</html>", nil
}

type NotificationServiceTestSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	notificationSvc  domain.NotificationSvc
	emailLogRepo     *repositorymock.MockEmailLogRepo
	emailSender      *servicemock.MockEmailSender
	templateRenderer email.TemplateRenderer
}

func (suite *NotificationServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.emailLogRepo = repositorymock.NewMockEmailLogRepo(suite.ctrl)
	suite.emailSender = servicemock.NewMockEmailSender(suite.ctrl)
	suite.templateRenderer = &stubTemplateRenderer{}

	suite.notificationSvc = NewNotificationSvc(&NotificationSvcConfig{
		EmailLogRepo:     suite.emailLogRepo,
		EmailSender:      suite.emailSender,
		TemplateRenderer: suite.templateRenderer,
	})
}

func (suite *NotificationServiceTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *NotificationServiceTestSuite) TestSendEnterpriseRequest_Success() {
	ctx := context.Background()
	req := &domain.EnterpriseRequestData{
		AccountID:       "acc_123",
		AccountName:     "Test Account",
		CurrentPlanName: "Professional",
		RequesterName:   "John Doe",
		RequesterEmail:  "john@example.com",
	}

	// Expect email to be sent to sales
	suite.emailSender.EXPECT().
		Send(
			gomock.Any(),
			gomock.Any(),
		).
		DoAndReturn(func(ctx context.Context, data domain.EmailData) (*string, *apierror.APIError) {
			suite.Equal([]string{"support@openmrp.ai"}, data.To)
			suite.Equal("Enterprise Upgrade Request: Test Account", data.Subject)
			suite.Nil(data.SendAs)
			return nil, nil
		}).
		Times(1)

	err := suite.notificationSvc.SendEnterpriseRequest(ctx, req)
	suite.Nil(err)
}

func (suite *NotificationServiceTestSuite) TestSendEnterpriseRequest_EmailFailure() {
	ctx := context.Background()
	req := &domain.EnterpriseRequestData{
		AccountID:       "acc_123",
		AccountName:     "Test Account",
		CurrentPlanName: "Professional",
		RequesterName:   "John Doe",
		RequesterEmail:  "john@example.com",
	}

	// Expect email to fail
	suite.emailSender.EXPECT().
		Send(
			gomock.Any(),
			gomock.Any(),
		).
		Return(nil, apierror.NewInternalError(nil, "Email sending failed")).
		Times(1)

	err := suite.notificationSvc.SendEnterpriseRequest(ctx, req)
	suite.NotNil(err)
	suite.Equal("internal_error", string(err.Code))
}

func (suite *NotificationServiceTestSuite) TestSendEmail_SandboxAccount_SkipsSendAndLogsWithHasSentFalse() {
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		AccountMode: constants.AccountModeSandbox,
	})

	accountID := "acct_sandbox_123"
	data := domain.EmailSendData{
		To:        []string{"user@example.com"},
		Subject:   "Welcome",
		Body:      "<html>Hello</html>",
		AccountID: &accountID,
	}

	// Expect a log entry to be created with HasSent=false
	suite.emailLogRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, log *domain.EmailLog) *apierror.APIError {
			suite.False(log.HasSent)
			suite.Equal(accountID, log.AccountID)
			suite.Equal("Welcome", *log.Subject)
			suite.True(strings.HasPrefix(*log.SesMessageID, "sandbox_"))
			return nil
		}).
		Times(1)

	// emailSender.Send should NOT be called
	suite.emailSender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)

	messageID, apiErr := suite.notificationSvc.SendEmail(ctx, data)
	suite.Nil(apiErr)
	suite.NotNil(messageID)
	suite.True(strings.HasPrefix(*messageID, "sandbox_"))
}

func (suite *NotificationServiceTestSuite) TestSendEmail_ProductionAccount_SendsNormally() {
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		AccountMode: constants.AccountModeProduction,
	})

	expectedID := "ses_msg_123"
	data := domain.EmailSendData{
		To:      []string{"user@example.com"},
		Subject: "Welcome",
		Body:    "<html>Hello</html>",
	}

	suite.emailSender.EXPECT().
		Send(gomock.Any(), gomock.Any()).
		Return(&expectedID, nil).
		Times(1)

	// emailLogRepo.Create should NOT be called (logging happens via event flow)
	suite.emailLogRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	messageID, apiErr := suite.notificationSvc.SendEmail(ctx, data)
	suite.Nil(apiErr)
	suite.Equal(&expectedID, messageID)
}

func (suite *NotificationServiceTestSuite) TestSendEnterpriseRequest_SandboxAccount_SkipsSend() {
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		AccountMode: constants.AccountModeSandbox,
	})

	req := &domain.EnterpriseRequestData{
		AccountID:       "acc_123",
		AccountName:     "Test Account",
		CurrentPlanName: "Professional",
		RequesterName:   "John Doe",
		RequesterEmail:  "john@example.com",
	}

	// Neither email sender nor template renderer should be called
	suite.emailSender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)

	err := suite.notificationSvc.SendEnterpriseRequest(ctx, req)
	suite.Nil(err)
}

// A failed send used to write nothing at all, so a dead email was indistinguishable from one that was never triggered.
func (suite *NotificationServiceTestSuite) TestLogFailedEmail_WritesUnsentLogWithRecipients() {
	ctx := context.Background()
	accountID := "ac_123"

	suite.emailLogRepo.EXPECT().
		FindBySesMessageID(gomock.Any(), "failed_mg_abc").
		Return(nil, apierror.NewResourceNotFoundError("not found"))

	var created *domain.EmailLog
	suite.emailLogRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, el *domain.EmailLog) { created = el }).
		Return(nil)

	apiErr := suite.notificationSvc.LogFailedEmail(ctx, "mg_abc", domain.EmailSendData{
		To:        []string{"kurt@example.com", "ap@example.com"},
		Subject:   "Your Order Checkout - 23124",
		AccountID: &accountID,
	})
	suite.Nil(apiErr)

	suite.Require().NotNil(created)
	suite.False(created.HasSent, "a failed send must not be logged as sent")
	suite.Equal(accountID, created.AccountID)
	suite.Equal([]string{"kurt@example.com", "ap@example.com"}, created.Recipients)
	suite.Require().NotNil(created.SesMessageID)
	suite.Equal("failed_mg_abc", *created.SesMessageID)
}

// Delivery is retried, so without deduplication one dead email would litter the log with a row per attempt.
func (suite *NotificationServiceTestSuite) TestLogFailedEmail_RetryDoesNotDuplicate() {
	ctx := context.Background()

	suite.emailLogRepo.EXPECT().
		FindBySesMessageID(gomock.Any(), "failed_mg_abc").
		Return(&domain.EmailLog{ID: "emlg_existing"}, nil)

	suite.emailLogRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	apiErr := suite.notificationSvc.LogFailedEmail(ctx, "mg_abc", domain.EmailSendData{
		Subject: "Your Order Checkout - 23124",
	})
	suite.Nil(apiErr)
}

// Without a message ID there is no stable dedup key, so logging would append a fresh row on every retry.
func (suite *NotificationServiceTestSuite) TestLogFailedEmail_NoMessageIDSkips() {
	ctx := context.Background()

	suite.emailLogRepo.EXPECT().FindBySesMessageID(gomock.Any(), gomock.Any()).Times(0)
	suite.emailLogRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	apiErr := suite.notificationSvc.LogFailedEmail(ctx, "", domain.EmailSendData{Subject: "x"})
	suite.Nil(apiErr)
}

// Emails sent through the Go path were logged without recipients, leaving the list's recipient column blank and unsearchable.
func (suite *NotificationServiceTestSuite) TestLogEmail_PersistsRecipients() {
	ctx := context.Background()
	accountID := "ac_123"

	suite.emailLogRepo.EXPECT().
		FindBySesMessageID(gomock.Any(), "ses_1").
		Return(nil, apierror.NewResourceNotFoundError("not found"))

	var created *domain.EmailLog
	suite.emailLogRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, el *domain.EmailLog) { created = el }).
		Return(nil)

	apiErr := suite.notificationSvc.LogEmail(ctx, domain.EmailLogData{
		SesMessageID: "ses_1",
		To:           []string{"kurt@example.com"},
		AccountID:    &accountID,
		Subject:      "Sales Order 023144",
	})
	suite.Nil(apiErr)

	suite.Require().NotNil(created)
	suite.True(created.HasSent)
	suite.Equal([]string{"kurt@example.com"}, created.Recipients)
}

func TestNotificationServiceTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(NotificationServiceTestSuite))
}
