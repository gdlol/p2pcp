package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBootstrapPeers(t *testing.T) {
	peers, err := getBootstrapPeers()
	assert.NoError(t, err)
	assert.NotEmpty(t, peers)
}
