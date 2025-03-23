package workspace

import (
	"os"
)

func ResetDir(path string) {
	err := os.RemoveAll(path)
	Check(err)
	err = os.MkdirAll(path, 0775)
	Check(err)
}

func ListDir(path string) []string {
	entries, err := os.ReadDir(path)
	Check(err)
	var names []string
	for _, file := range entries {
		names = append(names, file.Name())
	}
	return names
}
