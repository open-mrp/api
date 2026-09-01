# Architecture patterns

This document describes the layered architecture used across all backend services. Each service follows the same structure: **Services** orchestrate transactions and idempotency, **Mediators** encapsulate discrete business logic, and **Repositories** handle data access.

## Layers at a glance

```
gRPC Handler
  └─ Service        (transaction boundary, idempotency, orchestration)
       ├─ Mediator   (reusable business logic unit)
       │    └─ Repository  (data access via sqlc)
       └─ Repository       (direct reads when no business logic needed)
```

---

## Services

Services orchestrate business logic. Services are responsible for opening and managing database transactions to ensure each operation is atomic. Additionally, services handle all idempotency concerns, including setting recovery points. Services may call mediators and repositories.

### Transaction management

Services use a `withTx` helper to wrap business logic in a database transaction. Inside the transaction, a **new service instance** is created with a transaction-scoped repository factory, ensuring all mediators and repositories share the same `sql.Tx`:

```go
func (s *userSvcImpl) withTx(ctx context.Context, fn func(context.Context, *userSvcImpl) *apierror.APIError) *apierror.APIError {
    return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
        txSvc := &userSvcImpl{
            repos:           f,              // transaction-scoped factory
            mediatorFactory: s.mediatorFactory,
            txManager:       s.txManager,
        }
        return fn(txCtx, txSvc)
    })
}
```

### Idempotency and recovery points

Mutating service methods follow a recovery-point pattern:

1. Upsert an idempotency key (scoped to actor + service + handler + client key).
2. Switch on the recovery point to decide what to do:
   - `Finished` — return the cached response.
   - `Started` — execute business logic, then cache the result.

```go
func (s *userSvcImpl) Login(ctx context.Context, identifier, password string) (*domain.LoginResult, *apierror.APIError) {
    meds := s.mediators()

    idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
    if apiErr != nil {
        return nil, apiErr
    }

    switch idempotencyKey.RecoveryPoint {
    case domain.RecoveryPointFinished:
        cached, err := idempotency.UnmarshalCachedResponse[domain.LoginResult](
            ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody,
        )
        if err != nil {
            return nil, apierror.NewInternalError(err, "Issue unmarshalling cached response.")
        }
        return cached.Data, cached.Error

    case domain.RecoveryPointStarted:
        // non-transactional work
        user, apiErr := meds.Password.Validate(ctx, identifier, password)
        if apiErr != nil {
            return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
        }

        accessToken, apiErr := meds.User.GenAuthAccessToken(ctx, user.ID)
        if apiErr != nil {
            return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
        }

        // transactional work
        var refreshToken string
        apiErr = s.withTx(ctx, func(txCtx context.Context, svc *userSvcImpl) *apierror.APIError {
            txMeds := svc.mediators()
            rt, err := txMeds.RefreshToken.Create(txCtx, user.ID, nil)
            if err != nil {
                return err
            }
            refreshToken = rt.Token
            return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &domain.LoginResult{
                User: user, AccessToken: accessToken, RefreshToken: refreshToken,
            })
        })
        if apiErr != nil {
            return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
        }

        return &domain.LoginResult{User: user, AccessToken: accessToken, RefreshToken: refreshToken}, nil

    default:
        return nil, apierror.NewInvariantViolationError("Unexpected recovery point: " + idempotencyKey.RecoveryPoint.String())
    }
}
```

Key rules:
- Only **non-transient** errors are cached (transient errors allow the client to retry).
- Success responses are cached **inside** the transaction so caching and business state are committed atomically.

For RPCs backing **HTTP `POST` and `PATCH`** from the API gateway, gRPC handlers must also call `contracts.WithIdempotencyTracking` (see [`shared/contracts/idempotency_interceptor.go`](../../shared/contracts/idempotency_interceptor.go)) so client idempotency keys align with this service-layer pattern; repository-wide guidance lives in the root `AGENTS.md`.

### Construction

Services use a config struct for dependency injection:

