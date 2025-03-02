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
