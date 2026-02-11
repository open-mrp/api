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
}
```

- `Q` is the sqlc Queries type
- `F` is the factory type (typically `RepoFactory`) that gets created with transaction-bound queries

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

4. **Don't nest transactions**: The transaction manager doesn't support nested transactions. If you need nested behavior, restructure your code.

5. **Keep transactions short**: Don't do external API calls or long-running operations inside a transaction.

## Non-Transactional Operations

For read-only or single-write operations that don't need transactions, use the service's default repos directly:

```go
func (s *serviceSvcImpl) GetOrder(ctx context.Context, id string) (*Order, *apierror.APIError) {
    // No transaction needed for simple reads
    return s.mediators().Order.GetByID(ctx, id)
}
```

## Testing

Mock the `RepoFactory` interface to test mediators in isolation:

```go
func TestOrderMed_Create(t *testing.T) {
    mockRepoFactory := &mock.RepoFactoryMock{
        NewOrderRepoFunc: func() domain.OrderRepo {
            return &mock.OrderRepoMock{
                CreateFunc: func(ctx context.Context, input CreateOrderInput) (*Order, *apierror.APIError) {
                    return &Order{ID: "order_123"}, nil
                },
            },
        },
    }

    med := NewOrderMed(OrderMedConfig{Repos: mockRepoFactory})
    // ... test
}
```
