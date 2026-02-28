package models

// Response represents a standard API response [LOW: Code organization]
type Response[T any] struct {
	Data   T              `json:"data,omitempty"`
	Error  *ErrorResponse `json:"error,omitempty"`
	Status string         `json:"status"` // "success" or "error"
}

// ErrorResponse represents error details in response [LOW: Code organization]
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SuccessResponse creates a success response [LOW: Consistency]
func SuccessResponse[T any](data T) *Response[T] {
	return &Response[T]{
		Data:   data,
		Status: "success",
	}
}

// ErrorResponseData creates an error response [LOW: Consistency]
func ErrorResponseData[T any](code string, message string, details map[string]interface{}) *Response[T] {
	return &Response[T]{
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
		Status: "error",
	}
}

// Pagination represents pagination metadata [LOW: Consistency]
type Pagination struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
	Pages  int64 `json:"pages"`
}

// PaginatedResponse represents a paginated response [LOW: Consistency]
type PaginatedResponse[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
	Status     string     `json:"status"` // "success" or "error"
}

// BuildPaginatedResponse creates a paginated response [LOW: Code organization]
func BuildPaginatedResponse[T any](data []T, limit int, offset int, total int64) *PaginatedResponse[T] {
	pages := int64(1)
	if total > 0 && int64(limit) > 0 {
		pages = (total + int64(limit) - 1) / int64(limit)
	}

	return &PaginatedResponse[T]{
		Data: data,
		Pagination: Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  total,
			Pages:  pages,
		},
		Status: "success",
	}
}

// CreatedResponse represents a resource creation response [LOW: Consistency]
type CreatedResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

// BuildCreatedResponse creates a creation response [LOW: Code organization]
func BuildCreatedResponse(id string, createdAt interface{}) *CreatedResponse {
	createdAtStr := ""
	if createdAt != nil {
		createdAtStr = createdAt.(string)
	}
	return &CreatedResponse{
		ID:        id,
		CreatedAt: createdAtStr,
	}
}

// DeletedResponse represents a deletion response [LOW: Consistency]
type DeletedResponse struct {
	ID        string `json:"id"`
	DeletedAt string `json:"deleted_at"`
}

// BuildDeletedResponse creates a deletion response [LOW: Code organization]
func BuildDeletedResponse(id string) *DeletedResponse {
	return &DeletedResponse{
		ID: id,
	}
}

// CountResponse represents a count response [LOW: Consistency]
type CountResponse struct {
	Count int64 `json:"count"`
}

// BuildCountResponse creates a count response [LOW: Code organization]
func BuildCountResponse(count int64) *CountResponse {
	return &CountResponse{Count: count}
}

// HealthResponse represents health check response [LOW: Consistency]
type HealthResponse struct {
	Status string      `json:"status"`
	Time   string      `json:"time"`
	Uptime int64       `json:"uptime_seconds"`
	Info   interface{} `json:"info,omitempty"`
}

// BuildHealthResponse creates a health response [LOW: Code organization]
func BuildHealthResponse(status string, uptime int64) *HealthResponse {
	return &HealthResponse{
		Status: status,
		Uptime: uptime,
	}
}

// ListResponse represents a list response in standard format [LOW: Consistency]
type ListResponse[T any] struct {
	Items  []T    `json:"items"`
	Status string `json:"status"` // "success" or "error"
	Count  int    `json:"count"`
}

// BuildListResponse creates a list response [LOW: Code organization]
func BuildListResponse[T any](items []T) *ListResponse[T] {
	return &ListResponse[T]{
		Items:  items,
		Status: "success",
		Count:  len(items),
	}
}

// MessageResponse represents a simple message response [LOW: Consistency]
type MessageResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// BuildMessageResponse creates a message response [LOW: Code organization]
func BuildMessageResponse(message string) *MessageResponse {
	return &MessageResponse{
		Message: message,
		Status:  "success",
	}
}