```go
type UserSvcConfig struct {
    Repos                 domain.RepoFactory
    MediatorFactory       domain.MediatorFactory
    NotificationPublisher domain.NotificationPublisher
    TxManager             TransactionManager
}

func NewUserSvc(config UserSvcConfig) domain.UserSvc {
    return &userSvcImpl{
        repos:                 config.Repos,
        mediatorFactory:       config.MediatorFactory,
        notificationPublisher: config.NotificationPublisher,
        txManager:             config.TxManager,
    }
}
```

A convenience `Default*Config` function wires production dependencies:

```go
func DefaultUserSvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) UserSvcConfig {
    repoFactory := repository.NewRepoFactory(queries)
    notificationPublisher := event.NewOutboxNotificationPublisher()

    mediatorFactory := mediator.NewMediatorFactory(mediator.MediatorFactoryConfig{
        JWTSecret:             jwtSecret,
        APIKeyPepper:          pepper,
        NotificationPublisher: notificationPublisher,
        FrontendURL:           frontendURL,
        CoreClient:            coreClient,
    })

    return UserSvcConfig{
        Repos:           repoFactory,
        MediatorFactory: mediatorFactory,
        NotificationPublisher: notificationPublisher,
    }
}
```

### When to call mediators vs. repositories directly

- **Mediators** — when the operation involves business logic, side effects, or multiple steps.
- **Repositories directly** — for simple reads that need no business logic:

```go
// Simple read: call repository directly
func (s *accountSvcImpl) GetUserAccountAccess(ctx context.Context, userID, accountID string) (*domain.AccountUserAccess, *apierror.APIError) {
    accountUser, apiErr := s.accountUserRepo.FindByAccountAndUserID(ctx, userID, accountID)
    if apiErr != nil {
        return nil, apiErr
    }
    // ... assemble response from repo data ...
}
```

---

## Mediators

Mediators are a single unit of discrete business logic. Mediators can be strung together to define a larger piece of business logic. Mediators **never** open a database transaction or manage it. They always respect the context and repository factory they are provided (which in turn respects the database transaction).

### Structure

Each mediator is a struct that receives a repo factory (and optionally other mediators or utilities) at construction time:

```go
type refreshTokenMedImpl struct {
    repos    domain.RepoFactory
    jwtUtils domain.JWTUtils
}
```

### Transaction scoping

Mediators never open transactions. When a service needs transactional mediator calls, it rebuilds the mediators with a transaction-scoped factory:

```go
apiErr = s.withTx(ctx, func(txCtx context.Context, svc *userSvcImpl) *apierror.APIError {
    txMeds := svc.mediators()  // mediators now backed by transactional repos
    _, err := txMeds.RefreshToken.Create(txCtx, user.ID, nil)
    return err
})
```

The mediator itself has no awareness of whether it is running inside a transaction — it simply uses the repos it was given.

### Composition

Mediators may depend on other mediators. These dependencies are wired at factory build time:

```go
func (f *mediatorFactoryImpl) Build(repoFactory domain.RepoFactory) domain.Mediators {
    refreshTokenMed := NewRefreshTokenMed(RefreshTokenMedConfig{
        Repos:    repoFactory,
        JWTUtils: token.NewJWTUtils(&token.JWTConfig{Secret: f.jwtSecret}),
    })

    apiKeyMed := NewAPIKeyMed(APIKeyMedConfig{
        Repos:      repoFactory,
        Pepper:     f.apiKeyPepper,
        CoreClient: f.coreClient,
    })

    // UserMed depends on RefreshTokenMed and APIKeyMed
    userMed := NewUserMed(UserMedConfig{
        Repos:           repoFactory,
        JWTSecret:       f.jwtSecret,
        RefreshTokenMed: refreshTokenMed,
        APIKeyMed:       apiKeyMed,
        CoreClient:      f.coreClient,
    })

    return domain.Mediators{
        User:         userMed,
        APIKey:       apiKeyMed,
        RefreshToken: refreshTokenMed,
        Idempotency:  NewIdempotencyMed(IdempotencyMedConfig{Repos: repoFactory}),
        // ...
    }
}
```

