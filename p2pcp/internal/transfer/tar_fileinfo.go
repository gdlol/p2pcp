package transfer

import (
	"io/fs"
	"os"
)

// Wraps os.FileInfo and drops unsupported attributes for tar transfer.
type tarFileInfo struct {
	os.FileInfo
}

var allowedMode fs.FileMode = fs.ModeDir | fs.ModeSymlink | fs.ModePerm

func (t *tarFileInfo) Mode() fs.FileMode {
	return t.FileInfo.Mode() & allowedMode
}

func (t *tarFileInfo) Sys() any {
	return nil
}

func getTarFileInfo(info os.FileInfo) os.FileInfo {
	return &tarFileInfo{FileInfo: info}
}
