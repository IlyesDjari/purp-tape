// Package db provides database access and query operations [LOW: Documentation]
//
// Architecture:
// The database package follows the Repository pattern with pgx for PostgreSQL access.
//
// Connection Management:
// - Uses pgxpool for connection pooling
// - Automatic connection lifecycle management
// - Configurable max/min connections in config
// - Connection health monitoring
//
// Query Organization:
// - queries.go: Basic CRUD operations and common queries
// - queries_analytics.go: Analytics and aggregation queries
// - queries_search.go: Full-text and keyword search
// - queries_compliance.go: GDPR and compliance operations
// - queries_downloads.go: File download and streaming
// - queries_stats.go: Statistics and aggregation
// - optimized_queries.go: Performance-optimized queries
// - transactions.go: Multi-operation transactions
//
// Query Patterns:
//
// 1. Single Row Query:
//   var result Type
//   err := db.pool.QueryRow(ctx, "SELECT ... WHERE id = $1", id).Scan(&result.Fields...)
//   if err == sql.ErrNoRows {
//     return nil, nil // Not found
//   }
//   if err != nil {
//     return nil, fmt.Errorf("failed to query: %w", err)
//   }
//
// 2. Multiple Rows:
//   rows, err := db.pool.Query(ctx, "SELECT ...", args...)
//   if err != nil {
//     return nil, fmt.Errorf("failed to query: %w", err)
//   }
//   defer rows.Close()
//   var results []Type
//   for rows.Next() {
//     var item Type
//     if err := rows.Scan(&item.Fields...); err != nil {
//       return nil, err
//     }
//     results = append(results, item)
//   }
//   return results, rows.Err()
//
// 3. Batch Operations (with transaction):
//   return db.WithTx(ctx, func(tx *Tx) error {
//     for _, item := range items {
//       if err := tx.Exec(ctx, query, args...); err != nil {
//         return err  // Auto-rollback
//       }
//     }
//     return nil  // Auto-commit
//   })
//
// Error Handling:
// - All errors should be wrapped with context: fmt.Errorf("operation failed: %w", err)
// - Check for sql.ErrNoRows to distinguish "not found" from real errors
// - Database panics indicate serious issues (use recovery middleware)
//
// Context Usage:
// - Always pass context from request
// - All queries respect context cancellation
// - Use context.WithTimeout for long operations
//
// Pagination:
// - Use NewPaginationParams(limit, offset) to validate
// - Default limit 20, max 100
// - Always order results for consistent pagination
//
// Soft Deletes:
// - Use deleted_at IS NULL in WHERE clauses
// - Never hard delete production data
// - Use db.WithTx for cascading soft deletes
//
// Performance:
// - Use prepared statements for very hot paths (see optimized_queries.go)
// - Create indexes on frequently queried columns (see migrations)
// - Use EXPLAIN ANALYZE for query optimization
// - Avoid N+1 queries (use JOIN or batch queries)
//
// Testing:
// - Use migrations to set up test database
// - Use transactions to isolate tests (rollback after)
// - Mock *pgxpool.Pool for unit tests
package db

import _ "context" // Required for database operations
