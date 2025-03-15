package transfer

import (
	"archive/tar"
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

func TestInvalidTars(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), project.Name, "test", "invalid_tar")
	workspace.ResetDir(tempPath)
	outputPath := filepath.Join(tempPath, "output")
	workspace.ResetDir(outputPath)

	fileInfo, err := os.Stat(filepath.Join(workspace.GetProjectPath(), "package.json"))
	require.NoError(t, err)

	// Tar with absolute path
	absPathTar := func() string {
		filePath := filepath.Join(tempPath, "absolute_path.tar")
		file, err := os.Create(filePath)
		require.NoError(t, err)
		defer file.Close()

		writer := tar.NewWriter(file)
		defer writer.Close()

		tarHeader, err := tar.FileInfoHeader(fileInfo, "")
		require.NoError(t, err)
		tarHeader.Name = "/package.json"
		tarHeader.Size = 0
		err = writer.WriteHeader(tarHeader)
		require.NoError(t, err)

		return filePath
	}()

	func() {
		reader, err := os.Open(absPathTar)
		require.NoError(t, err)
		defer reader.Close()

		err = readTar(reader, outputPath)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "absolute path in archive: /package.json")
	}()

	// Tar with invalid path
	invalidPathTar := func() string {
		filePath := filepath.Join(tempPath, "absolute_path.tar")
		file, err := os.Create(filePath)
		require.NoError(t, err)
		defer file.Close()

		writer := tar.NewWriter(file)
		defer writer.Close()

		tarHeader, err := tar.FileInfoHeader(fileInfo, "")
		require.NoError(t, err)
		tarHeader.Name = "../../package.json"
		tarHeader.Size = 0
		err = writer.WriteHeader(tarHeader)
		require.NoError(t, err)

		return filePath
	}()

	func() {
		reader, err := os.Open(invalidPathTar)
		require.NoError(t, err)
		defer reader.Close()

		err = readTar(reader, outputPath)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "invalid path in archive: ../../package.json")
	}()

	// Tar with absolute symlink
	absSymlinkTar := func() string {
		filePath := filepath.Join(tempPath, "absolute_symlink.tar")
		file, err := os.Create(filePath)
		require.NoError(t, err)
		defer file.Close()

		symlinkPath := filepath.Join(tempPath, "abs_symlink")
		os.Symlink("/package.json", filepath.Join(tempPath, "abs_symlink"))
		fileInfo, err := os.Lstat(symlinkPath)
		require.NoError(t, err)

		writer := tar.NewWriter(file)
		defer writer.Close()

		tarHeader, err := tar.FileInfoHeader(fileInfo, "/package.json")
		require.NoError(t, err)
		err = writer.WriteHeader(tarHeader)
		require.NoError(t, err)

		return filePath
	}()

	func() {
		reader, err := os.Open(absSymlinkTar)
		require.NoError(t, err)
		defer reader.Close()

		err = readTar(reader, outputPath)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "absolute symbolic link in archive: abs_symlink -> /package.json")
	}()

	// Tar with invalid symlink
	invalidSymlinkTar := func() string {
		filePath := filepath.Join(tempPath, "absolute_symlink.tar")
		file, err := os.Create(filePath)
		require.NoError(t, err)
		defer file.Close()

		symlinkPath := filepath.Join(tempPath, "invalid_symlink")
		os.Symlink("/package.json", filepath.Join(tempPath, "invalid_symlink"))
		fileInfo, err := os.Lstat(symlinkPath)
		require.NoError(t, err)

		writer := tar.NewWriter(file)
		defer writer.Close()

		tarHeader, err := tar.FileInfoHeader(fileInfo, "../../../package.json")
		require.NoError(t, err)
		err = writer.WriteHeader(tarHeader)
		require.NoError(t, err)

		return filePath
	}()

	func() {
		reader, err := os.Open(invalidSymlinkTar)
		require.NoError(t, err)
		defer reader.Close()

		err = readTar(reader, outputPath)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "invalid symbolic link in archive: invalid_symlink -> ../../../package.json")
	}()

	invalidFileTypeTar := func() string {
		filePath := filepath.Join(tempPath, "invalid_file_type.tar")
		file, err := os.Create(filePath)
		require.NoError(t, err)
		defer file.Close()

		writer := tar.NewWriter(file)
		defer writer.Close()

		tarHeader, err := tar.FileInfoHeader(fileInfo, "")
		require.NoError(t, err)
		tarHeader.Name = "package.json"
		tarHeader.Typeflag |= tar.TypeChar
		err = writer.WriteHeader(tarHeader)
		require.NoError(t, err)

		return filePath
	}()

	func() {
		reader, err := os.Open(invalidFileTypeTar)
		require.NoError(t, err)
		defer reader.Close()

		err = readTar(reader, filepath.Join(tempPath, "output"))
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "unsupported file type for entry package.json")
	}()
}

