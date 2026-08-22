package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

const costAccountID = "acct_test"

type CostBasisConsumerTestSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	consumer *ItemCostBasisChangedConsumer
	itemRepo *repositorymock.MockItemRepo
}

func (s *CostBasisConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.itemRepo = repositorymock.NewMockItemRepo(s.ctrl)

	repoFactory := factorymock.NewMockRepoFactory(s.ctrl)
	repoFactory.EXPECT().NewItemRepo().Return(s.itemRepo).AnyTimes()

	s.consumer = &ItemCostBasisChangedConsumer{
		repos:  repoFactory,
		tracer: tracing.GetTracer("test.item_cost_basis_changed_consumer"),
	}
}

func (s *CostBasisConsumerTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestCostBasisConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CostBasisConsumerTestSuite))
}

// graph stubs the walk: each call returns what the given frontier produces.
func (s *CostBasisConsumerTestSuite) graph(levels map[string][]string) {
	s.itemRepo.EXPECT().
		FindItemsProducedFromConsumed(gomock.Any(), costAccountID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, itemIDs []string) ([]string, *apierror.APIError) {
			var out []string
			for _, id := range itemIDs {
				out = append(out, levels[id]...)
			}
			return out, nil
		}).AnyTimes()
}

// The item where the change happened is recosted too, and first: a material whose price moved is
// the reason everything after it is stale.
func (s *CostBasisConsumerTestSuite) TestIncludesOriginItemFirst() {
	s.graph(map[string][]string{})

	affected, err := s.consumer.affectedItems(context.Background(), costAccountID, "it_yarn")

	s.Require().Nil(err)
	s.Equal([]string{"it_yarn"}, affected)
}

// Nearest first, so each item is costed from inputs this same pass has already corrected. Yarn makes
// greige, greige makes a finished sock; costing the sock before the greige would use a stale part.
func (s *CostBasisConsumerTestSuite) TestWalksDownstreamNearestFirst() {
	s.graph(map[string][]string{
		"it_yarn":   {"it_greige"},
		"it_greige": {"it_finished"},
	})

	affected, err := s.consumer.affectedItems(context.Background(), costAccountID, "it_yarn")

	s.Require().Nil(err)
	s.Equal([]string{"it_yarn", "it_greige", "it_finished"}, affected)
}

// One material can feed many parts, and each of those many products.
func (s *CostBasisConsumerTestSuite) TestFansOutAcrossEveryDependent() {
	s.graph(map[string][]string{
		"it_yarn": {"it_greige_a", "it_greige_b"},
	})

	affected, err := s.consumer.affectedItems(context.Background(), costAccountID, "it_yarn")

	s.Require().Nil(err)
	s.Equal([]string{"it_yarn", "it_greige_a", "it_greige_b"}, affected)
}

// An item reachable by two routings is costed once, not once per path.
func (s *CostBasisConsumerTestSuite) TestVisitsADiamondOnce() {
	s.graph(map[string][]string{
		"it_yarn":     {"it_greige_a", "it_greige_b"},
		"it_greige_a": {"it_finished"},
		"it_greige_b": {"it_finished"},
	})

	affected, err := s.consumer.affectedItems(context.Background(), costAccountID, "it_yarn")

	s.Require().Nil(err)
	s.Equal([]string{"it_yarn", "it_greige_a", "it_greige_b", "it_finished"}, affected)
}

// A routing that loops back on itself must terminate. The dedup is what stops it; the depth bound
// is the second guard.
func (s *CostBasisConsumerTestSuite) TestTerminatesOnACycle() {
	s.graph(map[string][]string{
		"it_a": {"it_b"},
		"it_b": {"it_c"},
		"it_c": {"it_a"},
	})

	affected, err := s.consumer.affectedItems(context.Background(), costAccountID, "it_a")

	s.Require().Nil(err)
	s.Equal([]string{"it_a", "it_b", "it_c"}, affected)
}

// A chain longer than the bound stops at it rather than walking forever.
func (s *CostBasisConsumerTestSuite) TestStopsAtMaxDepth() {
	levels := map[string][]string{}
	for i := range 40 {
		levels[itemName(i)] = []string{itemName(i + 1)}
	}
	s.graph(levels)

	affected, err := s.consumer.affectedItems(context.Background(), costAccountID, itemName(0))

	s.Require().Nil(err)
	// The origin, plus one generation per level walked.
	s.Len(affected, maxCostGraphDepth+1)
	s.Equal(itemName(0), affected[0])
}

// A failure part-way through the walk surfaces rather than silently returning a truncated set, which
// would look like a completed recost of fewer items.
func (s *CostBasisConsumerTestSuite) TestWalkFailureIsReported() {
	s.itemRepo.EXPECT().
		FindItemsProducedFromConsumed(gomock.Any(), costAccountID, gomock.Any()).
		Return(nil, apierror.NewInternalError(nil, "boom"))

	affected, err := s.consumer.affectedItems(context.Background(), costAccountID, "it_yarn")

	s.Require().NotNil(err)
	s.Nil(affected)
}

// An item with no dependents leaves nothing else to do.
func (s *CostBasisConsumerTestSuite) TestLeafItemHasNoDependents() {
	s.graph(map[string][]string{"it_yarn": nil})

	affected, err := s.consumer.affectedItems(context.Background(), costAccountID, "it_yarn")

	s.Require().Nil(err)
	s.Equal([]string{"it_yarn"}, affected)
}

func itemName(i int) string {
	return "it_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
}
