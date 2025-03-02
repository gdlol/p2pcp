package asserts

import (
	"path/filepath"
	"project"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckFileEqual(t *testing.T) {
	areFilesEqual := func(file1, file2 string) (equal bool) {
		equal = true
		defer func() {
			if r := recover(); r != nil {
				equal = false
			}
		}()
		AssertFilesEqual(file1, file2)
		return equal
	}

	testDataPath := project.GetTestDataPath()
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
			assert.False(t, areFilesEqual(files[i], files[j]),
				"files %s and %s should not be equal", files[i], files[j])
		}
	}

	file7 := filepath.Join(testDataPath, "transfer_file_with_subdir", "file")
	file8 := filepath.Join(testDataPath, "transfer_file_with_subdir", "subdir", "file")
	assert.True(t, areFilesEqual(file7, file8),
		"files %s and %s should be equal", file7, file8)
}
