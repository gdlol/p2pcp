package transfer

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertFileEqual(t *testing.T, file1, file2 string) {
	info1, err := os.Stat(file1)
	require.NoError(t, err)

	info2, err := os.Stat(file2)
	require.NoError(t, err)

	require.Equal(t, info1.Name(), info2.Name())
	require.Equal(t, info1.Size(), info2.Size())
	require.Equal(t, info1.Mode(), info2.Mode())

	f1, err := os.Open(file1)
	require.NoError(t, err)
	defer f1.Close()

	f2, err := os.Open(file2)
	require.NoError(t, err)
	defer f2.Close()

	buf1 := make([]byte, 1024)
	buf2 := make([]byte, 1024)

	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)

		require.Equal(t, n1, n2)
		require.Equal(t, err1, err2)
		if err1 != nil {
			break
		}

		assert.ElementsMatch(t, buf1[:n1], buf2[:n2])
	}
}

func assertDirEqual(t *testing.T, dir1, dir2 string) {
	type walkData struct {
		path string
		info os.FileInfo
		err  error
	}

	var walkData1 []walkData
	var walkData2 []walkData

	err := filepath.Walk(dir1, func(path string, info os.FileInfo, err error) error {
		walkData1 = append(walkData1, walkData{path, info, err})
		return err
	})
	require.NoError(t, err)

	err = filepath.Walk(dir2, func(path string, info os.FileInfo, err error) error {
		walkData2 = append(walkData2, walkData{path, info, err})
		return err
	})
	require.NoError(t, err)

	require.Equal(t, len(walkData1), len(walkData2))

	for i := range walkData1 {
		walk1 := walkData1[i]
		walk2 := walkData2[i]

		require.NoError(t, walk1.err)
		require.NoError(t, walk2.err)

		rel1, err := filepath.Rel(dir1, walk1.path)
		require.NoError(t, err)

		rel2, err := filepath.Rel(dir2, walk2.path)
		require.NoError(t, err)

		assert.Equal(t, rel1, rel2)
		assert.Equal(t, walk1.info.IsDir(), walk2.info.IsDir())
		if !walk1.info.IsDir() {
			assertFileEqual(t, walk1.path, walk2.path)
		} else {
			assert.Equal(t, walk1.info.Mode(), walk2.info.Mode())
		}
	}
}

func resetDir(t *testing.T, path string) {
	err := os.RemoveAll(path)
	require.NoError(t, err)
	err = os.MkdirAll(path, 0775)
	require.NoError(t, err)
}

func getTestDataPath(t *testing.T) string {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "testdata")
}

/**
 * For each test case, send the sendPath under testDir to a tmp directory,
 * and compare it with the expectedPath under testDir.
 */
func TestTarReadWrite(t *testing.T) {
	testDataPath := getTestDataPath(t)

	// Create empty test directories as they are't committed to git.
	resetDir(t, filepath.Join(testDataPath, "transfer_empty_dir"))
	resetDir(t, filepath.Join(testDataPath, "transfer_empty_dir_with_subdir", "subdir"))

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
	}

	for _, tt := range tests {
		t.Run(tt.testDir, func(t *testing.T) {
			testPath := filepath.Join(testDataPath, tt.testDir)
			sendPath := filepath.Join(testPath, tt.sendPath)
			expectedPath := filepath.Join(testPath, tt.expectedPath)
			targetPath := filepath.Join(os.TempDir(), "p2pcp", "test", tt.testDir)
			resetDir(t, targetPath)

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
				readErr = ReadTar(pipeReader, targetPath)
			}()

			go func() {
				defer wg.Done()
				defer pipeWriter.Close()
				writeErr = WriteTar(pipeWriter, sendPath)
			}()

			wg.Wait()
			require.NoError(t, readErr)
			require.NoError(t, writeErr)
			if info.IsDir() {
				assertDirEqual(t,
					filepath.Join(targetPath, filepath.Base(expectedPath)),
					expectedPath,
				)
			} else {
				assertFileEqual(t,
					filepath.Join(targetPath, filepath.Base(expectedPath)),
					expectedPath,
				)
			}
		})
	}
}
