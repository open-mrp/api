package mediator

import "github.com/open-mrp/api/services/agent-service/internal/domain"

type mediatorFactoryImpl struct{}

func NewMediatorFactory() domain.MediatorFactory {
	return &mediatorFactoryImpl{}
}

func (f *mediatorFactoryImpl) Build(repoFactory domain.RepoFactory) domain.Mediators {
	return domain.Mediators{
		Idempotency: NewIdempotencyMed(&IdempotencyMedConfig{Repos: repoFactory}),
	}
}
