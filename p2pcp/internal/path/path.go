package path

import (
	"os"
	"p2pcp/internal/errors"
	"path/filepath"
)

func GetCurrentDirectory() string {
	path, err := os.Getwd()
	errors.Unexpected(err, "GetCurrentDirectory")
	return path
}

func GetAbsolutePath(path string) string {
	absPath, err := filepath.Abs(path)
	errors.Unexpected(err, "GetAbsolutePath")
	return absPath
}

func GetRelativePath(basePath string, targetPath string) string {
	relPath, err := filepath.Rel(basePath, targetPath)
	errors.Unexpected(err, "GetRelativePath")
	return relPath
}
