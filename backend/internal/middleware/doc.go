// Package middleware provides HTTP middleware for the PurpTape API [LOW: Documentation]
//
// Standard Middleware Stack (in order):
// 1. RecoveryMiddleware - Catches panics and logs them
// 2. ContextMiddleware - Extracts request context (request ID, IP, user agent)
// 3. RequestMetricsMiddleware - Tracks metrics (duration, bytes, status)
// 4. RateLimitMiddleware - Enforces rate limits
// 5. AuthMiddleware - Validates JWT and extracts user ID
// 6. RBACMiddleware - Checks authorization
//
// Usage:
//   router.Use(
//     middleware.Chain(
//       middleware.RecoveryMiddleware(log),
//       middleware.ContextMiddleware(log),
//       middleware.RequestMetricsMiddleware(log),
//     ),
//   )
//
// Common Middleware Patterns:
//
// 1. Request/Response Modification:
//   return func(next http.Handler) http.Handler {
//     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//       // Before: modify request
//       next.ServeHTTP(w, r)
//       // After: modify response (if wrapped)
//     })
//   }
//
// 2. Access Control:
//   return func(next http.Handler) http.Handler {
//     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//       if !isAuthorized(r) {
//         http.Error(w, "unauthorized", http.StatusUnauthorized)
//         return
//       }
//       next.ServeHTTP(w, r)
//     })
//   }
//
// 3. Context Enhancement:
//   ctx := context.WithValue(r.Context(), "key", "value")
//   r = r.WithContext(ctx)
//   next.ServeHTTP(w, r)
//
// Middleware Ordering Notes:
// - Outer middleware executes first (like layers)
// - Use Chain to apply in readable order
// - Recovery should be outermost (catches panics)
// - Auth should be inner (after metrics)
// - Use ContextMiddleware early to set request context
//
// Performance Considerations:
// - RecoveryMiddleware has minimal overhead
// - MetricsMiddleware wraps response (small overhead)
// - RateLimitMiddleware uses efficient checking
// - AuthMiddleware may cache tokens (see auth package)
package middleware

import _ "net/http" // Required for middleware operations
