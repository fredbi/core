// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loading

import (
	"bytes"
	"embed"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func ReaderFromFileOrHTTP(pth string, opts ...Option) io.ReadCloser {
	o := optionsWithDefaults(opts)
	switch selectStrategy(pth) {
	case loadStrategyRemote:
		return o.RemoteReader(pth)
	case loadStrategyLocal:
		return wrapLocalReader(o.LocalReader, opts...)(pth)
	default:
		panic("internal error: invalid strategy")
	}
}

// LoadFromFileOrHTTP loads the bytes from a file or a remote http server based on the path passed in
func LoadFromFileOrHTTP(pth string, opts ...Option) ([]byte, error) {
	o := optionsWithDefaults(opts)
	return LoadStrategy(pth, o.ReadFileFunc(), o.LoadHTTPBytes(opts...), opts...)(pth)
}

// LoadStrategy returns a loader function for a given path or URI.
//
// The load strategy returns the remote load for any path starting with `http`.
// So this works for any URI with a scheme `http` or `https`.
//
// The fallback strategy is to call the local loader.
//
// The local loader takes a local file system path (absolute or relative) as argument,
// or alternatively a `file://...` URI, **without host** (see also below for windows).
//
// There are a few liberalities, initially intended to be tolerant regarding the URI syntax,
// especially on windows.
//
// Before the local loader is called, the given path is transformed:
//   - percent-encoded characters are unescaped
//   - simple paths (e.g. `./folder/file`) are passed as-is
//   - on windows, occurrences of `/` are replaced by `\`, so providing a relative path such a `folder/file` works too.
//
// For paths provided as URIs with the "file" scheme, please note that:
//   - `file://` is simply stripped.
//     This means that the host part of the URI is not parsed at all.
//     For example, `file:///folder/file" becomes "/folder/file`,
//     but `file://localhost/folder/file` becomes `localhost/folder/file` on unix systems.
//     Similarly, `file://./folder/file` yields `./folder/file`.
//   - on windows, `file://...` can take a host so as to specify an UNC share location.
//
// Reminder about windows-specifics:
// - `file://host/folder/file` becomes an UNC path like `\\host\folder\file` (no port specification is supported)
// - `file:///c:/folder/file` becomes `C:\folder\file`
// - `file://c:/folder/file` is tolerated (without leading `/`) and becomes `c:\folder\file`
func LoadStrategy(pth string, local, remote func(string) ([]byte, error), opts ...Option) func(string) ([]byte, error) {
	switch selectStrategy(pth) {
	case loadStrategyRemote:
		return remote
	case loadStrategyLocal:
		return wrapLocalLoader(local, opts...)
	default:
		panic("internal error: invalid strategy")
	}
}

type loadStrategy uint8

const (
	loadStrategyLocal loadStrategy = iota
	loadStrategyRemote
)

func selectStrategy(pth string) loadStrategy {
	if strings.HasPrefix(pth, "http") {
		return loadStrategyRemote
	}
	return loadStrategyLocal
}

func wrapLocalLoader(local func(string) ([]byte, error), opts ...Option) func(string) ([]byte, error) {
	localReader := func(p string) io.ReadCloser {
		buf, err := local(p)
		if err != nil {
			return errReadCloser{err: err}
		}

		return io.NopCloser(bytes.NewReader(buf))
	}

	return func(p string) ([]byte, error) {
		rdr := wrapLocalReader(localReader, opts...)(p)
		defer func() {
			if rdr != nil {
				_ = rdr.Close()
			}
		}()

		return io.ReadAll(rdr)
	}
}

func wrapLocalReader(local func(string) io.ReadCloser, opts ...Option) func(string) io.ReadCloser {
	o := optionsWithDefaults(opts)
	_, isEmbedFS := o.fs.(embed.FS)

	return func(p string) io.ReadCloser {
		upth, err := url.PathUnescape(p)
		if err != nil {
			return errReadCloser{err: err}
		}

		// TODO(fred): this has to be fixed properly using fred/core/uri (forked from fredbi/uri)
		cpth, hasPrefix := strings.CutPrefix(upth, "file://")
		if !hasPrefix || isEmbedFS || runtime.GOOS != "windows" {
			// crude processing: trim the file:// prefix. This leaves full URIs with a host with a (mostly) unexpected result
			// regular file path provided: just normalize slashes
			if isEmbedFS {
				// on windows, we need to slash the path if FS is an embed FS.
				return local(strings.TrimLeft(filepath.ToSlash(cpth), "./")) // remove invalid leading characters for embed FS
			}

			return local(filepath.FromSlash(cpth))
		}

		// windows-only pre-processing of file://... URIs, excluding embed.FS

		// support for canonical file URIs on windows.
		u, err := url.Parse(filepath.ToSlash(upth))
		if err != nil {
			return errReadCloser{err: err}
		}

		if u.Host != "" {
			// assume UNC name (volume share)
			// NOTE: UNC port not yet supported

			// when the "host" segment is a drive letter:
			// file://C:/folder/... => C:\folder
			upth = path.Clean(strings.Join([]string{u.Host, u.Path}, `/`))
			if !strings.HasSuffix(u.Host, ":") && u.Host[0] != '.' {
				// tolerance: if we have a leading dot, this can't be a host
				// file://host/share/folder\... ==> \\host\share\path\folder
				upth = "//" + upth
			}
		} else {
			// no host, let's figure out if this is a drive letter
			upth = strings.TrimPrefix(upth, `file://`)
			first, _, _ := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
			if strings.HasSuffix(first, ":") {
				// drive letter in the first segment:
				// file:///c:/folder/... ==> strip the leading slash
				upth = strings.TrimPrefix(upth, `/`)
			}
		}

		return local(filepath.FromSlash(upth))
	}
}
