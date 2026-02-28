package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// QueryRequest represents incoming GraphQL query
type QueryRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// QueryResponse represents GraphQL query response
type QueryResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []string    `json:"errors,omitempty"`
}

// ExecuteQuery executes a parsed GraphQL query against the resolver
func (r *Resolver) ExecuteQuery(ctx context.Context, req *QueryRequest) *QueryResponse {
	query := strings.TrimSpace(req.Query)

	// Parse query type (query or mutation)
	if strings.Contains(query, "mutation ") || strings.Contains(query, "mutation{") {
		return r.executeMutation(ctx, query, req.Variables)
	}

	return r.executeQuery(ctx, query, req.Variables)
}

func (r *Resolver) executeQuery(ctx context.Context, query string, variables map[string]interface{}) *QueryResponse {
	// Extract field name from query: "{ fieldName(args) { subFields } }"
	fieldName := extractFieldName(query)
	if fieldName == "" {
		return &QueryResponse{Errors: []string{"invalid query format"}}
	}

	// Route to appropriate resolver method
	switch fieldName {
	case "me":
		user, err := r.Me(ctx)
		if err != nil {
			return &QueryResponse{Errors: []string{err.Error()}}
		}
		return &QueryResponse{Data: map[string]interface{}{"me": user}}

	case "projects":
		limit, offset := extractPagination(query, variables)
		conn, err := r.Projects(ctx, limit, offset)
		if err != nil {
			return &QueryResponse{Errors: []string{err.Error()}}
		}
		return &QueryResponse{Data: map[string]interface{}{"projects": conn}}

	case "notifications":
		limit, offset := extractPagination(query, variables)
		conn, err := r.Notifications(ctx, limit, offset)
		if err != nil {
			return &QueryResponse{Errors: []string{err.Error()}}
		}
		return &QueryResponse{Data: map[string]interface{}{"notifications": conn}}

	case "unreadNotificationCount":
		count, err := r.UnreadNotificationCount(ctx)
		if err != nil {
			return &QueryResponse{Errors: []string{err.Error()}}
		}
		return &QueryResponse{Data: map[string]interface{}{"unreadNotificationCount": count}}

	case "user":
		userID := extractArgument(query, "id", variables)
		if userID == "" {
			return &QueryResponse{Errors: []string{"missing required argument: id"}}
		}
		user, err := r.User(ctx, userID)
		if err != nil {
			return &QueryResponse{Errors: []string{err.Error()}}
		}
		return &QueryResponse{Data: map[string]interface{}{"user": user}}

	default:
		return &QueryResponse{Errors: []string{fmt.Sprintf("unknown query field: %s", fieldName)}}
	}
}

func (r *Resolver) executeMutation(ctx context.Context, query string, variables map[string]interface{}) *QueryResponse {
	fieldName := extractFieldName(query)
	if fieldName == "" {
		return &QueryResponse{Errors: []string{"invalid mutation format"}}
	}

	switch fieldName {
	case "registerDeviceToken":
		token := extractArgument(query, "token", variables)
		platform := extractArgument(query, "platform", variables)
		if token == "" || platform == "" {
			return &QueryResponse{Errors: []string{"missing required arguments: token, platform"}}
		}
		dt, err := r.RegisterDeviceToken(ctx, token, platform)
		if err != nil {
			return &QueryResponse{Errors: []string{err.Error()}}
		}
		return &QueryResponse{Data: map[string]interface{}{"registerDeviceToken": dt}}

	case "updateNotificationPreferences":
		input := extractPreferencesInput(query, variables)
		prefs, err := r.UpdateNotificationPreferences(ctx, input)
		if err != nil {
			return &QueryResponse{Errors: []string{err.Error()}}
		}
		return &QueryResponse{Data: map[string]interface{}{"updateNotificationPreferences": prefs}}

	case "markNotificationsAsRead":
		notificationIDs := extractIDArray(query, "ids", variables)
		if len(notificationIDs) == 0 {
			return &QueryResponse{Errors: []string{"missing required argument: ids"}}
		}
		count, err := r.MarkNotificationsAsRead(ctx, notificationIDs)
		if err != nil {
			return &QueryResponse{Errors: []string{err.Error()}}
		}
		return &QueryResponse{Data: map[string]interface{}{"markNotificationsAsRead": count}}

	default:
		return &QueryResponse{Errors: []string{fmt.Sprintf("unknown mutation field: %s", fieldName)}}
	}
}

// Helper functions for query parsing

