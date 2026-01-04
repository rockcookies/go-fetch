package fetch

// InvalidRequestError wraps errors that occur during request construction.
// This typically includes URL parsing errors or invalid options.
type InvalidRequestError struct {
	err error
}

// Error returns the error message.
func (ire *InvalidRequestError) Error() string {
	return ire.err.Error()
}

// Unwrap returns the underlying error for error chain inspection.
func (e *InvalidRequestError) Unwrap() error {
	return e.err
}
