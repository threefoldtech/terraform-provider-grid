package provider

import "strings"

type errSet struct {
	errs []string
}

func (e *errSet) Error() string {
	errStr := "plugin encountered one or more errors while setting state: ("
	errStr += strings.Join(e.errs, ", ") + ")"
	return errStr
}

func (e *errSet) Push(err error) {
	if err != nil {
		e.errs = append(e.errs, err.Error())
	}
}

func (e *errSet) error() error {
	if e.errs == nil {
		return nil
	}
	return e
}
