package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanics(t *testing.T) {
	Unexpected(nil, "test")

	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.Equal(t, "unexpected error: context: test", r.(error).Error())
	}()
	Unexpected(fmt.Errorf("test"), "context")
}

func TestHandleUnexpectedError(t *testing.T) {
	var err = RecoverUnexpected(func() error {
		Unexpected(fmt.Errorf("test"), "context")
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "unexpected error: context: test", err.Error())

	err = func() (err error) {
		defer func() {
			if recover() == nil {
				err = fmt.Errorf("recovered")
			}
		}()
		return RecoverUnexpected(func() error {
			panic("test")
		})
	}()
	assert.NoError(t, err)
}

func TestAssert(t *testing.T) {
	Assert(true, "test")

	var message string
	defer func() {
		r := recover()
		assert.NotNil(t, r)
		message = r.(error).Error()
	}()
	Assert(false, "test")
	assert.Equal(t, "unexpected error: assertion failed: test", message)
}
