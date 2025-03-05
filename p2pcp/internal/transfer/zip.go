package transfer

import (
	"compress/gzip"
	"io"
)

func ReadZip(r io.Reader, basePath string) error {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer reader.Close()

	return readTar(reader, basePath)
}

func WriteZip(w io.Writer, basePath string) error {
	writer := gzip.NewWriter(w)
	defer writer.Close()

	return writeTar(writer, basePath)
}
