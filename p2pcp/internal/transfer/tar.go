package transfer

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	progress "github.com/schollz/progressbar/v3"
)

// spell-checker: ignore Typeflag

func isInBasePath(basePath, targetPath string) bool {
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}
	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	path := targetPath
	for basePath != path {
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	return basePath == path
}

func readDir(header *tar.Header, path string) error {
	fileInfo := header.FileInfo()
	err := os.MkdirAll(path, fileInfo.Mode())
	if err != nil {
		return fmt.Errorf("error creating directory %s: %w", path, err)
	}
	return nil
}

func readFile(header *tar.Header, reader io.Reader, path string) error {
	fileInfo := header.FileInfo()

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", path, err)
	}
	defer file.Close()

	bar := progress.DefaultBytes(header.Size, filepath.Base(header.Name))
	defer bar.Close()

	_, err = io.Copy(io.MultiWriter(file, bar), reader)
	if err != nil {
		return fmt.Errorf("error writing file content for %s: %w", path, err)
	}

	return nil
}

func ReadTar(r io.Reader, basePath string) error {
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("error getting absolute path for %s: %w", basePath, err)
	}

	symlinks := make(map[string]string)
	reader := tar.NewReader(r)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return fmt.Errorf("error reading next tar header: %w", err)
		}

		// Validate path of entry.
		if filepath.IsAbs(header.Name) {
			return fmt.Errorf("absolute path in archive: %s", header.Name)
		}
		path := filepath.Clean(filepath.Join(basePath, header.Name))
		if !isInBasePath(basePath, path) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		// Handle symbolic links.
		if header.Typeflag&tar.TypeSymlink == tar.TypeSymlink {
			linkName := filepath.Clean(header.Linkname)
			if filepath.IsAbs(linkName) {
				return fmt.Errorf("absolute symbolic link in archive: %s -> %s", header.Name, header.Linkname)
			}
			targetPath := filepath.Clean(filepath.Join(filepath.Dir(path), header.Linkname))
			if !isInBasePath(basePath, targetPath) {
				return fmt.Errorf("invalid symbolic link in archive: %s -> %s", header.Name, header.Linkname)
			}
			symlinks[path] = linkName
			continue
		}

		// Handle directories.
		if header.Typeflag&tar.TypeDir == tar.TypeDir {
			err = readDir(header, path)
			if err != nil {
				return err
			}
			continue
		}

		// Handle regular files.
		if header.Typeflag&tar.TypeReg == tar.TypeReg {
			err = readFile(header, reader, path)
			if err != nil {
				return err
			}
			continue
		}

		return fmt.Errorf("unsupported file type for entry %s", header.Name)
	}

	// Create symbolic links
	for linkPath, linkName := range symlinks {
		if info, err := os.Lstat(linkPath); err == nil {
			if !info.IsDir() || info.Mode()&fs.ModeSymlink == fs.ModeSymlink {
				err = os.Remove(linkPath)
				if err != nil {
					return fmt.Errorf("error overwriting %s: %w", linkPath, err)
				}
			}
		}
		err = os.Symlink(linkName, linkPath)
		if err != nil {
			return fmt.Errorf("error creating symbolic link %s -> %s: %w", linkPath, linkName, err)
		}
	}

	// Drain padding
	buffer := make([]byte, 512)
	for {
		_, err = r.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func writeTarHeader(header *tar.Header, writer *tar.Writer) error {
	if err := writer.WriteHeader(header); err != nil {
		return fmt.Errorf("error writing tar header: %w", err)
	}
	return nil
}

func writeFile(header *tar.Header, writer *tar.Writer, path string) error {
	if err := writeTarHeader(header, writer); err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %w", path, err)
	}
	defer file.Close()

	bar := progress.DefaultBytes(header.Size, filepath.Base(header.Name))
	defer bar.Close()

	_, err = io.Copy(io.MultiWriter(writer, bar), file)
	if err != nil {
		return err
	}

	return nil
}

func WriteTar(w io.Writer, basePath string) error {
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("error getting absolute path for %s: %w", basePath, err)
	}
	rootInfo, err := os.Lstat(basePath)
	if err != nil {
		return err
	}

	writer := tar.NewWriter(w)
	if !rootInfo.IsDir() { // Single file
		if !rootInfo.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type: %s", basePath)
		}
		header, err := tar.FileInfoHeader(getTarFileInfo(rootInfo), "")
		if err != nil {
			return fmt.Errorf("error creating tar header: %w", err)
		}
		header.Name = rootInfo.Name()
		err = writeFile(header, writer, basePath)
		if err != nil {
			return err
		}
	} else { // Directory
		err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error walking path %s: %w", path, err)
			}

			link := ""
			if info.Mode()&fs.ModeSymlink == fs.ModeSymlink { // Handle symbolic links
				destination, err := os.Readlink(path)
				if err != nil {
					return fmt.Errorf("error reading symbolic link %s: %w", path, err)
				}
				destination = filepath.Clean(destination)
				if !filepath.IsAbs(destination) {
					destination = filepath.Join(filepath.Dir(path), destination)
				}
				if !isInBasePath(basePath, destination) {
					return nil // Skip symbolic links targeting outside of the base path.
				}
				link, err = filepath.Rel(filepath.Dir(path), destination) // All links become relative.
				if err != nil {
					return fmt.Errorf(
						"error getting relative path for symbolic link %s -> %s: %w",
						path, destination, err)
				}
			}
			header, err := tar.FileInfoHeader(getTarFileInfo(info), link)
			if err != nil {
				return fmt.Errorf("error creating tar header: %s: %w", path, err)
			}

			// Sets relative entry path to header, all paths are prefixed with the base directory name.
			name, err := filepath.Rel(basePath, path)
			if err != nil {
				return fmt.Errorf("error getting relative path for %s: %w", path, err)
			}
			name = filepath.Join(rootInfo.Name(), name)
			name = filepath.ToSlash(name)
			header.Name = name

			if info.Mode().IsRegular() {
				return writeFile(header, writer, path)
			} else if info.IsDir() || info.Mode()&fs.ModeSymlink == fs.ModeSymlink {
				return writeTarHeader(header, writer)
			} else {
				return nil // Skip unsupported file types.
			}
		})
		if err != nil {
			return err
		}
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("error closing tar ball: %w", err)
	}

	return nil
}
