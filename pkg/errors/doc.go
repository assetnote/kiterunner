/*
The errors package provides a custom error type and utilities used when performing recursive
analysis of kitebuilder and proute apis.

The error type will provide context about what depth and what specific component
of the API yielded the error, and the printing utilities assist in graphically
representing the errors with corresponding nested depth.

Usage

	import errors2 "github.com/assetnote/kiterunner/pkg/errors"

	...

	if err := inAPI.EncodeStringSlice(output); err != nil {
		var merr *multierror.Error
		if errors.As(err, &merr) {
			for _, v := range merr.Errors {
				errors2.PrintError(v, 0)
			}
		} else {
			return fmt.Errorf("converting to txt output error: %w", err)
		}
	}

 */
package errors
