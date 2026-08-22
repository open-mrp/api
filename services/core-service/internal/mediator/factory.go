package mediator

import "github.com/open-mrp/api/services/core-service/internal/domain"

type mediatorFactoryImpl struct{}

func NewMediatorFactory() domain.MediatorFactory {
	return &mediatorFactoryImpl{}
}

func (f *mediatorFactoryImpl) Build(repoFactory domain.RepoFactory) domain.Mediators {
	return domain.Mediators{
		Sandbox:        NewSandboxMed(&SandboxMedConfig{Repos: repoFactory}),
		Idempotency:    NewIdempotencyMed(&IdempotencyMedConfig{Repos: repoFactory}),
		ReadAccess:     NewReadAccessMed(&ReadAccessMedConfig{Repos: repoFactory}),
		EditAccess:     NewEditAccessMed(&EditAccessMedConfig{Repos: repoFactory}),
		ProductionFlow: NewProductionFlowMed(repoFactory),
		BurnRate:       NewBurnRateMed(&BurnRateMedConfig{Repos: repoFactory}),
	}
}
