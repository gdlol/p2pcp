package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBootstrapPeers(t *testing.T) {
	peers := getBootstrapPeers()
	assert.NotEmpty(t, peers)
}
