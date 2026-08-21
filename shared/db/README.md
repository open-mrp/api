# Transaction Manager

The `TransactionManager` provides a generic, type-safe way to execute database operations within a transaction. It ensures that all repository operations within a callback use the same transaction-bound query executor.

## Core Concepts

### TxQuerier Interface

Any sqlc-generated `Queries` struct must implement `TxQuerier` to be compatible with the transaction manager:

```go
type TxQuerier[Q any] interface {
    WithTx(tx *sql.Tx) Q
}
```

sqlc generates this method automatically when configured with `emit_methods_with_db_argument: false`.

### TransactionManager Interface

```go
type TransactionManager[Q TxQuerier[Q], F any] interface {
    WithTx(ctx context.Context, fn func(ctx context.Context, f F) *apierror.APIError) *apierror.APIError
    WithTxSavepoint(ctx context.Context, fn func(ctx context.Context, f F, sp SavepointRunner) *apierror.APIError) *apierror.APIError
}
```

- `Q` is the sqlc Queries type
- `F` is the factory type (typically `RepoFactory`) that gets created with transaction-bound queries

### Deadlock Retry

A transaction the database rolls back as a deadlock victim is re-run (up to 3 attempts, with a
jittered millisecond backoff). The callback therefore runs more than once, and only its database
writes are undone by the rollback — so a callback may write to the database and nothing else.

`make tx-audit` enforces this across the codebase. It reports four escaping effects inside a
`WithTx` / `withTx` / `WithTxSavepoint` closure:

