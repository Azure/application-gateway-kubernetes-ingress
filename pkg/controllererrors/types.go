// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controllererrors

import (
	"fmt"
)

// ErrorCode is string type for error codes
type ErrorCode string

// Error is complex error type
type Error struct {
	Code       ErrorCode
	Message    string
	InnerError error
}

// NewError creates new error
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithInnerError creates new error
func NewErrorWithInnerError(code ErrorCode, innerError error, message string) *Error {
	return &Error{
		Code:       code,
		InnerError: innerError,
		Message:    message,
	}
}

// NewErrorf creates new error after formatting
func NewErrorf(code ErrorCode, message string, a ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(message, a...),
	}
}

// NewErrorWithInnerErrorf creates new error
func NewErrorWithInnerErrorf(code ErrorCode, innerError error, message string, a ...interface{}) *Error {
	return &Error{
		Code:       code,
		InnerError: innerError,
		Message:    fmt.Sprintf(message, a...),
	}
}

// Error implements error interface to return error
func (e *Error) Error() string {
	if e.InnerError != nil {
		return fmt.Sprintf("Code: %s, Message: %s, InnerError: %s", e.Code, e.Message, e.InnerError.Error())
	}

	return fmt.Sprintf("Code: %s, Message: %s", e.Code, e.Message)
}

// IsErrorCode matches error code to the error
func IsErrorCode(err error, code ErrorCode) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}

	return false
}
