# PurpTape Backend Testing Guide

## Overview

This document outlines the testing strategy and available test suites for the PurpTape backend. The test suite covers authentication, validation, encryption, error handling, database operations, and middleware components.

## Running Tests

### All Tests
```bash
make test
```

### Tests with Coverage Report
```bash
make test-coverage
```

This generates `coverage.html` showing line coverage across all packages.

### Unit Tests Only
```bash
make test-unit
```

### Specific Package Tests
```bash
# Authentication tests
make test-auth

# Input validation tests
make test-validation

# Encryption tests
make test-encryption

# Error handling tests
make test-errors

# Middleware tests
make test-middleware
```

## Test Files and Coverage

### 1. Authentication Tests (`internal/auth/validator_test.go`)

**Coverage**: JWT token validation with Supabase integration

- ✅ Valid token parsing with signature verification
- ✅ Authorization header format validation
- ✅ Rejection of 'none' algorithm tokens (security)
- ✅ Expired token detection
- ✅ Invalid signature detection
- ✅ Token claims extraction

**Run with**:
```bash
make test-auth
```

### 2. Validation Tests (`internal/validation/inputs_test.go`)

**Coverage**: Input validation for all user-provided data

- ✅ Project name validation (length, whitespace)
- ✅ Track name validation
- ✅ Description validation (max length)
- ✅ Comment validation (min/max length)
- ✅ Password strength validation (uppercase, lowercase, digits)
- ✅ Email format validation
- ✅ String sanitization

**Run with**:
```bash
make test-validation
```

**Example test cases**:
- Valid names, whitespace handling, length boundaries
- Invalid passwords missing requirements
- Email format edge cases

### 3. Encryption Tests (`internal/encryption/aes_test.go`)

**Coverage**: AES-256-GCM encryption for sensitive data

- ✅ Valid key initialization
- ✅ Invalid key rejection
- ✅ Encrypt/decrypt round-trip verification
- ✅ Deterministic nonce generation (different ciphertexts from same plaintext)
- ✅ No-op mode for development (empty key)
- ✅ Invalid ciphertext handling
- ✅ Wrong key detection
- ✅ Large data support (1MB files)

**Run with**:
```bash
make test-encryption
```

### 4. Error Handling Tests (`internal/errors/errors_test.go`)

**Coverage**: Custom AppError type with proper HTTP status codes

- ✅ Invalid request error (400 Bad Request)
- ✅ Not found error (404 Not Found)
- ✅ Unauthorized error (401 Unauthorized)
- ✅ Forbidden error (403 Forbidden)
- ✅ Conflict error (409 Conflict)
- ✅ Error details attachment
- ✅ Error unwrapping (error chaining)

**Run with**:
```bash
make test-errors
```

### 5. Database Tests (`internal/db/db_test.go`)

**Coverage**: Connection pooling and database initialization

- ✅ Invalid connection string handling
- ✅ Connection pool stats validation
- ✅ Pool health calculations
- ✅ Connection limits verification

**Note**: Full integration tests require a running PostgreSQL instance.

### 6. Rate Limiting Tests (`internal/middleware/rate_limit_test.go`)

**Coverage**: In-memory rate limiter for DDoS protection

- ✅ New limiter initialization
- ✅ Allows requests under limit (100 req/min)
- ✅ Blocks excess requests
- ✅ Separate limits per client (IP-based)
- ✅ Window reset mechanism
- ✅ Configurable limits via `isAllowedWithLimit()`
- ✅ Memory cleanup of old entries

**Run with**:
```bash
make test-middleware
```

### 7. Handler Tests (`internal/handlers/handlers_test.go`)

**Coverage**: HTTP endpoint behavior and authorization

- ✅ Unauthorized request handling
- ✅ Invalid request body rejection
- ✅ Health check endpoints (no auth required)
- ✅ Project access verification
- ✅ Share link generation validation

### 8. FinOps Tests (`internal/finops/policy_test.go`)

**Coverage**: Cost optimization and budget management

- ✅ Settings loading from environment
- ✅ Budget decision structure
- ✅ Cost utilization tracking
- ✅ Budget threshold validation

## Test Quality Metrics

### Coverage Goals
- **Authentication**: 95%+ (critical security)
- **Validation**: 100% (prevents bugs upstream)
- **Encryption**: 100% (security-critical)
- **Error Handling**: 90%+
- **Middleware**: 85%+ (some integration testing needed)
- **Overall**: 80%+

### Current Status
Run `make test-coverage` to see current coverage percentages.

## Best Practices

### Writing New Tests

1. **Follow the pattern**:
   ```go
   func TestFunctionName_Scenario(t *testing.T) {
       // Arrange
       input := "test data"
       expected := "expected result"
       
       // Act
       result := FunctionUnderTest(input)
       
       // Assert
       if result != expected {
           t.Errorf("FunctionUnderTest(%q) = %q, expected %q", input, result, expected)
       }
   }
   ```

2. **Test edge cases**:
   - Boundary values (empty, max length, min length)
   - Invalid inputs
   - Null/nil values
   - Concurrent access (race conditions)

3. **Use table-driven tests** for multiple scenarios:
   ```go
   tests := []struct {
       name      string
       input     string
       shouldErr bool
   }{
       {"valid", "data", false},
       {"empty", "", true},
   }
   
   for _, tt := range tests {
       t.Run(tt.name, func(t *testing.T) {
           // Test implementation
       })
   }
   ```

4. **Keep tests focused**: Each test should verify one behavior
5. **Use meaningful names**: Describe what is being tested and the scenario
6. **Avoid dependencies**: Tests should be independent and runnable in any order

## Continuous Integration

These tests should be run:
- **Before commit**: Via pre-commit hooks
- **On PR**: GitHub Actions CI/CD pipeline
- **Before deployment**: Part of release-gate

```bash
# Full quality gate
./scripts/release_gate.sh

# Production-ready gate
./scripts/s_tier_gate.sh
```

## Performance Testing

Load tests are available in [tests/load_test.js](tests/load_test.js) using k6:

```bash
k6 run tests/load_test.js \
  -e API_URL=http://localhost:8080 \
  -e JWT_TOKEN="your-token"
```

This tests:
- Project listing (pagination)
- Track playback (signed URL generation)
- Health checks
- SLA compliance (95% < 500ms, 99% < 2s)

## Debugging Tests

### Verbose output
```bash
go test -v -run TestName ./package
```

### With race detection
```bash
go test -race -run TestName ./package
```

### With specific test case
```bash
go test -v -run TestValidatePassword_TooShort ./internal/validation
```

### See test execution
```bash
go test -v -count=1 ./...
```

## Known Limitations

- Database `*db.go` tests use mocked database (full integration tests require PostgreSQL)
- Handler tests are basic (full HTTP integration tests would require test fixtures)
- Middleware tests are unit-level (integration tests with full middleware chain recommended)

## Future Improvements

1. Add integration tests with containerized PostgreSQL (testcontainers-go)
2. Add mock database layer for comprehensive handler testing
3. Add performance benchmarks for encryption/compression
4. Add mutation testing to verify test quality
5. Add fuzz testing for input validation and parsing
