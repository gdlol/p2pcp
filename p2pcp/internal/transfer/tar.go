package transfer

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"p2pcp/internal/errors"
	Path "p2pcp/internal/path"
	"path/filepath"

	progress "github.com/schollz/progressbar/v3"
)

// spell-checker: ignore Typeflag

func isInBasePath(basePath, targetPath string) bool {
	basePath = Path.GetAbsolutePath(basePath)
	targetPath = Path.GetAbsolutePath(targetPath)

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

func readTar(r io.Reader, basePath string) error {
	basePath = Path.GetAbsolutePath(basePath)

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
		if header.Typeflag == tar.TypeSymlink {
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
		if header.Typeflag == tar.TypeDir {
			err = readDir(header, path)
			if err != nil {
				return err
			}
			continue
		}

		// Handle regular files.
		if header.Typeflag == tar.TypeReg {
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
		err := os.Symlink(linkName, linkPath)
		if err != nil {
			slog.Warn(fmt.Sprintf("error creating symbolic link %s -> %s: %v", linkPath, linkName, err))
		}
	}

	// Drain padding
	buffer := make([]byte, 512)
	for {
		_, err := r.Read(buffer)
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

func writeTar(w io.Writer, basePath string) error {
	basePath = Path.GetAbsolutePath(basePath)
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
		errors.Unexpected(err, fmt.Sprintf("error getting file info header for %s", basePath))
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
				link = Path.GetRelativePath(filepath.Dir(path), destination) // All links become relative.
			}
			header, err := tar.FileInfoHeader(getTarFileInfo(info), link)
			errors.Unexpected(err, fmt.Sprintf("error getting file info header for %s", path))

			// Sets relative entry path to header, all paths are prefixed with the base directory name.
			name := Path.GetRelativePath(basePath, path)
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
		return fmt.Errorf("error closing tar: %w", err)
	}

	return nil
}
