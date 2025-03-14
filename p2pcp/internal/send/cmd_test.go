package send

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCancelSend(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := Send(ctx, "", false, false)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}
