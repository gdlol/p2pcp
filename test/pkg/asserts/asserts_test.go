package asserts

import (
	"path/filepath"
	"project/pkg/workspace"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckFilesEqual(t *testing.T) {
	testDataPath := workspace.GetTestDataPath()
	file1 := filepath.Join(testDataPath, "transfer_dir_multiple_file", "file1")
	file2 := filepath.Join(testDataPath, "transfer_dir_multiple_file", "file2")
	file3 := filepath.Join(testDataPath, "transfer_dir_multiple_file", ".dot_file")
	file4 := filepath.Join(testDataPath, "transfer_dir", "file")
	file5 := filepath.Join(testDataPath, "transfer_file", "file")
	file6 := filepath.Join(testDataPath, "non_exist_dir", "file")
	files := []string{file1, file2, file3, file4, file5, file6}

	for i := range files {
		for j := range files {
			if i == j {
				continue
			}
			assert.False(t, AreFilesEqual(files[i], files[j]),
				"files %s and %s should not be equal", files[i], files[j])
		}
	}

	file7 := filepath.Join(testDataPath, "transfer_file_with_subdir", "file")
	file8 := filepath.Join(testDataPath, "transfer_file_with_subdir", "subdir", "file")
	assert.True(t, AreFilesEqual(file7, file8),
		"files %s and %s should be equal", file7, file8)
}

func TestCheckDirsEqual(t *testing.T) {
	testDataPath := workspace.GetTestDataPath()
	dir1 := filepath.Join(testDataPath, "transfer_dir")
	dir2 := filepath.Join(testDataPath, "transfer_dir_multiple_file")
	dir3 := filepath.Join(testDataPath, "transfer_file")
	dir4 := filepath.Join(testDataPath, "non_exist_dir")
	dirs := []string{dir1, dir2, dir3, dir4}

	for i := range dirs {
		for j := range dirs {
			if i == j {
				continue
			}
			assert.False(t, AreDirsEqual(dirs[i], dirs[j]),
				"dirs %s and %s should not be equal", dirs[i], dirs[j])
		}
	}

	dir6 := filepath.Join(testDataPath, "transfer_link_2/expected/dir")
	dir7 := filepath.Join(testDataPath, "transfer_unix_socket/expected/dir")
	assert.True(t, AreDirsEqual(dir6, dir7),
		"dirs %s and %s should be equal", dir6, dir7)
}
