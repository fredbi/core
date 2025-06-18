package genapp

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// GoMod initializes a go.mod file in the generated target or update and existing one with go mod tidy.
//
// The go mod tidy command is applied to build dependencies.
//
// All possibilities allowed by the go modules are not covered.
// The supported use-case is when a the package name in a folder is directly inferred from the folder name.
// The go.mod file may alter the path to that folder, but would always have the folder name as a base name.
//
// Example:
//
// Supported:
//
// In /tmp/go-folder:
//
//	go.mod
//	module swagger.io/go-openapi/go-folder/v2
//
//	doc.go
//	package folder // <- folder is the "official" short name for "go-folder/v2"
//
// Not supported:
//
// In /tmp/go-folder:
//
//	go.mod
//	module github.com/json-iterator/go
//
//	doc.go
//	package jsoniterator   // <- unrelated with folder. This is possible, but tooling easily gets lost.
//
// NOTE: this doesn't work when configuring [GoGenApp] with an [afero.Fs] which is not the os FS.
func (g *GoGenApp) GoMod(opts ...ModOption) error {
	o := modOptionsWithDefaults(opts)

	_, fileExistsErr := os.Stat(filepath.Join(g.outputPath, "go.mod"))
	if os.IsNotExist(fileExistsErr) {
		initCmd := exec.Command(
			"go",
			"mod",
			"init",
		)
		if o.modulePath != "" {
			initCmd.Args = append(initCmd.Args, o.modulePath)
		}

		initCmd.Dir = g.outputPath
		initCmd.Stderr = os.Stderr
		initCmd.Stdout = io.Discard

		if err := initCmd.Run(); err != nil {
			return err
		}
	}

	tidyCmd := exec.Command(
		"go",
		"mod",
		"tidy",
	)
	if o.goVersion != "" {
		tidyCmd.Args = append(tidyCmd.Args, "-go="+o.goVersion)
	}
	tidyCmd.Dir = g.outputPath
	tidyCmd.Stderr = os.Stderr
	tidyCmd.Stdout = io.Discard

	return tidyCmd.Run()
}
