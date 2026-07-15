package billingep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/billing"
)

func TestPricingPlanPresenter(t *testing.T) {
	t.Parallel()
	pricePerMonth := 19.0
	seatMin := int32(1)
	includesPrev := "Free"
	limitVal := int32(5)

	plan := &pb.PricingPlan{
		Id:            "pt_01abc",
		Name:          "Starter",
		PlanTypeCode:  "starter",
		PricePerSeat:  19.0,
		PricePerMonth: &pricePerMonth,
		SeatMinimum:   &seatMin,
		Limits: []*pb.PlanLimit{
			{Key: "seats_maximum", Value: &limitVal},
		},
		DisplayFeatures:      []string{"5 seats"},
		DisplayOrder:         2,
		IsHighlighted:        true,
		ButtonText:           "Start Trial",
		IncludesPreviousPlan: &includesPrev,
	}

	result := pricingPlanFromProto(plan)
	resourcetest.ValidateResourceStruct(t, "PricingPlan", result)
}

func TestPlanChangePreviewPresenter(t *testing.T) {
	t.Parallel()
	resp := &pb.PreviewPlanChangeResponse{
		Preview: &pb.PlanChangePreview{
			NetAmount:                  4900,
			FormattedNetAmount:         "$49.00",
			MonthlyBillAmount:          4900,
			FormattedMonthlyBillAmount: "$49.00",
			LineItems: []*pb.PlanChangePreviewLineItem{
				{Description: "Pro plan", Amount: 4900},
			},
			IsEstimate: true,
		},
	}

	result := planChangePreviewFromProto(resp)
	resourcetest.ValidateResourceStruct(t, "PlanChangeProration", result)
}

func TestAccountUsagePresenter(t *testing.T) {
	t.Parallel()
	limit := int32(10)

	resp := &pb.GetAccountUsageResponse{
		Seats:     &pb.UsageItem{Current: 5, Limit: &limit},
		Invoices:  &pb.UsageItem{Current: 100},
		Batches:   &pb.UsageItem{Current: 50},
		Sandboxes: &pb.UsageItem{Current: 1, Limit: &limit},
		Subscription: &pb.SubscriptionInfo{
			ServicingStatus:  "active",
			CollectionStatus: "current",
		},
		EstimatedAgentSpendCents: 4500,
		PlanName:                 "Founder",
		BaseFeeCents:             100,
		BaseFeeInterval:          "month",
	}

	result := accountUsageFromProto(resp)
	resourcetest.ValidateResourceStruct(t, "AccountUsageResponse", result)
}

func TestSwitchPlanPresenter(t *testing.T) {
	t.Parallel()
	intentID := "bi_test_123"

	resp := &pb.SwitchPlanResponse{
		Success:  true,
		IntentId: &intentID,
	}

	result := switchPlanFromProto(resp)
	resourcetest.ValidateResourceStruct(t, "SwitchPlanResponse", result)
}
