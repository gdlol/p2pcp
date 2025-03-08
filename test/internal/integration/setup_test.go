package integration

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	cleanup()
	code := m.Run()
	os.Exit(code)
}
