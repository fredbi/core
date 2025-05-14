package golang

/*
// AddFile adds a file to the default repository. It will create a new template based on the filename.
// It trims the .gotmpl from the end and converts the name using swag.ToJSONName. This will strip
// directory separators and Camelcase the next letter.
// e.g validation/primitive.gotmpl will become validationPrimitive
//
// If the file contains a definition for a template that is protected the whole file will not be added
func AddFile(name, data string) error {
	return templates.addFile(name, data, false)
}
*/

/*
// LoadDefaults will load the embedded templates
func (t *Repository) LoadDefaults() {
	for name, asset := range assets {
		if err := t.addFile(name, string(asset), true); err != nil {
			log.Fatal(err)
		}
	}
}
*/
/*
import (
	"embed"
	"io/fs"
)

//go:embed templates
var _bindata embed.FS

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0)
	_ = fs.WalkDir(_bindata, "templates", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		names = append(names, path)
		return nil
	})
	return names
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	return _bindata.ReadFile(name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}
*/
