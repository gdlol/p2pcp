package receive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	progress "github.com/schollz/progressbar/v3"
)

func handleFile(header *tar.Header, reader io.Reader, basePath string) error {
	fileInfo := header.FileInfo()
	joined := filepath.Join(basePath, header.Name)
	if fileInfo.IsDir() {
		err := os.MkdirAll(joined, fileInfo.Mode())
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", joined, err)
		}
		return nil
	}

	newFile, err := os.OpenFile(joined, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", joined, err)
	}

	bar := progress.DefaultBytes(header.Size, filepath.Base(header.Name))
	_, err = io.Copy(io.MultiWriter(newFile, bar), reader)
	if err != nil {
		return fmt.Errorf("error writing file content %s: %w", joined, err)
	}
	return nil
}

func readTar(r io.Reader, basePath string) error {
	reader := tar.NewReader(r)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return fmt.Errorf("error reading next tar element: %w", err)
		}
		err = handleFile(header, reader, basePath)
		if err != nil {
			return err
		}
	}

	// Drain padding
	buffer := make([]byte, 8192)
	_, err := r.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}