func extractFieldName(query string) string {
	// Remove extra whitespace and normalize
	query = strings.Join(strings.Fields(query), " ")

	// Extract field name: "{ fieldName(...)" or "mutation { fieldName(...)"
	if idx := strings.Index(query, "{"); idx != -1 {
		remainder := query[idx+1:]
		remainder = strings.TrimSpace(remainder)

		// Get word until '(' or whitespace
		var fieldName string
		for i, ch := range remainder {
			if ch == '(' || ch == ' ' || ch == '{' || ch == '\n' {
				fieldName = remainder[:i]
				break
			}
			fieldName = remainder[:i+1]
		}
		return strings.TrimSpace(fieldName)
	}

	return ""
}

func extractArgument(query string, argName string, variables map[string]interface{}) string {
	// Look for argument: argName: "value" or argName: $varName
	pattern := argName + ":"

	if idx := strings.Index(query, pattern); idx != -1 {
		remainder := query[idx+len(pattern):]
		remainder = strings.TrimSpace(remainder)

		// Check if it's a variable reference
		if remainder[0] == '$' {
			varName := extractVarName(remainder)
			if v, ok := variables[varName]; ok {
				return fmt.Sprintf("%v", v)
			}
		}

		// Extract quoted string value
		if remainder[0] == '"' {
			endIdx := strings.Index(remainder[1:], "\"")
			if endIdx != -1 {
				return remainder[1 : endIdx+1]
			}
		}
	}

	return ""
}

func extractVarName(s string) string {
	s = strings.TrimSpace(s)
	if s[0] == '$' {
		s = s[1:]
	}
	var name string
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			name += string(ch)
		} else {
			break
		}
	}
	return name
}

func extractIDArray(query string, argName string, variables map[string]interface{}) []string {
	pattern := argName + ":"
	if idx := strings.Index(query, pattern); idx != -1 {
		remainder := query[idx+len(pattern):]
		remainder = strings.TrimSpace(remainder)

		// Check for variable reference
		if remainder[0] == '$' {
			varName := extractVarName(remainder)
			if v, ok := variables[varName]; ok {
				// Handle both []interface{} and []string
				switch arr := v.(type) {
				case []interface{}:
					result := make([]string, len(arr))
					for i, item := range arr {
						result[i] = fmt.Sprintf("%v", item)
					}
					return result
				case []string:
					return arr
				}
			}
		}
	}
	return []string{}
}

func extractPagination(query string, variables map[string]interface{}) (int, int) {
	limit := 20
	offset := 0

	// Extract limit argument
	if limitStr := extractArgument(query, "limit", variables); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Extract offset argument
	if offsetStr := extractArgument(query, "offset", variables); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}

func extractPreferencesInput(query string, variables map[string]interface{}) *NotificationPreferencesInput {
	input := &NotificationPreferencesInput{}

	// Extract input argument
	pattern := "input:"
	if idx := strings.Index(query, pattern); idx != -1 {
		remainder := query[idx+len(pattern):]
		remainder = strings.TrimSpace(remainder)

		// Check for variable reference
		if remainder[0] == '$' {
			varName := extractVarName(remainder)
			if v, ok := variables[varName]; ok {
				// Convert map to NotificationPreferencesInput
				if m, ok := v.(map[string]interface{}); ok {
					if val, ok := m["pushEnabled"].(bool); ok {
						input.PushEnabled = &val
					}
					if val, ok := m["pushLikes"].(bool); ok {
						input.PushLikes = &val
					}
					if val, ok := m["pushComments"].(bool); ok {
						input.PushComments = &val
					}
					if val, ok := m["pushFollows"].(bool); ok {
						input.PushFollows = &val
					}
					if val, ok := m["pushShares"].(bool); ok {
						input.PushShares = &val
					}
					if val, ok := m["pushMentions"].(bool); ok {
						input.PushMentions = &val
					}
					if val, ok := m["quietHoursEnabled"].(bool); ok {
						input.QuietHoursEnabled = &val
					}
					if val, ok := m["quietHoursStart"].(string); ok {
						input.QuietHoursStart = &val
					}
					if val, ok := m["quietHoursEnd"].(string); ok {
						input.QuietHoursEnd = &val
					}
					if val, ok := m["bundleByType"].(bool); ok {
						input.BundleByType = &val
					}
				}
			}
		}
	}

	return input
}

// StructToMap converts a struct to a map for JSON serialization
func StructToMap(s interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	v := reflect.ValueOf(s)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return result
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			jsonTag = field.Name
		} else {
			jsonTag = strings.Split(jsonTag, ",")[0]
		}

		result[jsonTag] = v.Field(i).Interface()
	}

	return result
}

// MarshalGraphQLResponse converts response to JSON with proper GraphQL structure
func MarshalGraphQLResponse(response *QueryResponse) ([]byte, error) {
	return json.Marshal(response)
}
