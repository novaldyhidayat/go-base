package response

import "net/http"

// Envelope defines a standard API response structure.
type Envelope struct {
	Data  interface{} `json:"data,omitempty"`
	Error *Error      `json:"error,omitempty"`
	Meta  interface{} `json:"meta,omitempty"`
}

// Error defines a standard error response payload.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// JSON wraps successful responses.
func JSON(data interface{}, meta interface{}) Envelope {
	return Envelope{Data: data, Meta: meta}
}

// Fail wraps error responses.
func Fail(code string, message string, details interface{}) (int, Envelope) {
	return Error(http.StatusBadRequest, code, message, details)
}

// Error builds an error envelope with custom HTTP status.
func Error(status int, code string, message string, details interface{}) (int, Envelope) {
	return status, Envelope{Error: &Error{Code: code, Message: message, Details: details}}
}
