package errors

import (
	"fmt"
)

type ErrorKind string

const (
	KindNetwork  ErrorKind = "NETWORK_ERROR"
	KindParsing  ErrorKind = "PARSING_ERROR"
	KindDatabase ErrorKind = "DATABASE_ERROR"
	KindUnknown  ErrorKind = "UNKNOWN_ERROR"
)

// MarrowError is our custom error type carrying context and kind.
type MarrowError struct {
	Kind    ErrorKind
	Source  string
	Err     error
	Message string
}

func (e *MarrowError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s (root: %v)", e.Kind, e.Source, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Kind, e.Source, e.Message)
}

func (e *MarrowError) Unwrap() error {
	return e.Err
}

// Helper constructors
func NewNetworkError(source string, msg string, err error) *MarrowError {
	return &MarrowError{Kind: KindNetwork, Source: source, Err: err, Message: msg}
}

func NewParsingError(source string, msg string, err error) *MarrowError {
	return &MarrowError{Kind: KindParsing, Source: source, Err: err, Message: msg}
}

func NewDatabaseError(source string, msg string, err error) *MarrowError {
	return &MarrowError{Kind: KindDatabase, Source: source, Err: err, Message: msg}
}