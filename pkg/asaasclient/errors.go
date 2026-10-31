package asaasclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
)

// ErrorDetail is one entry of the "errors" array returned by Asaas.
type ErrorDetail struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// APIError is returned when Asaas answers with a non-2xx status code.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Errors     []ErrorDetail
	// Body holds the raw response body when it is not the standard
	// {"errors": [...]} envelope.
	Body string
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		parts := make([]string, len(e.Errors))
		for i, d := range e.Errors {
			parts[i] = d.Description
		}
		return fmt.Sprintf("asaas API %s %s returned HTTP %d: %s", e.Method, e.Path, e.StatusCode, strings.Join(parts, "; "))
	}
	return fmt.Sprintf("asaas API %s %s returned HTTP %d", e.Method, e.Path, e.StatusCode)
}

func newAPIError(method, path string, status int, body []byte) *APIError {
	e := &APIError{StatusCode: status, Method: method, Path: path}
	var envelope struct {
		Errors []ErrorDetail `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		e.Errors = envelope.Errors
	} else {
		e.Body = strings.TrimSpace(string(body))
	}
	return e
}

// NetworkError wraps transport failures (DNS, TLS, timeouts, resets...).
type NetworkError struct {
	Method string
	URL    string
	Err    error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error on %s %s: %v", e.Method, e.URL, e.Err)
}

func (e *NetworkError) Unwrap() error { return e.Err }

// Timeout reports whether the failure was caused by a timeout.
func (e *NetworkError) Timeout() bool {
	var ne net.Error
	return errors.As(e.Err, &ne) && ne.Timeout()
}

// ResponseError is returned when a successful response cannot be decoded.
type ResponseError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("unexpected response (HTTP %d): %s", e.StatusCode, e.Message)
}
