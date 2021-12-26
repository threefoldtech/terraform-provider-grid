package internal

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/pkg/errors"
)

/* 	usage:
 * 	func test() (uint32, error) {
 *   	t = NewTerror()
 *		if err := another(); err != nil {
 *			t = t.AppendErr(err)
 *			return 0, t
 *		}
 *		if err := anotherDiags(); err.HasErrors() {
 *			t = t.Append(err)
 *			return 0, t
 *		}
 *		for i := 1; i < 5; i++ {
 *			if err := check(i); err != nil {
 *				t = t.AppendErr(err)
 *			}
 *		}
 *		return 0, t.AsNullIfEmpty()
 *	}
 */

type Terror struct {
	diag.Diagnostics
}

func severityString(s diag.Severity) string {
	if s == diag.Error {
		return "Error"
	} else {
		return "Warn"
	}
}
func NewTerror() *Terror {
	return &Terror{Diagnostics: make(diag.Diagnostics, 0)}
}
func encodeDiagnostic(d diag.Diagnostic) string {
	if d.Detail == "" {
		return fmt.Sprintf("%s: %s", severityString(d.Severity), d.Summary)
	} else {
		return fmt.Sprintf("%s: %s\nDetails: %s", severityString(d.Severity), d.Summary, d.Detail)
	}
}
func (t *Terror) HasError() bool {
	return t.Diagnostics.HasError()
}
func (t *Terror) Append(d diag.Diagnostic) {
	if t == nil {
		return
	}
	t.Diagnostics = append(t.Diagnostics, d)
}

func (t *Terror) AppendErr(err error) {
	if t == nil || err == nil {
		return
	}
	if e, ok := err.(*Terror); ok {
		t.Diagnostics = append(t.Diagnostics, e.Diagnostics...)
	} else {
		t.Diagnostics = append(t.Diagnostics, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
	}
}

func (t *Terror) AppendWrappedErr(err error, nerr string, args ...interface{}) {
	if t == nil || err == nil {
		return
	}
	if e, ok := err.(*Terror); ok {
		for _, d := range e.Diagnostics {
			t.Diagnostics = append(t.Diagnostics, diag.Diagnostic{
				Severity: d.Severity,
				Summary:  fmt.Sprintf("%s: %s", fmt.Sprintf(nerr, args...), d.Summary),
				Detail:   d.Detail,
			})
		}
	} else {
		t.Diagnostics = append(t.Diagnostics, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  errors.Wrapf(err, nerr, args...).Error(),
		})
	}
}

func (t *Terror) Error() string {
	if t == nil || len(t.Diagnostics) == 0 {
		return "nothing bad happened, if you are presented with error, it's probably a plugin bug"
	} else if len(t.Diagnostics) == 1 {
		return encodeDiagnostic(t.Diagnostics[0])
	} else {
		result := "The following diagnostics occured:"
		for _, d := range t.Diagnostics {
			result = fmt.Sprintf("%s\n\n%s", result, encodeDiagnostic(d))
		}
		return result
	}
}

func (t *Terror) Diags() diag.Diagnostics {
	if t == nil {
		return nil
	}
	return t.Diagnostics
}

func (t *Terror) AsError() error {
	if t == nil || len(t.Diagnostics) == 0 {
		return nil
	}
	return t
}
