package workspace

import (
	"os"
	"sync"
)

var chdirLock sync.Mutex

func Chdir(path string) func() {
	if !chdirLock.TryLock() {
		panic("chdir: already locked")
	}
	var original string
	var err error
	defer func() {
		if err != nil {
			chdirLock.Unlock()
		}
	}()

	original, err = os.Getwd()
	Check(err)
	err = os.Chdir(path)
	Check(err)
	return func() {
		defer chdirLock.Unlock()
		err := os.Chdir(original)
		Check(err)
	}
}
