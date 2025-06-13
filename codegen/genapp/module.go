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
