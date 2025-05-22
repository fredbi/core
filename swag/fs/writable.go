package fs

/*
import (
	"io/fs"
	"os"

	"github.com/spf13/afero"
)

type Writable interface {
	fs.FS

	// Create creates a file in the filesystem, returning the file and an
	// error, if any happens.
	Create(name string) (afero.File, error)

	// Mkdir creates a directory in the filesystem, return an error if any
	// happens.
	Mkdir(name string, perm os.FileMode) error

	// MkdirAll creates a directory path and all parents that does not exist
	// yet.
	MkdirAll(path string, perm os.FileMode) error

	// OpenFile opens a file using the given flags and the given mode.
	OpenFile(name string, flag int, perm os.FileMode) (afero.File, error)

	// Remove removes a file identified by name, returning an error, if any
	// happens.
	Remove(name string) error

	// RemoveAll removes a directory path and any children it contains. It
	// does not fail if the path does not exist (return nil).
	RemoveAll(path string) error

	// Rename renames a file.
	Rename(oldname, newname string) error

	// Stat returns a FileInfo describing the named file, or an error, if any
	// happens.
	Stat(name string) (os.FileInfo, error)

	// The name of this FileSystem
	Name() string
}

func NewWritableFS(base afero.Fs) *WritableFS {
	return &WritableFS{
		base: base,
	}
}

// WritableFS is a file system that behaves like [fs.FS] for read-only operations
// and like [afero.Fs] for write operations. This implements the [Writable] interface.
type WritableFS struct {
	base afero.Fs
}
*/

/*
// Create creates a file in the filesystem, returning the file and an
// error, if any happens.
func (f *OverlayFS) Create(name string) (WritableFile, error) {
}

// Mkdir creates a directory in the filesystem, return an error if any
// happens.
func (f *OverlayFS) Mkdir(name string, perm FileMode) error {}

// MkdirAll creates a directory path and all parents that does not exist
// yet.
func (f *OverlayFS) MkdirAll(path string, perm FileMode) error {}

// Open opens a file, returning it or an error, if any happens.
func (f *OverlayFS) Open(name string) (WritableFile, error) {}

// OpenWritableFile opens a file using the given flags and the given mode.
func (f *OverlayFS) OpenFile(name string, flag int, perm FileMode) (WritableFile, error) {}

// Remove removes a file identified by name, returning an error, if any
// happens.
func (f *OverlayFS) Remove(name string) error {}

// RemoveAll removes a directory path and any children it contains. It
// does not fail if the path does not exist (return nil).
func (f *OverlayFS) RemoveAll(path string) error {}

// Rename renames a file.
func (f *OverlayFS) Rename(oldname, newname string) error {}

// Stat returns a FileInfo describing the named file, or an error, if any
// happens.
func (f *OverlayFS) Stat(name string) (FileInfo, error) {}

// The name of this FileSystem
func (f *OverlayFS) Name() string {}
*/
