package transfer

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"
	"sync"
	"test/pkg/asserts"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsInBasePath(t *testing.T) {
	tests := []struct {
		basePath   string
		targetPath string
		expected   bool
	}{
		{"/base", "/base/dir/file", true},
		{"/base", "/base/dir/../file", true},
		{"/base", "/base/../file", false},
		{"/base", "/base/dir/./file", true},
		{"/base", "/other/dir/file", false},
		{"/base", "/base", true},
		{"/base", "/base/dir/../../file", false},
		{".", "file", true},
		{".", "dir/file", true},
		{".", "dir/../file", true},
		{".", "../file", false},
		{".", "dir/./file", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s, %s", tt.basePath, tt.targetPath), func(t *testing.T) {
			result := isInBasePath(tt.basePath, tt.targetPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type readFunc func(r io.Reader, basePath string) error
type writeFunc func(w io.Writer, basePath string) error

func testReadWrite(t *testing.T, read readFunc, write writeFunc) {
	testDataPath := workspace.GetTestDataPath()

	// Create empty test directories as they are't committed to git.
	workspace.ResetDir(filepath.Join(testDataPath, "transfer_empty_dir"))
	workspace.ResetDir(filepath.Join(testDataPath, "transfer_empty_dir_with_subdir", "subdir"))

	// Create a unix socket for testing, expected to be ignored in transfer.
	socketPath := filepath.Join(testDataPath, "transfer_unix_socket/send/dir/socket")
	os.Remove(socketPath)
	assert.True(t, asserts.AreDirsEqual(
		filepath.Join(testDataPath, "transfer_unix_socket/send/dir"),
		filepath.Join(testDataPath, "transfer_unix_socket/expected/dir"),
	))
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()
	assert.False(t, asserts.AreDirsEqual(
		filepath.Join(testDataPath, "transfer_unix_socket/send/dir"),
		filepath.Join(testDataPath, "transfer_unix_socket/expected/dir"),
	))

	tests := []struct {
		testDir      string // relative path to testdata/
		sendPath     string // relative path to testDir to send
		expectedPath string // relative path to testDir to compare with received path
	}{
		{testDir: "transfer_dir", sendPath: ".", expectedPath: "."},
		{testDir: "transfer_dir_multiple_file", sendPath: ".", expectedPath: "."},
		{testDir: "transfer_empty_dir", sendPath: ".", expectedPath: "."},
		{testDir: "transfer_empty_dir_with_subdir", sendPath: ".", expectedPath: "."},
		{testDir: "transfer_file", sendPath: "file", expectedPath: "file"},
		{testDir: "transfer_file_with_subdir", sendPath: ".", expectedPath: "."},
		{testDir: "transfer_link", sendPath: ".", expectedPath: "."},
		{testDir: "transfer_link_2", sendPath: "send/dir", expectedPath: "expected/dir"},
		{testDir: "transfer_unix_socket", sendPath: "send/dir", expectedPath: "expected/dir"},
	}

	for _, tt := range tests {
		t.Run(tt.testDir, func(t *testing.T) {
			t.Parallel()

			testPath := filepath.Join(testDataPath, tt.testDir)
			sendPath := filepath.Join(testPath, tt.sendPath)
			expectedPath := filepath.Join(testPath, tt.expectedPath)
			targetPath := filepath.Join(os.TempDir(), project.Name, "test", tt.testDir)
			workspace.ResetDir(targetPath)

			info, err := os.Stat(sendPath)
			require.NoError(t, err)

			pipeReader, pipeWriter := io.Pipe()

			var wg sync.WaitGroup
			wg.Add(2)

			var readErr error
			var writeErr error

			go func() {
				defer wg.Done()
				defer pipeReader.Close()
				readErr = read(pipeReader, targetPath)
			}()

			go func() {
				defer wg.Done()
				defer pipeWriter.Close()
				writeErr = write(pipeWriter, sendPath)
			}()

			wg.Wait()
			require.NoError(t, readErr)
			require.NoError(t, writeErr)
			if info.IsDir() {
				asserts.AssertDirsEqual(
					filepath.Join(targetPath, filepath.Base(expectedPath)),
					expectedPath,
				)
			} else {
				asserts.AssertFilesEqual(
					filepath.Join(targetPath, filepath.Base(expectedPath)),
					expectedPath,
				)
			}
		})
	}
}

/**
 * For each test case, send the sendPath under testDir to a tmp directory,
 * and compare it with the expectedPath under testDir.
 */
func TestTarReadWrite(t *testing.T) {
	testReadWrite(t, readTar, writeTar)
}