- appending to a variable declared outside the callback
- calling a client that leaves the database (Stripe, S3, RabbitMQ, another service's gRPC client)
- starting a goroutine
- sending on a channel

This is why domain events go to the outbox rather than being published inline.

## Setup

### 1. Define a RepoFactory

Create a factory that produces repositories from a queries instance:

```go
type repoFactoryImpl struct {
    queries *sqlc.Queries
}

func NewRepoFactory(queries *sqlc.Queries) domain.RepoFactory {
    return &repoFactoryImpl{queries: queries}
}

func (r *repoFactoryImpl) NewUserRepo() domain.UserRepo {
    return NewUserRepo(r.queries)
}

func (r *repoFactoryImpl) NewOrderRepo() domain.OrderRepo {
    return NewOrderRepo(r.queries)
}
```

### 2. Create the TransactionManager

```go
type TransactionManager = db.TransactionManager[*sqlc.Queries, domain.RepoFactory]

func NewTransactionManager(sqlDB *sql.DB, queries *sqlc.Queries) TransactionManager {
    return db.NewTransactionManager(sqlDB, queries, repository.NewRepoFactory)
}
```

### 3. Inject into Your Service

```go
type serviceSvcImpl struct {
    repos           domain.RepoFactory
    mediatorFactory domain.MediatorFactory
    txManager       TransactionManager
}
```

## Usage Pattern

### Service Layer

The service layer wraps operations in transactions using a `withTx` helper:

```go
func (s *serviceSvcImpl) withTx(ctx context.Context, fn func(context.Context, *serviceSvcImpl) *apierror.APIError) *apierror.APIError {
    return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
        // Create a NEW service instance with transaction-bound repos
        txSvc := &serviceSvcImpl{
            repos:           f,  // Transaction-bound factory
            mediatorFactory: s.mediatorFactory,
            txManager:       s.txManager,
        }
        return fn(txCtx, txSvc)
    })
}
```

Then use it in service methods:

```go
func (s *serviceSvcImpl) CreateOrder(ctx context.Context, input CreateOrderInput) (*Order, *apierror.APIError) {
    var result *Order

    apiErr := s.withTx(ctx, func(txCtx context.Context, svc *serviceSvcImpl) *apierror.APIError {
        // All operations here use the same transaction
        order, err := svc.mediators().Order.Create(txCtx, input)
        if err != nil {
            return err
        }

        err = svc.mediators().Inventory.Reserve(txCtx, order.Items)
        if err != nil {
            return err // Transaction will rollback
        }

        result = order
        return nil
    })

    return result, apiErr
}
```

### Mediator Layer

Mediators receive their `RepoFactory` at construction time. When built within a transaction context, they automatically use transaction-bound repositories:

```go
type orderMedImpl struct {
    repos domain.RepoFactory  // Injected at build time
}

func (m *orderMedImpl) Create(ctx context.Context, input CreateOrderInput) (*Order, *apierror.APIError) {
    orderRepo := m.repos.NewOrderRepo()      // Uses tx-bound queries
    itemRepo := m.repos.NewOrderItemRepo()   // Same transaction

    order, err := orderRepo.Create(ctx, input)
    if err != nil {
        return nil, err
    }

    for _, item := range input.Items {
        _, err := itemRepo.Create(ctx, order.ID, item)
        if err != nil {
            return nil, err
        }
    }

    return order, nil
}
```

### Mediator Factory

The mediator factory builds mediators with a specific `RepoFactory`:

```go
func (f *mediatorFactoryImpl) Build(repoFactory domain.RepoFactory) domain.Mediators {
    return domain.Mediators{
        Order:     NewOrderMed(OrderMedConfig{Repos: repoFactory}),
        Inventory: NewInventoryMed(InventoryMedConfig{Repos: repoFactory}),
    }
}
```

The service calls `mediators()` which builds with the current repos:

```go
func (s *serviceSvcImpl) mediators() domain.Mediators {
    return s.mediatorFactory.Build(s.repos)
}
```

## Transaction Flow

```
Service.CreateOrder()
    │
    ▼
withTx() starts transaction
    │
    ├── Creates tx-bound RepoFactory
    │
    ├── Creates new service instance with tx-bound repos
    │
    ▼
txSvc.mediators().Order.Create()
    │
    ├── mediators() calls factory.Build(txSvc.repos)
    │   └── repos is tx-bound
    │
    ├── Mediator uses m.repos.NewOrderRepo()
    │   └── Returns repo with tx-bound queries
    │
    ▼
All repo operations use same transaction
    │
    ▼
withTx() commits or rolls back
```

## Rules

1. **Always create repos fresh**: Call `m.repos.NewXxxRepo()` when you need a repo. Don't cache repo instances.

2. **Pass the txCtx**: Always use the context provided by the transaction callback, not the outer context.

3. **Return errors to trigger rollback**: The transaction commits only if the callback returns `nil`. Any error causes a rollback.

4. **Don't nest transactions**: The transaction manager doesn't support nested transactions. When you need one unit of work to fail without discarding the rest, use `WithTxSavepoint` instead of a second transaction.

5. **Keep transactions short**: Don't do external API calls or long-running operations inside a transaction — beyond holding locks, they are not undone by a rollback and would run twice on a deadlock retry. `make tx-audit` fails the build on them.

## Non-Transactional Operations

For read-only or single-write operations that don't need transactions, use the service's default repos directly:

```go
func (s *serviceSvcImpl) GetOrder(ctx context.Context, id string) (*Order, *apierror.APIError) {
    // No transaction needed for simple reads
    return s.mediators().Order.GetByID(ctx, id)
}
```

## Partial-Success Batches

`WithTxSavepoint` is `WithTx` plus a `SavepointRunner` over the same transaction. Each `Run`
brackets a unit of work in a `SAVEPOINT`: it releases the savepoint on success and rolls back to
it on error, undoing only that unit's writes while the surrounding transaction stays open. Use it
when one item in a batch may fail without discarding the rest — everything that did succeed still
commits together at the end.

```go
apiErr := s.txManager.WithTxSavepoint(ctx, func(txCtx context.Context, f domain.RepoFactory, sp db.SavepointRunner) *apierror.APIError {
    for _, item := range items {
        if err := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
            return importItem(spCtx, f, item)
        }); err != nil {
            failures = append(failures, err) // recorded, not fatal
        }
    }
    return nil
})
```

## Testing

Mock the `RepoFactory` interface to test mediators in isolation. Mocks are generated with
[mockgen](https://github.com/uber-go/mock) by `make mocks [service]` and live under
`internal/domain/mock/`:

```go
func TestOrderMed_Create(t *testing.T) {
    ctrl := gomock.NewController(t)

    orderRepo := repositorymock.NewMockOrderRepo(ctrl)
    orderRepo.EXPECT().
        Create(gomock.Any(), gomock.Any()).
        Return(&domain.Order{ID: "so_j8cz0b79pwdb"}, nil)

    repos := factorymock.NewMockRepoFactory(ctrl)
    repos.EXPECT().NewOrderRepo().Return(orderRepo).AnyTimes()

    med := NewOrderMed(OrderMedConfig{Repos: repos})
    // ... test
}
```