### Docstring convention

Every mediator method should have:
1. A human-readable summary line.
2. Numbered steps describing the discrete operations.

```go
// FindAndValidate takes in a raw API key string and validates it. If a valid API key
// is found, we retrieve it.
//
//  1. Parses the API key into its components.
//  2. Finds the API key in the database.
//  3. Verify the secret against the secret hash in the database.
//  4. Make sure the key has not been revoked or expired.
//  5. Returns the API key.
func (s *apiKeyMedImpl) FindAndValidate(ctx context.Context, apiKey string) (*apikey.APIKey, *apierror.APIError) {
```

The same docstring should appear on both the domain interface and the implementation.

---

## Repositories

Repositories handle all data access. They translate between domain models and sqlc-generated types.

### Implementation

Each repository wraps `*sqlc.Queries` and implements a domain interface:

```go
type userRepoImpl struct {
    db *sqlc.Queries
}

func NewUserRepo(db *sqlc.Queries) domain.UserRepo {
    return &userRepoImpl{db: db}
}

func (r *userRepoImpl) Find(ctx context.Context, identifier string) (*types.User, *apierror.APIError) {
    ctx, span := userRepoTracer.Start(ctx, "repository.user.find")
    defer span.End()

    userModel, err := r.db.FindUserByIdentifier(ctx, sqlc.FindUserByIdentifierParams{
        ID:       identifier,
        Username: db.NullString(identifier),
        Email:    db.NullString(identifier),
    })
    if apiErr := db.MapSQLError(err); apiErr != nil {
        if apierror.IsNotFound(apiErr) {
            return nil, apiErr
        }
        return nil, tracing.Trace(span, apiErr)
    }

    return &types.User{
        ID:    userModel.ID,
        Email: db.StringFromNullString(userModel.Email),
        Name:  db.StringFromNullString(userModel.Name),
        // ...
    }, nil
}
```

Conventions:
- SQL errors are mapped via `db.MapSQLError()` to produce consistent `*apierror.APIError` values.
- Null conversions use `db.NullString()` / `db.StringFromNullString()` / `db.TimeFromNullTime()`.
- Each repository gets its own tracer: `tracing.GetTracer("service-name.repo_name")`.

### Repository factory

Each service defines a `RepoFactory` interface. The concrete implementation wraps `*sqlc.Queries`:

```go
// Domain interface
type RepoFactory interface {
    NewUserRepo() UserRepo
    NewRefreshTokenRepo() RefreshTokenRepo
    NewAPIKeyRepo() APIKeyRepo
    NewIdempotencyKeyRepo() IdempotencyKeyRepo
    NewOutboxRepo() messaging.OutboxRepo
}

// Infrastructure implementation
type repoFactoryImpl struct {
    queries *sqlc.Queries
}

func NewRepoFactory(queries *sqlc.Queries) domain.RepoFactory {
    return &repoFactoryImpl{queries: queries}
}

func (r *repoFactoryImpl) NewUserRepo() domain.UserRepo {
    return NewUserRepo(r.queries)
}
// ... one method per repository ...
```

When `queries` is backed by a `sql.Tx`, every repository created by the factory shares that transaction.

---

## Transaction manager

The shared `db.TransactionManager` is a generic type that:

1. Begins a transaction.
2. Creates a transaction-scoped `*sqlc.Queries` via `WithTx(tx)`.
3. Passes a new factory (built from the transactional queries) into the callback.
4. Commits on success, rolls back on error.

```go
type TransactionManager[Q TxQuerier[Q], F any] interface {
    WithTx(ctx context.Context, fn func(ctx context.Context, f F) *apierror.APIError) *apierror.APIError
}

func (m *transactionManagerImpl[Q, F]) WithTx(
    ctx context.Context,
    fn func(ctx context.Context, f F) *apierror.APIError,
) *apierror.APIError {
    tx, err := m.db.BeginTx(ctx, nil)
    if err != nil {
        return apierror.NewInternalError(err, "failed to begin transaction")
    }
    defer tx.Rollback()

    qTx := m.queries.WithTx(tx)     // transaction-scoped query executor
    factory := m.factoryCreate(qTx)  // factory bound to this transaction

    if apiErr := fn(ctx, factory); apiErr != nil {
        return apiErr
    }

    if err := tx.Commit(); err != nil {
        return apierror.NewInternalError(err, "failed to commit transaction")
    }
    return nil
}
```

