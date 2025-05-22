package fs

import (
	"io/fs"
	"os"
)

// OsFS exposes package os features as an [fs.FS], without having to use [os.Root].
//
// [osFS] implements [fs.FS], [fs.ReadFileFS] and [fs.ReadDirFS].
type OsFS struct {
}

// NewReadOnlyOsFS builds a [OsFS], which implements [fs.FS] (read-only)
// from the standard os file system.
func NewReadOnlyOsFS() *OsFS {
	return &OsFS{}
}

func (f *OsFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (f *OsFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (f *OsFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// FileReaderFS makes a [fs.FS] into a [fs.ReadFileFS], with a [FileReaderFS.ReadFile] method.
type FileReaderFS struct {
	fs.FS
}

// NewFileReaderFS transforms a [fs.FS] into a [fs.ReadFileFS].
func NewFileReaderFS(base fs.FS) *FileReaderFS {
	return &FileReaderFS{
		FS: base,
	}
}

func (f *FileReaderFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(f.FS, name)
}

// GlobOsFS is like [OsFS]. It additionally implements [fs.Glob], with a [GlobOsFS.Glob] method.
type GlobOsFS struct {
	*OsFS
}

// NewGlobOsFS is like NewReadOnlyOsFS, augmented to match the [fs.GlobFS] interface.
func NewGlobOsFS() *GlobOsFS {
	return &GlobOsFS{
		OsFS: NewReadOnlyOsFS(),
	}
}

func (f *GlobOsFS) Glob(pattern string) ([]string, error) {
	return fs.Glob(f.OsFS, pattern)
}
