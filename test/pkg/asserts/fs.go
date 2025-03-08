package asserts

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/blake2b"
)

func AssertFilesEqual(file1, file2 string) {
	info1, err := os.Stat(file1)
	checkEqual(nil, err)

	info2, err := os.Stat(file2)
	checkEqual(nil, err)

	checkEqual(false, info1.IsDir())
	checkEqual(false, info2.IsDir())
	checkEqual(info1.Name(), info2.Name())
	checkEqual(info1.Size(), info2.Size())
	checkEqual(info1.Mode(), info2.Mode())

	f1, err := os.Open(file1)
	checkEqual(nil, err)
	defer f1.Close()

	f2, err := os.Open(file2)
	checkEqual(nil, err)
	defer f2.Close()

	hash1, err := blake2b.New256(nil)
	checkEqual(nil, err)
	hash2, err := blake2b.New256(nil)
	checkEqual(nil, err)
	buffer := make([]byte, 1024)
	for {
		n, err := f1.Read(buffer)
		if err == io.EOF {
			break
		}
		checkEqual(nil, err)
		_, err = hash1.Write(buffer[:n])
		checkEqual(nil, err)
	}
	for {
		n, err := f2.Read(buffer)
		if err == io.EOF {
			break
		}
		checkEqual(nil, err)
		_, err = hash2.Write(buffer[:n])
		checkEqual(nil, err)
	}
	checkEqual(hex.Dump(hash1.Sum(nil)), hex.Dump(hash2.Sum(nil)))
}

func AssertDirsEqual(dir1, dir2 string) {
	info1, err := os.Stat(dir1)
	checkEqual(nil, err)

	info2, err := os.Stat(dir1)
	checkEqual(nil, err)

	checkEqual(true, info1.IsDir())
	checkEqual(true, info2.IsDir())

	type walkData struct {
		path string
		info os.FileInfo
		err  error
	}

	var walkData1 []walkData
	var walkData2 []walkData

	err = filepath.Walk(dir1, func(path string, info os.FileInfo, err error) error {
		walkData1 = append(walkData1, walkData{path, info, err})
		return err
	})
	checkEqual(nil, err)

	err = filepath.Walk(dir2, func(path string, info os.FileInfo, err error) error {
		walkData2 = append(walkData2, walkData{path, info, err})
		return err
	})
	checkEqual(nil, err)

	checkEqual(len(walkData1), len(walkData2))

	for i := range walkData1 {
		walk1 := walkData1[i]
		walk2 := walkData2[i]

		checkEqual(nil, walk1.err)
		checkEqual(nil, walk2.err)

		rel1, err := filepath.Rel(dir1, walk1.path)
		checkEqual(nil, err)

		rel2, err := filepath.Rel(dir2, walk2.path)
		checkEqual(nil, err)

		checkEqual(rel1, rel2)
		checkEqual(walk1.info.IsDir(), walk2.info.IsDir())
		if !walk1.info.IsDir() {
			AssertFilesEqual(walk1.path, walk2.path)
		} else {
			checkEqual(walk1.info.Mode(), walk2.info.Mode())
		}
	}
}

func AreFilesEqual(file1, file2 string) (equal bool) {
	equal = true
	defer func() {
		if r := recover(); r != nil {
			equal = false
		}
	}()
	AssertFilesEqual(file1, file2)
	return equal
}

func AreDirsEqual(dir1, dir2 string) (equal bool) {
	equal = true
	defer func() {
		if r := recover(); r != nil {
			equal = false
		}
	}()
	AssertDirsEqual(dir1, dir2)
	return equal
}
