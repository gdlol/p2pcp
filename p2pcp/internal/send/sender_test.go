package send

import (
	"archive/tar"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransferHandler is a mock transfer handler that can be registered for the TransferProtocol.
type TestTransferHandler struct {
	handler func(*tar.Header, io.Reader)
	done    func()
}

func (tth *TestTransferHandler) HandleFile(hdr *tar.Header, r io.Reader) { tth.handler(hdr, r) }

func (tth *TestTransferHandler) Done() {
	tth.done()
}

func Test_relativePath(t *testing.T) {
	tests := []struct {
		basePath   string // Path given by user.
		baseIsDir  bool
		targetPath string // Path retrieved from "walking" basePath.
		wantPath   string // Path we want to put in the tar - this is how the files are extracted on the receiving side.
	}{
		{basePath: "file", baseIsDir: false, targetPath: "file", wantPath: "file"},
		{basePath: "a/file", baseIsDir: false, targetPath: "a/file", wantPath: "file"},
		{basePath: "../../file", baseIsDir: false, targetPath: "../../file", wantPath: "file"},
		{basePath: "../a", baseIsDir: true, targetPath: "../a/file", wantPath: "a/file"},
		{basePath: "a/", baseIsDir: true, targetPath: "a/file", wantPath: "a/file"},
		{basePath: "a", baseIsDir: true, targetPath: "a", wantPath: "a"},
		{basePath: "a/b/", baseIsDir: true, targetPath: "a/b/file", wantPath: "b/file"},
		{basePath: "../a/./b/", baseIsDir: true, targetPath: "../a/b/c/file", wantPath: "b/c/file"},
		{basePath: "../a/./b/", baseIsDir: true, targetPath: "../a/b/c/file", wantPath: "b/c/file"},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("base: %s (%v), target: %s -> %s", tt.basePath, tt.baseIsDir, tt.targetPath, tt.wantPath)
		t.Run(name, func(t *testing.T) {
			got, err := relativePath(tt.basePath, tt.baseIsDir, tt.targetPath)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, got)
		})
	}
}