func TestTarOverwriteLinkWithoutPermission(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), project.Name, "test", "TestTarOverwriteLinkWithoutPermission")
	workspace.Run("sudo", "rm", "-rf", tempPath)
	inputPath := filepath.Join(tempPath, "input")
	workspace.ResetDir(inputPath)
	outputPath := filepath.Join(tempPath, "output")
	workspace.ResetDir(outputPath)

	fileInfo, err := os.Stat(filepath.Join(workspace.GetProjectPath(), "package.json"))
	require.NoError(t, err)

	file, err := os.Create(filepath.Join(outputPath, "file"))
	require.NoError(t, err)
	file.Close()

	err = os.Symlink("file", filepath.Join(outputPath, "link"))
	require.NoError(t, err)

	workspace.Run("chmod", "a-w", outputPath)

	tarPath := func() string {
		filePath := filepath.Join(tempPath, "link.tar")
		file, err := os.Create(filePath)
		require.NoError(t, err)
		defer file.Close()

		writer := tar.NewWriter(file)
		defer writer.Close()

		tarHeader, err := tar.FileInfoHeader(fileInfo, "file")
		require.NoError(t, err)
		tarHeader.Name = "link"
		tarHeader.Size = 0
		tarHeader.Typeflag = tar.TypeSymlink
		err = writer.WriteHeader(tarHeader)
		require.NoError(t, err)

		return filePath
	}()

	reader, err := os.Open(tarPath)
	require.NoError(t, err)
	defer reader.Close()

	err = readTar(reader, filepath.Join(tempPath, "output"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error overwriting")
	assert.Contains(t, err.Error(), filepath.Join(outputPath, "link"))
	assert.Contains(t, err.Error(), "permission denied")
}

func TestTarReadFileWithoutPermission(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), project.Name, "test", "TestTarReadFileWithoutPermission")
	workspace.Run("sudo", "rm", "-rf", tempPath)
	inputPath := filepath.Join(tempPath, "input")
	workspace.ResetDir(inputPath)
	outputPath := filepath.Join(tempPath, "output")
	workspace.ResetDir(outputPath)

	filePath := filepath.Join(inputPath, "file")
	file, err := os.Create(filePath)
	require.NoError(t, err)
	file.Close()

	workspace.Run("chmod", "a-r", filePath)

	tarPath := filepath.Join(tempPath, "test.tar")
	tarFile, err := os.Create(tarPath)
	require.NoError(t, err)
	defer tarFile.Close()

	err = writeTar(tarFile, inputPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error opening file")
	assert.Contains(t, err.Error(), filePath)
}

func TestTarReadDirWithoutPermission(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), project.Name, "test", "TestTarReadDirWithoutPermission")
	workspace.Run("sudo", "rm", "-rf", tempPath)
	inputPath := filepath.Join(tempPath, "input")
	workspace.ResetDir(inputPath)
	outputPath := filepath.Join(tempPath, "output")
	workspace.ResetDir(outputPath)

	dirPath := filepath.Join(inputPath, "dir")
	workspace.ResetDir(dirPath)
	workspace.Run("chmod", "a-r", dirPath)

	tarPath := filepath.Join(tempPath, "test.tar")
	tarFile, err := os.Create(tarPath)
	require.NoError(t, err)
	defer tarFile.Close()

	err = writeTar(tarFile, inputPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error walking path")
	assert.Contains(t, err.Error(), dirPath)
}

func TestTarReadNonExistDir(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), project.Name, "test", "TestTarReadDirWithoutPermission")
	workspace.Run("sudo", "rm", "-rf", tempPath)
	inputPath := filepath.Join(tempPath, "input")
	workspace.ResetDir(inputPath)
	outputPath := filepath.Join(tempPath, "output")
	workspace.ResetDir(outputPath)

	dirPath := filepath.Join(inputPath, "dir")

	tarPath := filepath.Join(tempPath, "test.tar")
	tarFile, err := os.Create(tarPath)
	require.NoError(t, err)
	defer tarFile.Close()

	err = writeTar(tarFile, dirPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
	assert.Contains(t, err.Error(), inputPath)
}

func TestTarWriteFileError(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), project.Name, "test", "TestTarWriteCloseError")
	workspace.Run("sudo", "rm", "-rf", tempPath)
	inputPath := filepath.Join(tempPath, "input")
	workspace.ResetDir(inputPath)
	outputPath := filepath.Join(tempPath, "output")
	workspace.ResetDir(outputPath)

	filePath := filepath.Join(inputPath, "file")
	file, err := os.Create(filePath)
	require.NoError(t, err)
	file.Close()

	reader, writer := io.Pipe()

	done := make(chan struct{})
	go func() {
		err = writeTar(writer, inputPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error writing tar header")
		done <- struct{}{}
	}()

	tarReader := tar.NewReader(reader)
	_, err = tarReader.Next()
	require.NoError(t, err)
	reader.Close()

	<-done
}

func TestTarWriteDirError(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), project.Name, "test", "TestTarWriteCloseError")
	workspace.Run("sudo", "rm", "-rf", tempPath)
	inputPath := filepath.Join(tempPath, "input")
	workspace.ResetDir(inputPath)
	outputPath := filepath.Join(tempPath, "output")
	workspace.ResetDir(outputPath)

	dirPath := filepath.Join(inputPath, "dir")
	workspace.ResetDir(dirPath)

	reader, writer := io.Pipe()

	done := make(chan struct{})
	go func() {
		err := writeTar(writer, inputPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error writing tar header")
		done <- struct{}{}
	}()

	tarReader := tar.NewReader(reader)
	_, err := tarReader.Next()
	require.NoError(t, err)
	reader.Close()

	<-done
}

func TestTarWriteCloseError(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), project.Name, "test", "TestTarWriteCloseError")
	workspace.Run("sudo", "rm", "-rf", tempPath)
	inputPath := filepath.Join(tempPath, "input")
	workspace.ResetDir(inputPath)
	outputPath := filepath.Join(tempPath, "output")
	workspace.ResetDir(outputPath)

	reader, writer := io.Pipe()

	done := make(chan struct{})
	go func() {
		err := writeTar(writer, inputPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error closing tar")
		done <- struct{}{}
	}()

	tarReader := tar.NewReader(reader)
	info, err := tarReader.Next()
	require.NoError(t, err)
	require.Equal(t, "input", info.Name)
	reader.Close()
	<-done
}
