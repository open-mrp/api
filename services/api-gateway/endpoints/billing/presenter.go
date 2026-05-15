package billingep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/billing"
)

func PricingPlanPresenter(p *pb.PricingPlan) apiresource.PricingPlan {
	if p == nil {
		return apiresource.PricingPlan{}
	}

	limitSlice := make([]apiresource.PlanLimit, len(p.Limits))
	for i, l := range p.Limits {
		var value *int
		if l.Value != nil {
			v := int(*l.Value)
			value = &v
		}
		limitSlice[i] = apiresource.PlanLimit{
			Object: constants.ObjectTypePlanLimit,
			Key:    l.Key,
			Value:  value,
		}
	}

	var pricePerMonth *float64
	if p.PricePerMonth != nil {
		pricePerMonth = p.PricePerMonth
	}

	var seatMinimum *int
	if p.SeatMinimum != nil {
		v := int(*p.SeatMinimum)
		seatMinimum = &v
	}

	var includesPreviousPlan *string
	if p.IncludesPreviousPlan != nil {
		includesPreviousPlan = p.IncludesPreviousPlan
	}

	return apiresource.PricingPlan{
		ID:                   p.Id,
		Object:               constants.ObjectTypePricingPlan,
		Name:                 p.Name,
		PlanTypeCode:         constants.PublicPlanCode(p.PlanTypeCode),
		PricePerSeat:         p.PricePerSeat,
		PricePerMonth:        pricePerMonth,
		SeatMinimum:          seatMinimum,
		Limits:               apiresource.NewList(limitSlice, apiresource.PageInfo{}),
		DisplayFeatures:      p.DisplayFeatures,
		DisplayOrder:         int(p.DisplayOrder),
		IsHighlighted:        p.IsHighlighted,
		ButtonText:           p.ButtonText,
		IncludesPreviousPlan: includesPreviousPlan,
	}
}

func PricingPlansListPresenter(resp *pb.ListPricingPlansResponse) *apiresource.List[apiresource.PricingPlan] {
	if resp == nil {
		return apiresource.NewList([]apiresource.PricingPlan{}, apiresource.PageInfo{})
	}

	plans := make([]apiresource.PricingPlan, len(resp.Plans))
	for i, p := range resp.Plans {
		plans[i] = PricingPlanPresenter(p)
	}

	return apiresource.NewList(plans, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func usageItemPresenter(item *pb.UsageItem) apiresource.UsageItem {
	if item == nil {
		return apiresource.UsageItem{}
	}
	ui := apiresource.UsageItem{
		Object:  constants.ObjectTypeUsageItem,
		Current: int(item.Current),
	}
	if item.Limit != nil {
		v := int(*item.Limit)
		ui.Limit = &v
	}
	return ui
}

func PlanChangePreviewPresenter(resp *pb.PreviewPlanChangeResponse) *apiresource.PlanChangeProration {
	if resp == nil || resp.Preview == nil {
		return &apiresource.PlanChangeProration{}
	}

	p := resp.Preview
	lineItems := make([]apiresource.PlanChangeLineItem, len(p.LineItems))
	for i, li := range p.LineItems {
		lineItems[i] = apiresource.PlanChangeLineItem{
			Object:      constants.ObjectTypePlanChangeLineItem,
			Description: li.Description,
			Amount:      li.Amount,
		}
	}

	return &apiresource.PlanChangeProration{
		Object:                     constants.ObjectTypePlanChangeProration,
		NetAmount:                  p.NetAmount,
		FormattedNetAmount:         p.FormattedNetAmount,
		MonthlyBillAmount:          p.MonthlyBillAmount,
		FormattedMonthlyBillAmount: p.FormattedMonthlyBillAmount,
		LineItems:                  apiresource.NewList(lineItems, apiresource.PageInfo{}),
		IsEstimate:                 p.IsEstimate,
	}
}

func SwitchPlanPresenter(resp *pb.SwitchPlanResponse) *apiresource.SwitchPlanResponse {
	if resp == nil {
		return &apiresource.SwitchPlanResponse{}
	}

	return &apiresource.SwitchPlanResponse{
		Object:   constants.ObjectTypeSwitchPlanResponse,
		Success:  resp.Success,
		IntentID: resp.IntentId,
	}
}

func AccountUsagePresenter(resp *pb.GetAccountUsageResponse) *apiresource.AccountUsageResponse {
	if resp == nil {
		return &apiresource.AccountUsageResponse{}
	}

	result := &apiresource.AccountUsageResponse{
		Object:    constants.ObjectTypeAccountUsageResponse,
		Seats:     usageItemPresenter(resp.Seats),
		Invoices:  usageItemPresenter(resp.Invoices),
		Batches:   usageItemPresenter(resp.Batches),
		Sandboxes: usageItemPresenter(resp.Sandboxes),
	}

	if resp.Subscription != nil {
		result.Subscription = &apiresource.SubscriptionInfo{
			Object:           constants.ObjectTypeSubscriptionInfo,
			ServicingStatus:  resp.Subscription.ServicingStatus,
			CollectionStatus: resp.Subscription.CollectionStatus,
		}
	}

	result.AgentSpend = &apiresource.AgentSpendInfo{
		Object:              constants.ObjectTypeAgentSpendInfo,
		EstimatedSpendCents: resp.EstimatedAgentSpendCents,
	}

	if resp.AgentTokenDetail != nil {
		d := resp.AgentTokenDetail
		billingPeriodEnd := ""
		if d.BillingPeriodEnd != nil {
			billingPeriodEnd = d.BillingPeriodEnd.AsTime().UTC().Format("2006-01-02T15:04:05Z")
		}
		result.AgentTokenDetail = &apiresource.AgentTokenDetail{
			Object:                      constants.ObjectTypeAgentTokenDetail,
			IncludedTokens:              d.IncludedTokens,
			UsedTokens:                  d.UsedTokens,
			InputTokens:                 d.InputTokens,
			OutputTokens:                d.OutputTokens,
			AdditionalTokensPurchased:   d.AdditionalTokensPurchased,
			TotalAvailable:              d.TotalAvailable,
			CurrentPeriodCost:           d.CurrentPeriodCost,
			BillingPeriodEnd:            billingPeriodEnd,
			OverageCostPerMillionTokens: d.OverageCostPerMillionTokens,
		}
	}

	return result
}
