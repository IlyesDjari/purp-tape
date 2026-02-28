// Package interfaces defines common interfaces for the PurpTape application [LOW: Documentation]
//
// Benefits of Using Interfaces:
// - Makes code more testable (easy to mock)
// - Defines clear contracts between components
// - Enables dependency injection
// - Makes swapping implementations easy
//
// Core Interfaces:
//
// 1. Handler
//   Any HTTP handler implementing ServeHTTP
//
// 2. Service
//   Business logic services with Health() method
//
// 3. Repository
//   Data access objects with Close() method
//
// 4. StorageProvider
//   Abstraction for file storage (R2, S3, GCS, etc.)
//   Enables swapping cloud providers easily
//
// 5. CacheProvider
//   Abstraction for caching (Redis, Memcached, in-memory, etc.)
//
// Usage Patterns:
//
// Dependency Injection:
//   type UserService struct {
//     db       interfaces.Repository
//     cache    interfaces.CacheProvider
//     storage  interfaces.StorageProvider
//   }
//
//   func NewUserService(
//     db interfaces.Repository,
//     cache interfaces.CacheProvider,
//   ) *UserService {
//     return &UserService{db: db, cache: cache}
//   }
//
// Testing with Mocks:
//   type MockCache struct{}
//   func (m *MockCache) Get(ctx context.Context, key string) (interface{}, error) {
//     return nil, fmt.Errorf("cache miss")
//   }
//   // ... implement other methods
//
//   func TestService(t *testing.T) {
//     mockCache := &MockCache{}
//     service := NewService(db, mockCache)
//     // Test service behavior with mock cache
//   }
//
// Adding New Interfaces:
// 1. Identify the abstraction boundary
// 2. Define minimal set of methods needed
// 3. Document expected behavior
// 4. Implement interface in concrete types
// 5. Use interface in dependent code for flexibility
//
// Do write small, focused interfaces (1-5 methods)
// Don't write fat interfaces (too many methods)
//
// Interface Segregation Principle:
// Keep interfaces small and specific.
// Clients should depend on interfaces specific to their needs,
// not on large general-purpose interfaces.
//
// Example - Good:
//   type Reader interface {
//     Read(ctx context.Context, key string) ([]byte, error)
//   }
//
// Example - Bad:
//   type Store interface {
//     Read(ctx context.Context, key string) ([]byte, error)
//     Write(ctx context.Context, key string, value []byte) error
//     Delete(ctx context.Context, key string) error
//     List(ctx context.Context, prefix string) ([]string, error)
//     Watch(ctx context.Context, key string) (<-chan Event, error)
//     // ... many more methods
//   }
//
// When in doubt, make the interface smaller.
package interfaces
