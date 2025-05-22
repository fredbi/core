package fs

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// OverlayFS is a read-only [fs.FS] that merges a list of overlays on top of base file system.
//
// [OverlayFS] implements [fs.FS], [fs.ReadFileFS] and [fs.StatFS], and [fs.ReadDirFS] if the underlying
// file systems provide those, otherwise the corresponding methods return an error.
//
// [fs.Glob] is not supported by [OverlayFS].
type OverlayFS struct {
	layers []fs.FS
}

/*
// NewAferoOverlayFS builds an overlay from [afero.FS] s.
func NewAferoOverlayFS(base afero.Fs, overlays ...afero.Fs) *OverlayFS {
	aferoOverlays := make([]fs.FS, 0, len(overlays))
	for _, overlay := range overlays {
		aferoOverlays = append(aferoOverlays, &afero.IOFS{Fs: overlay})
	}

	return NewOverlayFS(&afero.IOFS{Fs: base}, aferoOverlays...)
}
*/

// NewOverlayFS builds an overlay file system from [fs.FS] s.
func NewOverlayFS(base fs.FS, overlays ...fs.FS) *OverlayFS {
	layers := make([]fs.FS, 0, len(overlays)+1)
	if len(overlays) > 0 {
		for i := len(overlays) - 1; i >= 0; i-- {
			layers = append(layers, overlays[i])
		}
	}
	layers = append(layers, base)

	return &OverlayFS{
		layers: layers,
	}
}

func (f *OverlayFS) Open(name string) (fs.File, error) {
	return f.openInLayers(name)
}

func (f *OverlayFS) ReadFile(name string) ([]byte, error) {
	layer, err := f.findInLayers(name)
	if err != nil {
		return nil, err
	}

	return fs.ReadFile(layer, name)
}

func (f *OverlayFS) Stat(name string) (fs.FileInfo, error) {
	layer, err := f.findInLayers(name)
	if err != nil {
		return nil, err
	}

	return fs.Stat(layer, name)
}

func (f *OverlayFS) ReadDir(name string) ([]fs.DirEntry, error) {
	layer, err := f.findInLayers(name)
	if err != nil {
		return nil, err
	}

	return fs.ReadDir(layer, name)
}

func (f *OverlayFS) openInLayers(name string) (fs.File, error) {
	for _, layer := range f.layers {
		file, err := layer.Open(name)
		if err == nil {
			return file, nil
		}

		if isNotFound(err) {
			continue
		}

		return nil, err
	}

	return nil, os.ErrNotExist
}

func (f *OverlayFS) findInLayers(name string) (fs.FS, error) {
	for _, layer := range f.layers {
		if statFS, supportsStat := layer.(fs.StatFS); supportsStat {
			_, err := statFS.Stat(name)
			if err == nil {
				return layer, nil
			}
			if isNotFound(err) {
				continue
			}

			return nil, err
		}
		file, err := layer.Open(name)
		if err == nil {
			_ = file.Close()

			return layer, nil
		}

		if isNotFound(err) {
			continue
		}

		return nil, err
	}

	return nil, os.ErrNotExist
}

func isNotFound(err error) bool {
	if oerr, isNotFound := err.(*os.PathError); isNotFound {
		if errors.Is(oerr.Err, os.ErrNotExist) || errors.Is(oerr.Err, syscall.ENOENT) || errors.Is(oerr.Err, syscall.ENOTDIR) {
			return true
		}
	}

	return false
}
