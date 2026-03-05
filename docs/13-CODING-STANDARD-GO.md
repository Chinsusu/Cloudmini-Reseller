# Go Coding Standard — ProxyVPS Platform

**Document ID**: PVP-DOC-013  
**Version**: 1.0.0  
**Applies To**: All Go services  

---

## 1. Project Structure (Per Service)

```
services/{service-name}/
├── cmd/
│   └── server/
│       └── main.go          ← entry point only, wire dependencies
├── internal/
│   ├── config/
│   │   └── config.go        ← env loading, validation
│   ├── domain/
│   │   ├── entity.go        ← domain entities (pure structs)
│   │   ├── errors.go        ← domain-specific errors
│   │   └── repository.go    ← repository interfaces (no implementation)
│   ├── usecase/
│   │   └── {usecase}.go     ← business logic, depends on interfaces only
│   ├── handler/
│   │   └── http/
│   │       ├── handler.go   ← HTTP handlers, input parsing only
│   │       └── router.go    ← route registration
│   ├── repository/
│   │   └── postgres/
│   │       └── {repo}.go    ← DB implementation
│   ├── events/
│   │   ├── publisher.go     ← NATS publish
│   │   └── consumer.go      ← NATS consume
│   └── middleware/
│       └── {middleware}.go
├── pkg/                     ← utilities reusable within this service
├── Dockerfile
├── Makefile
└── go.mod
```

---

## 2. Naming Conventions

### Files
- Lowercase with underscores: `proxy_service.go`, `order_handler.go`
- Test files: `proxy_service_test.go`
- Interfaces in `domain/`: `repository.go`, `service.go`

### Packages
- Lowercase, single word preferred: `handler`, `usecase`, `domain`
- Avoid generic names: `utils`, `helpers`, `common` (use specific names)

### Variables & Functions
```go
// Good
userID      uuid.UUID
orderNumber string
maxRetries  int

// Bad
uid         uuid.UUID  // too short
UserID      uuid.UUID  // exported but shouldn't be local var
```

### Interfaces
```go
// Interface names: I prefix + noun
type IUserRepository interface { ... }
type IProxyProvider interface { ... }
type IBillingService interface { ... }
```

### Errors
```go
// Domain errors: Err prefix
var (
    ErrUserNotFound       = errors.New("user not found")
    ErrInsufficientFunds  = errors.New("insufficient wallet balance")
    ErrNodeUnavailable    = errors.New("no available proxmox node")
)
```

### Constants
```go
const (
    OrderStatusPending    = "pending"
    OrderStatusActive     = "active"
    OrderStatusExpired    = "expired"
    
    RoleUser      = "user"
    RoleReseller  = "reseller"
    RoleAdmin     = "admin"
)
```

---

## 3. Error Handling

### Rule: Never discard errors
```go
// Bad
user, _ := repo.GetUser(ctx, id)

// Good
user, err := repo.GetUser(ctx, id)
if err != nil {
    return fmt.Errorf("GetUser %s: %w", id, err)
}
```

### Wrapping errors with context
```go
// Always wrap with operation context
func (u *OrderUsecase) CreateOrder(ctx context.Context, req CreateOrderReq) (*Order, error) {
    user, err := u.userRepo.GetByID(ctx, req.UserID)
    if err != nil {
        return nil, fmt.Errorf("OrderUsecase.CreateOrder: get user: %w", err)
    }
    // ...
}
```

### HTTP error response
```go
// Use consistent error response
type APIError struct {
    Code      string `json:"code"`
    Message   string `json:"message"`
    RequestID string `json:"request_id"`
    Details   any    `json:"details,omitempty"`
}

// In handler
func respondError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
    requestID := r.Header.Get("X-Request-ID")
    render.Status(r, statusCode)
    render.JSON(w, r, APIError{
        Code:      code,
        Message:   message,
        RequestID: requestID,
    })
}
```

---

## 4. Context Usage

```go
// Always pass ctx as first argument
func (r *OrderRepository) Create(ctx context.Context, order *Order) error { ... }

// Always check ctx cancellation in long operations
for _, item := range items {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    // process item
}

// Always set timeouts for external calls
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
resp, err := httpClient.Do(req.WithContext(ctx))
```

