package ingest

import "fmt"

type ErrorClass int

const (
	ErrorRetryable ErrorClass = iota
	ErrorPermanent
)

type ClassifiedError struct {
	Class   ErrorClass
	Code    string
	Message string
}

func (e *ClassifiedError) Error() string {
	return e.Message
}

func Permanent(code, msg string) error {
	return &ClassifiedError{Class: ErrorPermanent, Code: code, Message: msg}
}

func Retryable(code, msg string) error {
	return &ClassifiedError{Class: ErrorRetryable, Code: code, Message: msg}
}

func Classify(err error) *ClassifiedError {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*ClassifiedError); ok {
		return ce
	}
	return &ClassifiedError{
		Class:   ErrorRetryable,
		Code:    "INGEST_FAILED",
		Message: err.Error(),
	}
}

func WrapPermanent(code string, err error) error {
	if err == nil {
		return nil
	}
	return Permanent(code, err.Error())
}

func WrapRetryable(code string, err error) error {
	if err == nil {
		return nil
	}
	return Retryable(code, fmt.Sprintf("%v", err))
}
