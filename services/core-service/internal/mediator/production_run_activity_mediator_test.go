package mediator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

const (
	activityAccountID = "acct_run"
	activityRunID     = "prun_1"
	activityOwner     = "acus_owner"
)

type ProductionRunActivityMedTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	med             domain.ProductionRunActivityMed
	accountUserRepo *repositorymock.MockAccountUserRepo
	runRepo         *repositorymock.MockProductionRunRepo
	outbox          *recordingOutboxRepo
	run             *domain.ProductionRun
}

func (s *ProductionRunActivityMedTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.accountUserRepo = repositorymock.NewMockAccountUserRepo(s.ctrl)
	s.runRepo = repositorymock.NewMockProductionRunRepo(s.ctrl)
	s.outbox = &recordingOutboxRepo{}

	repos := factorymock.NewMockRepoFactory(s.ctrl)
	repos.EXPECT().NewAccountUserRepo().Return(s.accountUserRepo).AnyTimes()
	repos.EXPECT().NewProductionRunRepo().Return(s.runRepo).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(s.outbox).AnyTimes()

	s.med = NewProductionRunActivityMed(&ProductionRunActivityMedConfig{Repos: repos})
	s.run = &domain.ProductionRun{ID: activityRunID, Number: "PR-7", AccountID: activityAccountID, ResponsibleUserID: "us_owner", BatchCount: 12}
}

func (s *ProductionRunActivityMedTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestProductionRunActivityMedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ProductionRunActivityMedTestSuite))
}

func activityIdentity(identityType types.IdentityActorType, actorID string) *types.Identity {
	name := "Casey Doe"
	accountID := activityAccountID
	return &types.Identity{
		Type:   identityType,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor:  &types.IdentityActor{ID: actorID, Name: &name, AccountID: &accountID, RelationType: types.IdentityRelationTypeInternal},
	}
}

func (s *ProductionRunActivityMedTestSuite) expectResolve(id, accountUserID string) {
	s.accountUserRepo.EXPECT().ResolveAccountUserID(gomock.Any(), activityAccountID, id).Return(accountUserID, nil)
}

func (s *ProductionRunActivityMedTestSuite) alerts() []messaging.AlertFanoutData {
	var alerts []messaging.AlertFanoutData
	for _, msg := range s.outbox.messages {
		if msg.RoutingKey != string(contracts.NotificationCmdFanout) {
			continue
		}
		var data messaging.AlertFanoutData
		s.Require().NoError(json.Unmarshal(msg.Payload.Data, &data))
		alerts = append(alerts, data)
	}
	return alerts
}

func (s *ProductionRunActivityMedTestSuite) TestAlertsTheOwnerWhenSomeoneElseAddsBatches() {
	s.expectResolve("us_owner", activityOwner)
	s.expectResolve("us_other", "acus_other")

	apiErr := s.med.NotifyBatchesAdded(context.Background(), activityIdentity(types.IdentityActorTypeUser, "us_other"), s.run, 3)

	s.Require().Nil(apiErr)
	alerts := s.alerts()
	s.Require().Len(alerts, 1)
	s.Equal(activityAccountID, alerts[0].AccountID)
	s.Equal(string(constants.NotificationCategoryProductionRunUpdated), alerts[0].Category)
	s.Equal([]string{activityOwner}, alerts[0].RecipientAccountUserIDs)
	s.Equal("Production run PR-7 updated", alerts[0].Title)
	s.Equal("Casey Doe added 3 batches.", alerts[0].Body)
	s.Equal(string(constants.ObjectTypeProductionRun), alerts[0].LinkResourceType)
	s.Equal(activityRunID, alerts[0].LinkResourceID)
	s.Contains(alerts[0].DedupeKey, "runact_"+activityRunID+"_")
}

func (s *ProductionRunActivityMedTestSuite) TestSkipsTheOwnersOwnChange() {
	s.accountUserRepo.EXPECT().ResolveAccountUserID(gomock.Any(), activityAccountID, "us_owner").Return(activityOwner, nil).Times(2)

	apiErr := s.med.NotifyBatchesAdded(context.Background(), activityIdentity(types.IdentityActorTypeUser, "us_owner"), s.run, 1)

	s.Require().Nil(apiErr)
	s.Empty(s.alerts())
}

func (s *ProductionRunActivityMedTestSuite) TestSkipsAnOwnerNoLongerActiveInTheAccount() {
	s.accountUserRepo.EXPECT().ResolveAccountUserID(gomock.Any(), activityAccountID, "us_owner").Return("", apierror.NewResourceNotFoundError("Account user not found."))

	apiErr := s.med.NotifyBatchesAdded(context.Background(), activityIdentity(types.IdentityActorTypeUser, "us_other"), s.run, 1)

	s.Require().Nil(apiErr)
	s.Empty(s.alerts())
}

func (s *ProductionRunActivityMedTestSuite) TestAlertsOnAnAPIKeyChange() {
	s.expectResolve("us_owner", activityOwner)

	apiErr := s.med.NotifyBatchesAdded(context.Background(), activityIdentity(types.IdentityActorTypeAPIKey, "apik_1"), s.run, 1)

	s.Require().Nil(apiErr)
	alerts := s.alerts()
	s.Require().Len(alerts, 1)
	s.Equal("Casey Doe added a batch.", alerts[0].Body)
}

func (s *ProductionRunActivityMedTestSuite) TestNamesTheDeletedBatchBySKU() {
	s.runRepo.EXPECT().Get(gomock.Any(), domain.GetProductionRunParams{ProductionRunID: activityRunID, AccountID: activityAccountID}).Return(s.run, nil)
	s.expectResolve("us_owner", activityOwner)
	s.expectResolve("us_other", "acus_other")
	batch := &domain.Batch{ID: "bt_1", Item: domain.LightItem{SKU: "WIDGET-BLUE"}, ProductionRun: &domain.LightProductionRun{ID: activityRunID}}

	apiErr := s.med.NotifyBatchDeleted(context.Background(), activityIdentity(types.IdentityActorTypeUser, "us_other"), activityAccountID, batch)

	s.Require().Nil(apiErr)
	alerts := s.alerts()
	s.Require().Len(alerts, 1)
	s.Equal("Casey Doe deleted a WIDGET-BLUE batch.", alerts[0].Body)
	s.Equal(activityRunID, alerts[0].LinkResourceID)
}

func (s *ProductionRunActivityMedTestSuite) TestIgnoresADeletedBatchNotOnARun() {
	apiErr := s.med.NotifyBatchDeleted(context.Background(), activityIdentity(types.IdentityActorTypeUser, "us_other"), activityAccountID, &domain.Batch{ID: "bt_1"})

	s.Require().Nil(apiErr)
	s.Empty(s.alerts())
}

func (s *ProductionRunActivityMedTestSuite) TestADeletedRunAlertsWithoutALinkOrRollingRow() {
	s.expectResolve("us_owner", activityOwner)
	s.expectResolve("us_other", "acus_other")

	apiErr := s.med.NotifyRunDeleted(context.Background(), activityIdentity(types.IdentityActorTypeUser, "us_other"), s.run)

	s.Require().Nil(apiErr)
	alerts := s.alerts()
	s.Require().Len(alerts, 1)
	s.Equal("Production run PR-7 deleted", alerts[0].Title)
	s.Equal("Casey Doe deleted the production run and its 12 batches.", alerts[0].Body)
	s.Empty(alerts[0].LinkResourceType)
	s.Empty(alerts[0].LinkResourceID)
	s.Empty(alerts[0].DedupeKey)
}