---

## 5. Dependency Injection

Constructor injection. Không dùng global state hoặc init().

```go
// Good: explicit dependencies
type OrderUsecase struct {
    orderRepo   domain.IOrderRepository
    billing     domain.IBillingService
    provider    domain.IProxyProvider
    eventPub    domain.IEventPublisher
    logger      *slog.Logger
}

func NewOrderUsecase(
    orderRepo domain.IOrderRepository,
    billing domain.IBillingService,
    provider domain.IProxyProvider,
    eventPub domain.IEventPublisher,
    logger *slog.Logger,
) *OrderUsecase {
    return &OrderUsecase{
        orderRepo: orderRepo,
        billing:   billing,
        provider:  provider,
        eventPub:  eventPub,
        logger:    logger,
    }
}

// Bad: global state
var globalDB *sql.DB
```

---

## 6. Logging Standard

Dùng `log/slog` (stdlib, Go 1.21+) với structured fields.

```go
// Initialize logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

// Log with context fields — always include request_id, user_id
logger.InfoContext(ctx, "proxy order created",
    slog.String("request_id", requestID),
    slog.String("user_id", userID.String()),
    slog.String("order_id", order.ID.String()),
    slog.String("provider", providerName),
    slog.Float64("amount", amount),
    slog.Duration("duration", time.Since(start)),
)

// Error with stack context
logger.ErrorContext(ctx, "provider API call failed",
    slog.String("provider", providerName),
    slog.String("error", err.Error()),
    slog.Int("attempt", attempt),
)
```

---

## 7. Database Access

```go
// Use sqlx or pgx, never raw database/sql for complex queries
// Always use named parameters for clarity

// Good
query := `
    SELECT id, user_id, status, total_amount
    FROM proxy.orders
    WHERE user_id = $1 AND status = $2
    ORDER BY created_at DESC
    LIMIT $3 OFFSET $4
`
rows, err := db.QueryContext(ctx, query, userID, status, limit, offset)

// Always use transactions for multi-step DB operations
tx, err := db.BeginTx(ctx, nil)
if err != nil { return err }
defer tx.Rollback() // safe to call after Commit

// ... operations ...
return tx.Commit()
```

---

## 8. Testing Standards

```go
// Unit test: test business logic in isolation
func TestOrderUsecase_CreateOrder_InsufficientFunds(t *testing.T) {
    // Arrange
    mockBilling := &MockBillingService{}
    mockBilling.On("DeductWallet", mock.Anything, mock.Anything).
        Return(domain.ErrInsufficientFunds)
    
    uc := NewOrderUsecase(mockRepo, mockBilling, ...)
    
    // Act
    _, err := uc.CreateOrder(context.Background(), req)
    
    // Assert
    assert.ErrorIs(t, err, domain.ErrInsufficientFunds)
    mockBilling.AssertExpectations(t)
}

// Test naming: TestTypeName_MethodName_Scenario
// Test structure: Arrange → Act → Assert
```

---

## 9. Code Review Checklist

Before submitting PR, verify:

- [ ] All errors handled (no `_` discards for errors)
- [ ] Context passed through all function calls
- [ ] External calls have timeout
- [ ] Sensitive data (passwords, keys) NOT logged
- [ ] New DB queries have indexes reviewed
- [ ] Unit tests written for all new usecases
- [ ] No hardcoded configuration values (use config package)
- [ ] API changes reflected in API Design Standard
- [ ] Events published where state changes occur
- [ ] No fmt.Println (use structured logger)
- [ ] Dockerfile updated if new env vars added

---

## 10. Forbidden Patterns

```go
// ❌ NEVER: panic in business logic
func GetUser(id string) *User {
    user, err := repo.Get(id)
    if err != nil {
        panic(err) // FORBIDDEN
    }
    return user
}

// ❌ NEVER: global mutable state
var currentUser *User // FORBIDDEN

// ❌ NEVER: init() with side effects
func init() {
    db.Connect() // FORBIDDEN
}

// ❌ NEVER: log sensitive data
logger.Info("user login", "password", password) // FORBIDDEN
```
