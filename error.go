package fetch

type InvalidRequestError struct {
	err error
}

func (ire *InvalidRequestError) Error() string {
	return ire.err.Error()
}

func (e *InvalidRequestError) Unwrap() error {
	return e.err
}
