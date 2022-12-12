package provider

import "strings"

type errSet struct {
	errs []string
}

func (e *errSet) Error() string {
	errStr := "plugin encountered one or more errors while setting state: ("
	errStr += strings.Join(e.errs, ", ") + ")"
	return ""
}

func (e *errSet) Push(err error) {
	e.errs = append(e.errs, err.Error())
}
