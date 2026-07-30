package services

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Status  int
	Message string
	Err     error
}

func (err *Error) Error() string { return err.Message }
func (err *Error) Unwrap() error { return err.Err }

func clientError(status int, message string) error {
	return &Error{Status: status, Message: message}
}

func internalError(message string, err error) error {
	return &Error{Status: http.StatusInternalServerError, Message: message, Err: err}
}

func unavailableError(message string, err error) error {
	return &Error{Status: http.StatusServiceUnavailable, Message: message, Err: err}
}

func PublicError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	var serviceErr *Error
	if errors.As(err, &serviceErr) {
		return serviceErr.Status, serviceErr.Message
	}
	return http.StatusInternalServerError, "internal server error"
}

func wrapDatabase(err error) error {
	if err == nil {
		return nil
	}
	return internalError("database operation failed", fmt.Errorf("database: %w", err))
}
