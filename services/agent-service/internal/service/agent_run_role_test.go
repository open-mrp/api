package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/client"
	factorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/repository"
	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"go.uber.org/mock/gomock"
)

const (
	driveDefID  = "agd_wide"
	driveRoleID = "rl_agent"
	driveRunID  = "agr_drive"
)

// errReachedTx is returned by the stub transaction so a test can tell a run got past every check.
var errReachedTx = apierror.NewValidationError("reached the run transaction")

type stopTxManager struct{}

func (stopTxManager) WithTx(context.Context, func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return errReachedTx
}

type driveCase struct {
	name      string
	identity  *types.Identity
	roleType  constants.RoleType
	rolePerms map[string]bool
	wantMsg   string
}

func driveCases() []driveCase {
	agentPerms := map[string]bool{"products:read": true, "roles:update": true, "users:create": true}
	return []driveCase{
		{
			name:      "non-admin missing the agent's permissions is refused",
			identity:  roleTestIdentity(constants.RoleTypeCustom, "agent_runs:create", "agent_runs:update", "products:read"),
			roleType:  constants.RoleTypeCustom,
			rolePerms: agentPerms,
			wantMsg:   "roles:update, users:create",
		},
		{
			name:     "non-admin cannot drive an admin-role agent",
			identity: roleTestIdentity(constants.RoleTypeCustom, "agent_runs:create", "agent_runs:update", "products:read", "roles:update", "users:create"),
			roleType: constants.RoleTypeAdmin,
			wantMsg:  "admin role",
		},
		{
			name:      "non-admin holding every agent permission may drive it",
			identity:  roleTestIdentity(constants.RoleTypeCustom, "agent_runs:create", "agent_runs:update", "products:read", "roles:update", "users:create"),
			roleType:  constants.RoleTypeCustom,
			rolePerms: agentPerms,
		},
		{
			name:     "admin may drive any agent",
			identity: roleTestIdentity(constants.RoleTypeAdmin),
			roleType: constants.RoleTypeAdmin,
		},
	}
}

func expectAgentRole(core *clientmock.MockCoreClient, tc driveCase) {
	if tc.identity.IsAdmin() {
		return
	}
	acct := testAccountID
	core.EXPECT().GetRoleInfo(gomock.Any(), driveRoleID).Return(&domain.RoleInfo{ID: driveRoleID, RoleType: string(tc.roleType), AccountID: &acct}, nil)
	if tc.roleType != constants.RoleTypeAdmin {
		core.EXPECT().GetRolePermissions(gomock.Any(), driveRoleID).Return(tc.rolePerms, nil)
	}
}

func driveDef() *sqlc.AgentDefinition {
	return &sqlc.AgentDefinition{ID: driveDefID, Name: "Wide agent", Slug: "wide-agent", RoleID: agentdb.PgText(driveRoleID)}
}

func newDriveSvc(ctrl *gomock.Controller, factory *factorymock.MockRepoFactory, core *clientmock.MockCoreClient) *agentDefSvcImpl {
	idem := mediatormock.NewMockIdempotencyMed(ctrl)
	idem.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{TypeID: "idk_1", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil).AnyTimes()
	idem.EXPECT().CacheErrorResponse(gomock.Any(), "idk_1", gomock.Any()).DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
		return apiErr
	}).AnyTimes()
	meds := mediatormock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: idem}).AnyTimes()
	return &agentDefSvcImpl{repos: factory, mediatorFactory: meds, txManager: stopTxManager{}, coreClient: core}
}

func assertDriveResult(t *testing.T, tc driveCase, apiErr *apierror.APIError) {
	t.Helper()
	if tc.wantMsg == "" {
		if apiErr != errReachedTx {
			t.Fatalf("expected the run to pass the role check, got %+v", apiErr)
		}
		return
	}
	if apiErr == nil || apiErr == errReachedTx {
		t.Fatalf("expected a 403, got %+v", apiErr)
	}
	if got := apierror.GetHTTPStatusCode(apiErr.Code); got != http.StatusForbidden {
		t.Errorf("status = %d, want 403", got)
	}
	if !strings.Contains(apiErr.PublicMessage, tc.wantMsg) {
		t.Errorf("message %q does not contain %q", apiErr.PublicMessage, tc.wantMsg)
	}
}

func TestTriggerRunRequiresTheAgentsRole(t *testing.T) {
	for _, tc := range driveCases() {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			core := clientmock.NewMockCoreClient(ctrl)
			expectAgentRole(core, tc)
			defRepo := repositorymock.NewMockAgentDefinitionRepo(ctrl)
			defRepo.EXPECT().GetBySlug(gomock.Any(), "wide-agent").Return(driveDef(), nil)
			configRepo := repositorymock.NewMockAgentConfigRepo(ctrl)
			configRepo.EXPECT().GetByAccountAndDefinition(gomock.Any(), testAccountID, driveDefID).Return(&sqlc.AgentConfig{ID: "agc_1"}, nil).AnyTimes()
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewAgentDefinitionRepo().Return(defRepo).AnyTimes()
			factory.EXPECT().NewAgentConfigRepo().Return(configRepo).AnyTimes()

			s := newDriveSvc(ctrl, factory, core)
			_, apiErr := s.TriggerRun(appctx.WithIdentity(context.Background(), tc.identity), domain.TriggerRunParams{AgentDefinitionCode: "wide-agent"})
			assertDriveResult(t, tc, apiErr)
		})
	}
}

