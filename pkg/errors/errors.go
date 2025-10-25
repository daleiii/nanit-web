package errors

import (
	"fmt"
)

// ErrorType represents different categories of errors in the application
type ErrorType string

const (
	// Authentication and authorization errors
	ErrorTypeAuth ErrorType = "authentication"
	
	// Configuration and validation errors
	ErrorTypeConfig ErrorType = "configuration"
	
	// Network and API communication errors
	ErrorTypeNetwork ErrorType = "network"
	
	// File system and storage errors
	ErrorTypeStorage ErrorType = "storage"
	
	// Business logic and validation errors
	ErrorTypeValidation ErrorType = "validation"
	
	// External service errors (RTMP, MQTT, etc.)
	ErrorTypeExternal ErrorType = "external_service"
)

// AppError represents a structured application error with context
type AppError struct {
	Type        ErrorType `json:"type"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Cause       error     `json:"-"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Retryable   bool      `json:"retryable"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for error wrapping
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewAuthError creates a new authentication error
func NewAuthError(code, message string, cause error) *AppError {
	return &AppError{
		Type:      ErrorTypeAuth,
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: false,
	}
}

// NewConfigError creates a new configuration error
func NewConfigError(code, message string, cause error) *AppError {
	return &AppError{
		Type:      ErrorTypeConfig,
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: false,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(code, message string, cause error) *AppError {
	return &AppError{
		Type:      ErrorTypeNetwork,
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: true,
	}
}

// NewStorageError creates a new storage error
func NewStorageError(code, message string, cause error) *AppError {
	return &AppError{
		Type:      ErrorTypeStorage,
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: false,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(code, message string, cause error) *AppError {
	return &AppError{
		Type:      ErrorTypeValidation,
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: false,
	}
}

// NewExternalError creates a new external service error
func NewExternalError(code, message string, cause error) *AppError {
	return &AppError{
		Type:      ErrorTypeExternal,
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: true,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Retryable
	}
	return false
}

// GetErrorType returns the error type if it's an AppError
func GetErrorType(err error) ErrorType {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type
	}
	return ""
}