Each service creates a type alias with concrete types:

```go
type TransactionManager = db.TransactionManager[*sqlc.Queries, domain.RepoFactory]

func NewTransactionManager(sqlDB *sql.DB, queries *sqlc.Queries) TransactionManager {
    return db.NewTransactionManager(sqlDB, queries, repository.NewRepoFactory)
}
```

---

## Inventory ledger lock order

A transaction that writes any of `inventory_issue`, `inventory_receipt`, `inventory_allocation`, or a
`quantity`/`rate` row owned by one of them must, **before any other statement**:

1. resolve the complete set of `item_id`s it will touch, and
2. acquire `inventory_item_lock` for each of them in **ascending `item_id` order**.

Inside the critical section, the order in which it reaches those tables does not matter.

That last sentence is the whole reason the rule exists. The flows genuinely disagree about order and
cannot be reconciled: an allocator must hold the demand before it can choose which receipts to draw,
and a reversal must free the receipts before it can restore the demand they were covering. Neither is
wrong. The only fix is to stop them overlapping.

```go
// The item set is resolved on the pool, before the transaction opens.
itemIDs, apiErr := repo.StockedItemIDs(ctx, params)
if apiErr != nil {
    return apiErr
}

apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *svcImpl) *apierror.APIError {
    scope, apiErr := ledgerlock.Acquire(txCtx, txSvc.repos.NewInventoryMutationRepo(), itemIDs)
    if apiErr != nil {
        return apiErr
    }
    // ... the transaction's work, with scope threaded to every ledger writer ...
})
```

**Corollary A.** Never acquire the root while already holding a lock on a ledger row. Resolve the item
set before `WithTx`. Discovering it inside the transaction and locking afterwards is the inversion,
not the fix.

**Corollary B.** A late acquisition — one taken after the transaction already holds ledger row locks —
is itself an ordering inversion against any transaction that holds the root and is waiting for those
rows. `Scope.EnsureLocked` permits it rather than refusing, because refusing dead-letters a shop-floor
scan or 500s a ship, and the worst outcome is a 1213 the transaction manager retries. It is **not
safe**: it means a flow's item pre-read is wrong, it is logged at ERROR, and every occurrence is a
page.

**Corollary C.** The root is keyed on `item_id` alone, never `(account_id, item_id)`.
`FindReceiptsForAllocation` matches `owner_account_id = ? OR holder_account_id = ?`, so consigned stock
under one account is drawn down by another's demand, and an account-scoped key would hand two
contending flows two different rows over one contended receipt.

**Corollary D.** Rows in `inventory_item_lock` are never deleted and never read with `SELECT`.
`INSERT … ON DUPLICATE KEY UPDATE` must remain the only statement ever issued against that table —
that, and nothing about the backfill, is what makes gap locks structurally impossible there.
`TestLedgerLock_RejectedShapeIsUnsafe` measures what the alternative costs.

**Corollary E.** The root is a Go-side ordering device. It is not what keeps the ledger correct while a
second, unlocked writer exists — `dashboard/apps/api` writes these same tables from Prisma with no
locking read at all. Never remove a server-side guard on the grounds that the root covers it.

### Enforcement is the compiler, not a linter

Every ledger-writing repository method takes a `*ledgerlock.Scope`, so a transaction that never called
`Acquire` cannot compile a call to one. `tools/txaudit` deliberately does not walk transitively (see
its own header), and every ledger write here is reached indirectly from its `WithTx` closure — the
compiler sees that call graph; a name-based linter does not.

