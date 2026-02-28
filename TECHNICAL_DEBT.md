# 🏗️ Technical Debt Tracker

Track known issues, deprecations, and areas needing refactoring.

## 🔴 Critical (Blocks Production)

None currently.

## 🟡 High Priority (Q2-Q3 2026)

### 1. Error Handling in Retention Jobs

**Files**: 
- `internal/retention/manager.go`

**Issue**: Unchecked error returns in:
- `drm.ExecutePlayHistoryRetention()`
- `drm.ExecuteOfflineDownloadRetention()`
- `drm.ExecuteAuditLogRetention()`

**Impact**: Retention jobs may fail silently; data not cleaned up

**Fix**: Wrap with error handling and logging:
```go
if err := drm.ExecutePlayHistoryRetention(); err != nil {
  logger.Error("failed to execute play history retention", err)
  metrics.IncrementRetentionFailures("play_history")
}
```

**Effort**: 2 hours | **Owner**: TBD

---

### 2. AWS S3 Manager Deprecation

**Files**:
- `internal/storage/r2_client.go`

**Issue**: Using deprecated `manager.Uploader` (AWS SDK v2 feature)

**Deprecated Since**: AWS SDK v0.73.0  
**Replacement**: Use `feature/s3/transfermanager`

**Impact**: Will break in AWS SDK v2.0.0

**Fix**: 
```bash
go get github.com/aws/aws-sdk-go-v2@latest
# Update R2Client.Upload() to use transfermanager
```

**Effort**: 4 hours | **Owner**: TBD

---

### 3. JSON Encoding Error Handling

**Files**:
- `internal/handlers/*.go` (multiple)
- `internal/api/response.go`

**Issue**: `(*json.Encoder).Encode()` return values not checked

**Impact**: Response encoding failures silently ignored

**Fix**: Add error handling:
```go
encoder := json.NewEncoder(w)
if err := encoder.Encode(data); err != nil {
  logger.Error("failed to encode response", err)
  http.Error(w, "internal error", 500)
}
```

**Effort**: 3 hours | **Owner**: TBD

---

### 4. Nil Pointer Dereferences

**Files**:
- `internal/models/*.go`
- `internal/handlers/*.go`

**Issue**: Possible nil pointer dereference in:
- Database query results
- Parsed request bodies
- Configuration values

**Impact**: Server crashes on edge cases

**Fix**: Add nil checks before dereference:
```go
if user == nil {
  return errors.New("user not found")
}
```

**Effort**: 5 hours | **Owner**: TBD

---

### 5. Transaction Rollback Errors

**Files**:
- `internal/db/*.go`

**Issue**: `tx.Rollback()` return values not checked (in defer blocks)

**Fix**:
```go
defer func() {
  if err := tx.Rollback(); err != nil {
    logger.Error("failed to rollback transaction", err)
  }
}()
```

**Effort**: 2 hours | **Owner**: TBD

---

## 🟠 Medium Priority (Q3-Q4 2026)

### 6. Unused Code Cleanup

**Issue**: Unused fields and functions detected by linter:
- `mu` (mutex in cache manager)
- `eventBus` (event dispatcher)
- Constants: `userRoleCacheKey`, `projectAccessCacheKey`
- Function: `scanInt()` helper

**Impact**: Code confusion, maintenance burden

**Fix**: 
- Remove if truly unused
- Export if needed internally
- Document if reserved for future use

**Effort**: 1 hour | **Owner**: TBD

---

### 7. Context Keys Type Safety

**Files**: `internal/middleware/context.go`

**Issue**: Using `string` as context key (SA1029) instead of custom type

**Current**:
```go
ctx.Value("user_id")  // string key
```

**Should be**:
```go
type contextKey string
const userIDKey contextKey = "user_id"
ctx.Value(userIDKey)  // type-safe
```

**Impact**: Type safety, prevents key collisions

**Effort**: 1 hour | **Owner**: TBD

---

### 8. Simplification Opportunities

**Issue**: 
- Unnecessary blank identifier assignments (S1005)
- `make([]byte, 256)` instead of `make([]byte, 256)`

**Impact**: Code clarity, performance

**Effort**: 1 hour (automatic with gofmt)

---

## 🟢 Low Priority (Nice to Have)

### 9. Configuration Assignment Cleanup

**Files**: `internal/config/loader.go`

**Issue**: Ineffectual assignments to:
- `maxConns`
- `minConns`
- `maxConcurrent`
- `daysElapsed`

**Impact**: Dead code, potential logic errors

**Effort**: 1 hour | **Owner**: TBD

---

### 10. godotenv Error Handling

**Files**: `cmd/api/main.go`

**Issue**: `godotenv.Load()` error not checked (should probably be ignored if .env missing)

**Fix**:
```go
_ = godotenv.Load()  // Explicitly ignore - okay if .env not present
```

**Effort**: 15 minutes | **Owner**: TBD

---

## 📊 Summary

| Priority | Count | Effort | Owner |
|----------|-------|--------|-------|
| 🔴 Critical | 0 | 0h | - |
| 🟡 High | 5 | 16h | TBD |
| 🟠 Medium | 3 | 3h | TBD |
| 🟢 Low | 2 | 1.25h | TBD |
| **TOTAL** | **10** | **20.25h** | **TBD** |

---

## 🔧 How to Address

### Daily Development
1. Run linter before committing: `golangci-lint run ./...`
2. Fix critical issues immediately
3. Log medium/low issues to this tracker

### Sprint Planning
- Allocate ~5-10% of sprint for technical debt
- Prioritize high-priority items quarterly

### Release Checklist (Before v1.1.0)
- [ ] All 🔴 critical issues resolved
- [ ] All 🟡 high priority items fixed or documented
- [ ] Code review checklist includes linter compliance
- [ ] Add CI gate: "Must pass all linters on main"

---

## 🚀 Prevention Going Forward

1. **Pre-commit Hook**:
   ```bash
   git config --local core.hooksPath .githooks
   # Add .githooks/pre-commit with: golangci-lint run
   ```

2. **CI Gate**: Already in `.github/workflows/ci-tests.yml`

3. **IDE Integration**: 
   - VS Code: Install `golang.go` extension
   - GoLand: Built-in linter support

4. **Code Review**: Check linter output in PR reviews

---

**Last Updated**: February 28, 2026  
**Status**: Active - Tracking for v1.0.0 release