func TestContinueRunRequiresTheAgentsRole(t *testing.T) {
	for _, tc := range driveCases() {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			core := clientmock.NewMockCoreClient(ctrl)
			expectAgentRole(core, tc)
			runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
			runRepo.EXPECT().GetByID(gomock.Any(), driveRunID).Return(&sqlc.AgentRun{ID: driveRunID, AccountID: testAccountID, AgentDefinitionID: driveDefID, StatusCode: domain.RunStatusAwaitingInput}, nil)
			defRepo := repositorymock.NewMockAgentDefinitionRepo(ctrl)
			defRepo.EXPECT().GetByID(gomock.Any(), driveDefID).Return(driveDef(), nil).AnyTimes()
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewAgentRunRepo().Return(runRepo).AnyTimes()
			factory.EXPECT().NewAgentDefinitionRepo().Return(defRepo).AnyTimes()

			s := newDriveSvc(ctrl, factory, core)
			_, apiErr := s.ContinueRun(appctx.WithIdentity(context.Background(), tc.identity), domain.ContinueRunParams{AgentRunID: driveRunID, Message: "go on"})
			assertDriveResult(t, tc, apiErr)
		})
	}
}

func TestRetryRunRequiresTheAgentsRole(t *testing.T) {
	for _, tc := range driveCases() {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			core := clientmock.NewMockCoreClient(ctrl)
			expectAgentRole(core, tc)
			runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
			runRepo.EXPECT().GetByID(gomock.Any(), driveRunID).Return(&sqlc.AgentRun{ID: driveRunID, AccountID: testAccountID, AgentDefinitionID: driveDefID, StatusCode: domain.RunStatusFailed}, nil)
			defRepo := repositorymock.NewMockAgentDefinitionRepo(ctrl)
			defRepo.EXPECT().GetByID(gomock.Any(), driveDefID).Return(driveDef(), nil).AnyTimes()
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewAgentRunRepo().Return(runRepo).AnyTimes()
			factory.EXPECT().NewAgentDefinitionRepo().Return(defRepo).AnyTimes()

			s := newDriveSvc(ctrl, factory, core)
			_, apiErr := s.RetryRun(appctx.WithIdentity(context.Background(), tc.identity), domain.RetryRunParams{AgentRunID: driveRunID})
			assertDriveResult(t, tc, apiErr)
		})
	}
}

func TestDriveAgentWithoutRoleNeedsNoCheck(t *testing.T) {
	s := &agentDefSvcImpl{}
	if apiErr := s.checkCanDriveAgent(context.Background(), grantOf(roleTestIdentity(constants.RoleTypeCustom)), agentdb.PgText("")); apiErr != nil {
		t.Fatalf("an agent without a role grants nothing to borrow: %v", apiErr.PublicMessage)
	}
}

func TestCreateChatRunRequiresTheSendersRole(t *testing.T) {
	for _, tc := range driveCases() {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			core := clientmock.NewMockCoreClient(ctrl)
			core.EXPECT().GetUserAccess(gomock.Any(), "us_sender", testAccountID).Return(&domain.UserAccess{
				RoleType:    *tc.identity.Actor.RoleType,
				Permissions: tc.identity.Actor.Permissions,
			}, nil)
			expectAgentRole(core, tc)
			defRepo := repositorymock.NewMockAgentDefinitionRepo(ctrl)
			defRepo.EXPECT().GetByID(gomock.Any(), driveDefID).Return(driveDef(), nil)
			configRepo := repositorymock.NewMockAgentConfigRepo(ctrl)
			configRepo.EXPECT().GetByAccountAndDefinition(gomock.Any(), testAccountID, driveDefID).Return(&sqlc.AgentConfig{ID: "agc_1"}, nil).AnyTimes()
			outbox := &fakeOutbox{}
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewAgentDefinitionRepo().Return(defRepo).AnyTimes()
			factory.EXPECT().NewAgentConfigRepo().Return(configRepo).AnyTimes()
			factory.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

			s := newDriveSvc(ctrl, factory, core)
			apiErr := s.CreateChatRun(context.Background(), domain.ChatRunInput{
				AccountID:         testAccountID,
				AgentDefinitionID: driveDefID,
				ConversationID:    "cnv_1",
				TriggerMessageID:  "msg_trigger",
				Message:           "@wide do it",
				SenderUserID:      "us_sender",
			})

			if tc.wantMsg == "" {
				if apiErr != errReachedTx {
					t.Fatalf("expected the chat run to pass the role check, got %+v", apiErr)
				}
				return
			}
			if apiErr != nil {
				t.Fatalf("a refused chat trigger is answered, not retried: %+v", apiErr)
			}
			if len(outbox.inputs) != 1 || outbox.inputs[0].RoutingKey != string(contracts.NotificationCmdAgentReply) {
				t.Fatalf("expected one agent reply explaining the refusal, got %+v", outbox.inputs)
			}
			var reply messaging.AgentReplyData
			if err := json.Unmarshal(outbox.inputs[0].Payload.Data, &reply); err != nil {
				t.Fatal(err)
			}
			if !reply.Failed || reply.ReplyToMessageID != "msg_trigger" || reply.AgentConfigID != driveDefID || reply.AgentRunID != "" {
				t.Errorf("unexpected denial reply: %+v", reply)
			}
			if !strings.Contains(reply.Body, tc.wantMsg) {
				t.Errorf("reply %q does not contain %q", reply.Body, tc.wantMsg)
			}
		})
	}
}

