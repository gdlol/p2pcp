package path

import (
	"os"
	"path/filepath"
	"project/pkg/workspace"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentDirectory(t *testing.T) {
	testPath := filepath.Join(os.TempDir(), "p2pcp/test/GetCurrentDirectory")
	workspace.ResetDir(testPath)

	restore := workspace.Chdir(testPath)
	defer restore()

	err := os.Remove(testPath)
	require.NoError(t, err)

	func() {
		defer func() {
			err = recover().(error)
		}()
		GetCurrentDirectory()
	}()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestGetAbsolutePath(t *testing.T) {
	testPath := filepath.Join(os.TempDir(), "p2pcp/test/GetAbsolutePath")
	workspace.ResetDir(testPath)

	restore := workspace.Chdir(testPath)
	defer restore()

	err := os.Remove(testPath)
	require.NoError(t, err)

	func() {
		defer func() {
			err = recover().(error)
		}()
		GetAbsolutePath("test")
	}()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestGetRelativePath(t *testing.T) {
	testPath := filepath.Join(os.TempDir(), "p2pcp/test/GetRelativePath")
	workspace.ResetDir(testPath)

	restore := workspace.Chdir(testPath)
	defer restore()

	err := os.Remove(testPath)
	require.NoError(t, err)

	func() {
		defer func() {
			err = recover().(error)
		}()
		GetRelativePath("/test", "test")
	}()
	require.Error(t, err)
}
