package transfer

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	progress "github.com/schollz/progressbar/v3"
)

func readFile(header *tar.Header, reader io.Reader, basePath string) error {
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

// Builds the path structure for the tar archive - this will be the structure as it is received.
func relativePath(basePath string, baseIsDir bool, targetPath string) (string, error) {
	if baseIsDir {
		rel, err := filepath.Rel(basePath, targetPath)
		if err != nil {
			return "", err
		}
		return filepath.Clean(filepath.Join(filepath.Base(basePath), rel)), nil
	} else {
		return filepath.Base(basePath), nil
	}
}

func ReadTar(r io.Reader, basePath string) error {
	reader := tar.NewReader(r)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return fmt.Errorf("error reading next tar element: %w", err)
		}
		err = readFile(header, reader, basePath)
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

func WriteTar(w io.Writer, root string) error {
	rootInfo, err := os.Stat(root)
	if err != nil {
		return err
	}

	writer := tar.NewWriter(w)
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking path %s: %w", path, err)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("error writing tar file info header: %s: %w", path, err)
		}

		// To preserve directory structure in the tar ball.
		header.Name, err = relativePath(root, rootInfo.IsDir(), path)
		if err != nil {
			return fmt.Errorf("error building relative path: %s (IsDir: %v) %s: %w", root, rootInfo.IsDir(), path, err)
		}

		if err = writer.WriteHeader(header); err != nil {
			return fmt.Errorf("error writing tar header: %w", err)
		}

		// Continue as all information was written above with WriteHeader.
		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("error opening file for taring at: %s: %w", path, err)
		}
		defer f.Close()

		bar := progress.DefaultBytes(info.Size(), info.Name())
		if _, err = io.Copy(io.MultiWriter(writer, bar), f); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("error closing tar ball: %w", err)
	}

	return nil
}
