package service

import (
	"context"
	"strings"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/notification-service/internal/domain"
	repositorymock "github.com/augno/api/services/notification-service/internal/domain/mock/repository"
	servicemock "github.com/augno/api/services/notification-service/internal/domain/mock/service"
	"github.com/augno/api/services/notification-service/internal/email"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

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
			suite.Equal([]string{"sales@augno.com"}, data.To)
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

func TestNotificationServiceTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationServiceTestSuite))
}
