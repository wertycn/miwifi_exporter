package errors

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

type ErrorType string

const (
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeInternal       ErrorType = "internal"
)

type AppError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    int       `json:"code"`
	Cause   error    `json:"cause,omitempty"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func NewAuthenticationError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrorTypeAuthentication,
		Message: message,
		Code:    http.StatusUnauthorized,
		Cause:   cause,
	}
}

func NewNetworkError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrorTypeNetwork,
		Message: message,
		Code:    http.StatusBadGateway,
		Cause:   cause,
	}
}

func NewTimeoutError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrorTypeTimeout,
		Message: message,
		Code:    http.StatusRequestTimeout,
		Cause:   cause,
	}
}

func NewValidationError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrorTypeValidation,
		Message: message,
		Code:    http.StatusBadRequest,
		Cause:   cause,
	}
}

func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrorTypeInternal,
		Message: message,
		Code:    http.StatusInternalServerError,
		Cause:   cause,
	}
}

func IsAuthenticationError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == ErrorTypeAuthentication
}

func IsNetworkError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == ErrorTypeNetwork
}

func IsTimeoutError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == ErrorTypeTimeout
}

func IsValidationError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Type == ErrorTypeValidation
}

type RetryHandler struct {
	maxRetries int
	maxDelay   time.Duration
	logger     Logger
}

type Logger interface {
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func NewRetryHandler(maxRetries int, maxDelay time.Duration, logger Logger) *RetryHandler {
	return &RetryHandler{
		maxRetries: maxRetries,
		maxDelay:   maxDelay,
		logger:     logger,
	}
}

func (r *RetryHandler) WithRetry(fn func() error) error {
	var lastErr error
	
	for i := 0; i < r.maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// 如果是验证错误，不重试
		if IsAuthenticationError(err) || IsValidationError(err) {
			return err
		}
		
		// 计算延迟时间
		delay := time.Duration(i+1) * time.Second
		if delay > r.maxDelay {
			delay = r.maxDelay
		}
		
		r.logger.Warnf("Attempt %d failed: %v, retrying in %v...", i+1, err, delay)
		time.Sleep(delay)
	}
	
	return fmt.Errorf("after %d retries: %w", r.maxRetries, lastErr)
}