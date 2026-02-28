// Package helpers provides utility functions and helpers [LOW: Documentation]
//
// HTTP Response Helpers:
// Use these functions to send consistent HTTP responses
//
//   helpers.WriteJSON(w, http.StatusOK, data)           // 200 OK with JSON
//   helpers.WriteBadRequest(w, "message")               // 400 Bad Request
//   helpers.WriteUnauthorized(w)                        // 401 Unauthorized
//   helpers.WriteForbidden(w, "forbidden message")      // 403 Forbidden
//   helpers.WriteNotFound(w, "resource not found")      // 404 Not Found
//   helpers.WriteCreated(w, data)                       // 201 Created
//   helpers.WriteNoContent(w)                           // 204 No Content
//   helpers.WriteInternalError(w, log, err)             // 500 Internal Server Error
//
// Context Extraction:
//   userID, err := helpers.GetUserID(r)                 // Extract user ID from context
//   userID, ok := helpers.GetUserIDSafe(r)              // Safe extraction without error
//
// Type Safe Helpers:
//   value, err := helpers.SafeTypeAssertion[Type](v)    // Type assertion with error
//   err := helpers.SafeJSONDecode(r.Body, &obj)         // JSON decode with strict fields
//   err := helpers.SafeJSONEncode(w, obj)               // JSON encode with HTML escaping
//   param, ok := helpers.GetPathValue(r, "id")          // Get path parameter safely
//   param, ok := helpers.GetQueryParam(r, "key")        // Get query parameter safely
//
// Nil-Safe Dereference:
//   helpers.SafeString(ptr)                             // Dereference *string
//   helpers.SafeInt64(ptr)                              // Dereference *int64
//   helpers.SafeInt(ptr)                                // Dereference *int
//   helpers.SafeFloat64(ptr)                            // Dereference *float64
//   helpers.SafeBool(ptr)                               // Dereference *bool
//
// Pointer Creation:
//   helpers.Ptr(value)                                  // Create *T from T
//   helpers.NilIfEmpty(str)                             // Create *string or nil
//   helpers.NilIfZero(num)                              // Create *int or nil
//
// Validation Helpers:
//   helpers.ValidateAndParseInt(str, min, max)          // Parse int with bounds
//   helpers.ExtractPaginationParams(r)                  // Get limit/offset from query
//   helpers.ValidateSortField(field, allowed)           // Validate sort field
//   helpers.SanitizeFilename(filename)                  // Remove unsafe chars from filename
//
// Usage Patterns:
//
//   1. Extract and validate user ID:
//     userID, ok := helpers.GetUserIDSafe(r)
//     if !ok {
//       helpers.WriteUnauthorized(w)
//       return
//     }
//
//   2. Safe type assertion:
//     value, err := helpers.SafeTypeAssertion[MyType](untrustedValue)
//     if err != nil {
//       helpers.WriteBadRequest(w, fmt.Sprintf("invalid type: %v", err))
//       return
//     }
//
//   3. Safe JSON handling:
//     var req RequestType
//     if err := helpers.SafeJSONDecode(r.Body, &req); err != nil {
//       helpers.WriteBadRequest(w, "invalid request body")
//       return
//     }
//
//   4. Response with error handling:
//     if err := helpers.WriteJSON(w, http.StatusOK, data); err != nil {
//       log.Error("failed to write response", "error", err)
//     }
//
// Error Handling:
// All helpers return errors for proper error handling.
// Always check errors and log them appropriately.
package helpers

import _ "encoding/json" // Required for JSON operations
import _ "net/http"      // Required for HTTP operations
