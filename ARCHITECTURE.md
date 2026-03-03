# Marketplace Backend - Architecture Documentation

## Modular Domain-Driven Design

This project follows a **modular, domain-driven architecture** where each business entity is self-contained with its own interfaces for both data access and business logic.

## Project Structure

```
marketplace-backend/
├── api/openapi/              # OpenAPI specification
├── cmd/server/               # Application entry point
├── internal/
│   ├── users/               # User domain module
│   │   ├── repository.go         # Repository interface
│   │   ├── repository_impl.go    # PostgreSQL implementation
│   │   ├── service.go            # Service interface
│   │   └── service_impl.go       # Business logic implementation
│   ├── products/            # Product domain module (to be created)
│   ├── orders/              # Order domain module (to be created)
│   ├── promos/              # Promo code domain module (to be created)
│   ├── domain/              # Shared domain models
│   ├── errors/              # Error definitions
│   ├── middleware/          # HTTP middleware
│   ├── generated/           # OpenAPI generated code
│   └── config/              # Configuration
├── migrations/              # Database migrations
└── docker-compose.yml       # Docker services
```

## Architecture Principles

### 1. **Domain Modules**

Each business entity (users, products, orders, promos) is organized in its own package with:

```
internal/users/
├── repository.go       # Data access interface
├── repository_impl.go  # Database implementation
├── service.go         # Business logic interface
└── service_impl.go    # Business logic implementation
```

### 2. **Interface-Based Design**

**Repository Interface** (`repository.go`):
```go
type Repository interface {
    Create(user *domain.User) error
    GetByID(id uuid.UUID) (*domain.User, error)
    GetByEmail(email string) (*domain.User, error)
}
```

**Service Interface** (`service.go`):
```go
type Service interface {
    Register(email, password string, role domain.UserRole) (*domain.User, error)
    Login(email, password string) (accessToken, refreshToken string, err error)
    RefreshToken(refreshToken string) (string, error)
    ValidateAccessToken(tokenString string) (*Claims, error)
}
```

### 3. **Dependency Injection**

Services depend on repository **interfaces**, not implementations:

```go
type serviceImpl struct {
    repo Repository  // Interface, not concrete type
    // ... other fields
}

func NewService(repo Repository, ...) Service {
    return &serviceImpl{
        repo: repo,
        // ...
    }
}
```

### 4. **Benefits**

✅ **Testability**: Easy to mock interfaces for unit tests
✅ **Maintainability**: Changes to one module don't affect others
✅ **Clarity**: Clear separation between data access and business logic
✅ **Flexibility**: Can swap implementations (e.g., different databases)
✅ **Modularity**: Each domain is self-contained

## Layer Responsibilities

### Repository Layer (Data Access)

**Responsibility**: Database operations only
- CRUD operations
- Queries and filters
- Transaction management
- No business logic

**Example** (`users/repository_impl.go`):
```go
func (r *repositoryImpl) GetByEmail(email string) (*domain.User, error) {
    var user domain.User
    query := `SELECT * FROM users WHERE email = $1`
    err := r.db.Get(&user, query, email)
    // ... error handling
    return &user, nil
}
```

### Service Layer (Business Logic)

**Responsibility**: Business rules and orchestration
- Validation
- Business logic
- Orchestrating multiple repositories
- Transaction coordination
- Uses repository through interface

**Example** (`users/service_impl.go`):
```go
func (s *serviceImpl) Login(email, password string) (string, string, error) {
    user, err := s.repo.GetByEmail(email)  // Uses interface
    if err != nil {
        return "", "", fmt.Errorf("invalid credentials")
    }
    
    // Business logic: password verification
    if err := bcrypt.CompareHashAndPassword(/*...*/); err != nil {
        return "", "", fmt.Errorf("invalid credentials")
    }
    
    // Business logic: token generation
    accessToken, err := s.generateToken(user, s.accessExpiry)
    // ...
}
```

## Dependency Flow

```
HTTP Request
    ↓
Handler (validates, extracts data)
    ↓
Service Interface (business logic)
    ↓
Service Implementation
    ↓
Repository Interface (data access)
    ↓
Repository Implementation
    ↓
PostgreSQL Database
```

## Example: Users Module

### Files Structure

```
internal/users/
├── repository.go          # Interface: Create, GetByID, GetByEmail
├── repository_impl.go     # Implementation using sqlx
├── service.go            # Interface: Register, Login, RefreshToken, ValidateAccessToken
└── service_impl.go       # Implementation with JWT, bcrypt
```

### Wiring in Main

```go
// Create repository
userRepo := users.NewRepository(db)

// Create service with repository interface
userService := users.NewService(
    userRepo,           // Repository interface
    cfg.JWT.Secret,
    cfg.JWT.AccessExpiry,
    cfg.JWT.RefreshExpiry,
)

// Use service in middleware
authMiddleware := middleware.AuthMiddleware(userService)
```

### Testing Example

```go
// Mock repository for testing
type mockRepo struct{}

func (m *mockRepo) GetByEmail(email string) (*domain.User, error) {
    return &domain.User{Email: email}, nil
}

// Test service with mock
func TestLogin(t *testing.T) {
    mockRepo := &mockRepo{}
    service := users.NewService(mockRepo, "secret", time.Hour, time.Hour)
    
    token, _, err := service.Login("test@example.com", "password")
    // ... assertions
}
```

## Migration from Old Structure

### Old Structure (Flat)
```
internal/
├── repository/
│   ├── user.go
│   ├── product.go
│   └── order.go
└── service/
    ├── auth.go
    ├── product.go
    └── order.go
```

### New Structure (Modular)
```
internal/
├── users/
│   ├── repository.go
│   ├── repository_impl.go
│   ├── service.go
│   └── service_impl.go
├── products/
│   ├── repository.go
│   ├── repository_impl.go
│   ├── service.go
│   └── service_impl.go
└── orders/
    ├── repository.go
    ├── repository_impl.go
    ├── service.go
    └── service_impl.go
```

## Next Steps

To complete the modular architecture:

1. **Create `internal/products/` module**
   - `repository.go` - Interface for product data access
   - `repository_impl.go` - PostgreSQL implementation
   - `service.go` - Interface for product business logic
   - `service_impl.go` - CRUD operations, stock management

2. **Create `internal/orders/` module**
   - `repository.go` - Interface for order data access
   - `repository_impl.go` - PostgreSQL implementation with transactions
   - `service.go` - Interface for order business logic
   - `service_impl.go` - Complex order logic, state machine, stock reservation

3. **Create `internal/promos/` module**
   - `repository.go` - Interface for promo code data access
   - `repository_impl.go` - PostgreSQL implementation
   - `service.go` - Interface for promo business logic
   - `service_impl.go` - Validation, discount calculation

4. **Create handlers** that use service interfaces

5. **Wire everything in `cmd/server/main.go`**

## Key Advantages

1. **Clear Boundaries**: Each module is independent
2. **Easy Testing**: Mock interfaces, not implementations
3. **Flexible**: Swap implementations without changing business logic
4. **Maintainable**: Changes are localized to modules
5. **Scalable**: Easy to add new modules
6. **Professional**: Follows industry best practices (DDD, Clean Architecture)

## Design Patterns Used

- **Repository Pattern**: Abstracts data access
- **Service Pattern**: Encapsulates business logic
- **Dependency Injection**: Services depend on interfaces
- **Interface Segregation**: Small, focused interfaces
- **Single Responsibility**: Each file has one purpose