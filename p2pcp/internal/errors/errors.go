package errors

import "fmt"

type UnexpectedError struct {
	Err     error
	Context string
}

func (u UnexpectedError) Error() string {
	return fmt.Sprintf("unexpected error: %s: %s", u.Context, u.Err.Error())
}

var _ error = UnexpectedError{}

func Unexpected(err error, context string) {
	if err != nil {
		panic(UnexpectedError{Err: err, Context: context})
	}
}

func RecoverUnexpected(action func() error) error {
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if u, ok := r.(UnexpectedError); ok {
					err = u
				} else {
					panic(r)
				}
			}
		}()
		err = action()
	}()
	return err
}

func Assert(condition bool, message string) {
	if !condition {
		panic(UnexpectedError{Err: fmt.Errorf("%s", message), Context: "assertion failed"})
	}
}