func TestCreateChatRunRefusesAFormerMember(t *testing.T) {
	ctrl := gomock.NewController(t)
	core := clientmock.NewMockCoreClient(ctrl)
	core.EXPECT().GetUserAccess(gomock.Any(), "us_gone", testAccountID).Return(nil, nil)
	defRepo := repositorymock.NewMockAgentDefinitionRepo(ctrl)
	defRepo.EXPECT().GetByID(gomock.Any(), driveDefID).Return(driveDef(), nil)
	outbox := &fakeOutbox{}
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAgentDefinitionRepo().Return(defRepo).AnyTimes()
	factory.EXPECT().NewAgentConfigRepo().Return(repositorymock.NewMockAgentConfigRepo(ctrl)).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

	s := newDriveSvc(ctrl, factory, core)
	// An "always" agent has no trigger message, so the refusal is silent rather than repeated on every message.
	if apiErr := s.CreateChatRun(context.Background(), domain.ChatRunInput{
		AccountID:         testAccountID,
		AgentDefinitionID: driveDefID,
		ConversationID:    "cnv_1",
		Message:           "hello",
		SenderUserID:      "us_gone",
	}); apiErr != nil {
		t.Fatalf("unexpected error: %+v", apiErr)
	}
	if len(outbox.inputs) != 0 {
		t.Errorf("expected no run and no reply, got %+v", outbox.inputs)
	}
}

func TestCreateChatRunWithoutAMemberSenderSkipsTheCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	core := clientmock.NewMockCoreClient(ctrl)
	defRepo := repositorymock.NewMockAgentDefinitionRepo(ctrl)
	defRepo.EXPECT().GetByID(gomock.Any(), driveDefID).Return(driveDef(), nil)
	configRepo := repositorymock.NewMockAgentConfigRepo(ctrl)
	configRepo.EXPECT().GetByAccountAndDefinition(gomock.Any(), testAccountID, driveDefID).Return(&sqlc.AgentConfig{ID: "agc_1"}, nil)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAgentDefinitionRepo().Return(defRepo).AnyTimes()
	factory.EXPECT().NewAgentConfigRepo().Return(configRepo).AnyTimes()

	s := newDriveSvc(ctrl, factory, core)
	apiErr := s.CreateChatRun(context.Background(), domain.ChatRunInput{
		AccountID:         testAccountID,
		AgentDefinitionID: driveDefID,
		ConversationID:    "cnv_1",
		TriggerMessageID:  "msg_email",
		Message:           "an inbound email",
	})
	if apiErr != errReachedTx {
		t.Fatalf("expected the run to start, got %+v", apiErr)
	}
}

func TestContinueChatRunIgnoresAnotherAgentsRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
	runRepo.EXPECT().GetByID(gomock.Any(), driveRunID).Return(&sqlc.AgentRun{ID: driveRunID, AccountID: testAccountID, AgentDefinitionID: "agd_other", StatusCode: domain.RunStatusAwaitingInput}, nil)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAgentRunRepo().Return(runRepo)

	s := &agentDefSvcImpl{repos: factory}
	continued, apiErr := s.continueChatRun(context.Background(), domain.ChatRunInput{AccountID: testAccountID, AgentDefinitionID: driveDefID, ContinueRunID: driveRunID})
	if apiErr != nil || continued {
		t.Fatalf("a run of another agent must not be continued under this agent's check: continued=%v err=%+v", continued, apiErr)
	}
}

func TestCreateChatRunIgnoresAnotherAccountsAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	def := driveDef()
	def.AccountID = agentdb.PgText("ac_other")
	defRepo := repositorymock.NewMockAgentDefinitionRepo(ctrl)
	defRepo.EXPECT().GetByID(gomock.Any(), driveDefID).Return(def, nil)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAgentDefinitionRepo().Return(defRepo).AnyTimes()
	factory.EXPECT().NewAgentConfigRepo().Return(repositorymock.NewMockAgentConfigRepo(ctrl)).AnyTimes()

	s := newDriveSvc(ctrl, factory, clientmock.NewMockCoreClient(ctrl))
	if apiErr := s.CreateChatRun(context.Background(), domain.ChatRunInput{
		AccountID:         testAccountID,
		AgentDefinitionID: driveDefID,
		ConversationID:    "cnv_1",
		Message:           "hello",
		SenderUserID:      "us_sender",
	}); apiErr != nil {
		t.Fatalf("unexpected error: %+v", apiErr)
	}
}
