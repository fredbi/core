package repo

import (
	"io/fs"
	"os"
)

// osfs exposes package os features as an fs.FS, without having to use [os.Root].
type osfs struct {
}

func (f *osfs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (f *osfs) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (f *osfs) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// readfilefs makes a [fs.FS] into a [fs.ReadFileFS]
type readfilefs struct {
	fs.FS
}

func (f *readfilefs) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(f.FS, name)
}

func optionsWithDefaults(opts []Option) options {
	o := defaultOptions

	for _, apply := range opts {
		apply(&o)
	}

	return o
}
