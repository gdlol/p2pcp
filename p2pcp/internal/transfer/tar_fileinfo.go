package transfer

import (
	"io/fs"
	"os"
	"time"
)

// Wraps os.FileInfo and drops unsupported attributes for tar transfer.
type tarFileInfo struct {
	info os.FileInfo
}

var allowedMode fs.FileMode = fs.ModeDir | fs.ModeSymlink | fs.ModePerm

func (t *tarFileInfo) IsDir() bool {
	return t.info.IsDir()
}

func (t *tarFileInfo) ModTime() time.Time {
	return t.info.ModTime()
}

func (t *tarFileInfo) Mode() fs.FileMode {
	return t.info.Mode() & allowedMode
}

func (t *tarFileInfo) Name() string {
	return t.info.Name()
}

func (t *tarFileInfo) Size() int64 {
	return t.info.Size()
}

func (t *tarFileInfo) Sys() any {
	return nil
}

func getTarFileInfo(info os.FileInfo) os.FileInfo {
	return &tarFileInfo{info: info}
}