The residual is honest: `&ledgerlock.Scope{}` is constructible by anyone, and a forged one holds
nothing, so it degrades to a paged late acquisition rather than a silent bypass.

### Statement ordering inside the section

> No consistent read may precede the last locking statement of a ledger transaction.

InnoDB opens the REPEATABLE READ view on the first *consistent* read, and a locking read is not one. So
a transaction that takes its locks first has a view strictly newer than every conforming writer that
released them. Getting this wrong is silent: commit `7044443a` made the paged open-issue read
`FOR UPDATE` and claimed exactly this guarantee, but the caller resolved unit ratios next — a plain
read — so the view opened before any receipt lock was held, and a transaction that queued on one woke
up and computed what was left from a pre-lock snapshot.

Ordering covers every writer that takes the same locks. It does not cover the one that takes none, so
`ReadReceiptAllocationsForUpdate` and `ReadIssueCoverageForUpdate` read what a receipt has been drawn
*currently*, and those must name every joined table in their `FOR UPDATE OF` list — a locking read is
current only for the tables it names, and a satellite left out is read from the snapshot and silently
drops its row from the join.

## Domain layer

Each service defines its domain types in `internal/domain/`:

- **Models** — plain structs representing business entities:

```go
type RefreshToken struct {
    Token     string
    UserID    string
    ExpiresAt time.Time
    RevokedAt *time.Time
}
```

- **Repository interfaces** — contracts for data access:

```go
type RefreshTokenRepo interface {
    Find(ctx context.Context, token string) (*RefreshToken, *apierror.APIError)
    Create(ctx context.Context, userID string, token string, expiresInDays int) (*RefreshToken, *apierror.APIError)
    Revoke(ctx context.Context, token string) *apierror.APIError
    RevokeAll(ctx context.Context, userID string) *apierror.APIError
}
```

- **Mediator interfaces** — contracts for business logic units (with docstrings).
- **Service interfaces** — contracts for the service layer.
- **Factory interfaces** — `RepoFactory` and `MediatorFactory`.
- **Mocks** — generated via GoMock in `domain/mock/`.

---

## Error handling

All business-layer functions return `*apierror.APIError` instead of `error`. This type separates public-facing information from internal diagnostics:

```go
type APIError struct {
    Code            ErrorCode  // machine-readable code (e.g. "validation_failed")
    Type            ErrorType  // category (e.g. "invalid_request_error")
    PublicMessage   string     // sent to clients
    Param           string     // which field caused the error (optional)
    IsTransient     bool       // whether the client should retry
    InternalMessage string     // logs/traces only, never sent to clients
    Internal        error      // wrapped error for diagnostics
}
```

Errors are mapped to HTTP status codes (`GetHTTPStatusCode`) and gRPC codes (`ConvertAPIErrorToGRPC`). When crossing gRPC boundaries, the full `APIError` is JSON-encoded in the status message with a `__API_ERROR__:` prefix, allowing lossless reconstruction on the client side.

---

## Tracing

Every layer participates in OpenTelemetry tracing with a consistent naming convention:

```go
// Each file declares a package-level tracer
var userSvcTracer = tracing.GetTracer("auth-service.user_service")

// Each method starts and defers a span
func (s *userSvcImpl) Login(ctx context.Context, ...) (...) {
    ctx, span := userSvcTracer.Start(ctx, "service.user.login")
    defer span.End()
    // ...
}
```

Naming pattern: `"<layer>.<entity>.<operation>"` — e.g. `service.user.login`, `mediator.api_key.create`, `repository.refresh_token.find`.

Errors are recorded on spans via `tracing.Trace(span, apiErr)`.

---

## Conventions

1. Each service and mediator method should include a docstring that outlines the overall business logic being implemented.
2. Business logic should never be defined outside of the context of a mediator or service.
3. All mutating methods return `*apierror.APIError`; never bare `error`.
4. Repositories translate between domain models and sqlc types — no business logic.
5. Services own the transaction boundary; mediators and repositories never open transactions.
6. Mediators are rebuilt per transaction via the mediator factory to ensure they use the correct repository factory.
