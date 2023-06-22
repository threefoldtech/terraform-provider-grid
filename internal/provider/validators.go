package provider

import (
	"fmt"
	"regexp"
)

// IsGPUID is a SchemaValidateFunc which tests if the provided value is of type string and is a valid gpu
func IsGPUId(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %q to be string", k))
		return warnings, errors
	}

	match, _ := regexp.MatchString("^[A-Za-z0-9:.]+/[A-Za-z0-9]+/[A-Za-z0-9]+$", v)
	if !match {
		errors = append(errors, fmt.Errorf("expected %s to contain a valid gpu id, got: %s", k, v))
	}

	return warnings, errors
}
