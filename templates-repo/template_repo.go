package repo

import (
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"text/template/parse"
)

// New creates a new template repository.
func New(opts ...Option) *Repository {
	repo := Repository{
		files:     make(map[string]string),             // index of assets, by template name. This is used by [Repository.Dump]
		templates: make(map[string]*template.Template), // index of templates, by template name
		protected: make(map[string]bool),               // index of protected templates
		options:   optionsWithDefaults(opts),
	}

	if repo.parseComments { // index of docstrings, when the parseComment option is enabled
		repo.docstrings = make(map[string][]string)
	}

	return &repo
}

// Repository is a cache for golang text templates.
//
// A fresh [Repository] is initialized with [New].
//
// Default settings may be altered using [Option] s.
//
// # Supported templates
//
// The [Repository] supports golang text [template.Template]. html templates are not supported at this moment.
//
// The default file extension for template assets is ".gotmpl".
//
// You may change the list of supported extensions using [WithExtensions].
// Any asset with an extensions that is not in this list is ignored.
//
// # Structure
//
// The [Repository] organizes all resolved templates from the source templates directory on [fs.FS] as a flat index
// of named templates.
//
// Example:
//
// The template defined in file "{root}/cli/generate.gotmpl" would be later called as "cliGenerate".
//
// Templates defined in dfferent folders would therefore never conflict.
// However, inner templates (using inner "{{ define }}" statements) are resolved at the same level.
//
// You should therefore make sure that you don't define the same template several times in the same folder.
//
// # Template dependencies
//
// When complex templates call each other, or define inner "{{ define ...}}" templates, the graph of dependencies
// may become hard to resolve.
//
// The [Repository) solves this by checking all inner definitions and dependencies at loading time and ensure
// that all dependencies are resolved for each dependent template.
//
// # All templates defined in the
//
// # Overlays
//
// You may reload templates from an alternate source and override existing templates.
//
// Using [Repository.LoadOverlay] allows to specify the prefix to strip when producing unique template names.
//
// # Protected templates
//
// Using [Repository.AddFile] or [Repository.LoadOverlay], it is possible to override already loaded templates.
//
// Protected templates are a configuration that prevents undesirable overrides to occur.
//
// You may define which templates are protected with [Repository.SetProtected].
//
// # Concurrency
//
// The repository may be used concurrently: templates are compiled and dependencies resolved early when loading
// a directory or a template asset.
//
// Loading may also be carried out concurrently. However, the outcome of dependency resolution may depend on the order
// in which loading occurs.
type Repository struct {
	files      map[string]string
	templates  map[string]*template.Template
	protected  map[string]bool
	mux        sync.RWMutex
	docstrings map[string][]string
	options
}

// Clone builds a clone of a repository.
// with cloned maps of templates. Compiled templates are shallow clone.
//
// The clone may be parameterized with options that differ from the original.
func (r *Repository) Clone(opts ...Option) *Repository {
	clone := &Repository{
		files:     make(map[string]string, len(r.files)),
		templates: make(map[string]*template.Template, len(r.templates)),
	}

	r.mux.Lock()
	defer r.mux.Unlock()

	clone.files = maps.Clone(r.files)
	clone.templates = maps.Clone(r.templates)
	clone.options = r.cloneOptions(opts)

	return clone
}

// SetProtected defines the map of protected templates, by their name.
//
// Template names mapped to true are protected, and unprotected from overrides otherwise.
//
// Using [Repository.SetProtected](map[string]bool{}) would unprotected all templates.
//
// Setting protected templates that are not loaded yet has not effect.
func (r *Repository) SetProtected(protected map[string]bool) {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.protected = protected

	maps.DeleteFunc(r.protected, func(name string, _ bool) bool {
		// ensure protected settings correspond to already loaded templates
		_, ok := r.templates[name]

		return !ok
	})
}

// AddProtected protects already loaded templates.
func (r *Repository) AddProtected(protectedTemplates ...string) {
	r.mux.Lock()
	defer r.mux.Unlock()

	for _, protected := range protectedTemplates {
		if _, ok := r.templates[protected]; ok {
			r.protected[protected] = true
		}
	}
}

// Load will walk the specified path and add to the repository each template asset it finds in this tree.
func (r *Repository) Load(templatePath string) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	err := fs.WalkDir(r.fs, templatePath, r.loadWithAssetName(func(path string) (string, error) {
		return filepath.Rel(templatePath, path) // TODO: won't work with embed on windows
	}))

	if err != nil {
		return fmt.Errorf("could not complete template processing in directory %q: %w: %w", templatePath, err, ErrTemplateRepo)
	}

	return r.resolveDependencies()
}

// LoadOverlay loads templates from a directory as an overlay.
//
// The directory name is stripped from the name prefix, so templates in this folder may
// override existing templates.
//
// Example:
//
// If we define the following structure:
//
//	templates/headers.gotmpl
//	templates/contrib/override/headers.gotmpl
//
// Then [Repository.LoadOverlay]("./contrib/override", "contrib/override") would replace the template "headers" by the template located in "contrib/override".
//
// If no prefix is provided the directory name (without leading . or /) is used.
//
// Overlay templates must ensure that they provide the right dependencies.
//
// If an overlay redefines an existing dependency, this dependency will be taken into account on by overridden templates.
// Templates that are not overridden and have been already resolved won't resolve to the new dependency.
func (r *Repository) LoadOverlay(overlayPath, prefix string) error {
	if prefix == "" {
		prefix = filepath.Clean(overlayPath)
	}
	r.mux.Lock()
	defer r.mux.Unlock()

	err := fs.WalkDir(r.fs, overlayPath, r.loadWithAssetName(func(path string) (string, error) {
		return filepath.Rel(overlayPath, strings.TrimPrefix(path, prefix)) // TODO: won't work with embed on windows
	}))

	if err != nil {
		return fmt.Errorf("could not complete template processing in directory %q: %w: %w", overlayPath, err, ErrTemplateRepo)
	}

	return r.resolveDependencies()
}

// Get returns a named template from the repository, ensuring that all dependent templates are loaded.
//
// It yields an error if the requested template or any template it depends on is not defined in the repository.
//
// [Repository.Get] may be called concurrently.
func (r *Repository) Get(name string) (*template.Template, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	return r.get(name)
}

// MustGet retrieves a template by its name, or panics if it fails.
//
// [Repository.MustGet] may be called concurrently.
func (r *Repository) MustGet(name string) *template.Template {
	tpl, err := r.Get(name)
	if err != nil {
		panic(err)
	}

	return tpl
}

// AddFile adds or replace a single template asset to the repository.
//
// It creates a new template based on the asset name and the template content.
//
// The name of the template, to be retrieved using [Repository.Get], is built from the asset name:
//
// - It trims the extension from the end and converts the name using swag.ToJSONName
// - This strips directory separators and camel-cases the next letter.
//
// Example:
//
//	file validation/primitive.gotmpl is referred to as "validationPrimitive"
//
// The asset is not added if it contains a definition for a template that is protected.
//
// The newly added asset should not add new dependencies: these should be already loaded.
//
// If you are not sure about the order of dependencies, prefer [Repository.Load] or [Repository.LoadOverlay]
// to resolve dependencies only after all file assets are loaded.
func (r *Repository) AddFile(filename string, data []byte) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	name, err := r.addFile(filename, data)
	if err != nil {
		return err
	}

	tpl, err := r.get(name)
	if err != nil {
		return err
	}

	return r.resolveDependenciesFor(name, tpl)
}

// loadWithAssetName returns the [fs.WalkDirFunc] to be used in calls to [fs.WalkDir].
//
// The way we define the template name (asset name) may be parameterized with assetNameFunc.
func (r *Repository) loadWithAssetName(assetNameFunc func(path string) (string, error)) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, _ error) error {
		if d.IsDir() {
			for _, skipped := range r.skipDirectories {
				_, skip := strings.CutSuffix(path, skipped)
				if skip {
					return fs.SkipDir
				}
			}
		}

		supported := false
		for _, ext := range r.extensions {
			_, ok := strings.CutSuffix(path, ext)
			if ok {
				supported = true
				break
			}
		}
		if !supported { // unsupported assets are skipped
			return nil
		}

		assetName, err := assetNameFunc(path)
		if err != nil {
			return err
		}

		data, err := r.fs.ReadFile(path)
		if err != nil {
			return err
		}

		if _, ee := r.addFile(assetName, data); ee != nil {
			return fmt.Errorf("could not add template %q: %w", assetName, ee)
		}

		return nil
	}
}

func (r *Repository) get(name string) (*template.Template, error) {
	tpl, found := r.templates[name]
	if !found {
		return nil, fmt.Errorf("template doesn't exist %s: %w", name, ErrTemplateRepo)
	}

	return tpl, nil
}

func (r *Repository) addFile(filename string, data []byte) (string, error) {
	name := r.mangler.ToJSONName(r.trimExtension(filename))
	tpl, err := template.New(name).Funcs(r.funcs).Parse(string(data))
	if err != nil {
		return name, fmt.Errorf("failed to load template %s: %w: %w", name, err, ErrTemplateRepo)
	}

	// check if any protected templates are defined
	if !r.allowOverride {
		for _, template := range tpl.Templates() {
			if r.protected[template.Name()] {
				return name, fmt.Errorf("cannot overwrite protected template %s: %w", template.Name(), ErrTemplateRepo)
			}
		}
	}

	// add each defined template into the cache
	for _, template := range tpl.Templates() {
		r.files[template.Name()] = filename
		r.templates[template.Name()] = template.Lookup(template.Name())
	}

	if r.parseComments {
		err = r.parseCommentsFor(name, data)
		if err != nil {
			return name, fmt.Errorf("internal error: could not parse comments in template %s: %w :%w", name, err, ErrTemplateRepo)
		}
	}

	return name, nil
}

// parseCommentsFor parses comments in the template source, and generates docstrings.
func (r *Repository) parseCommentsFor(name string, data []byte) error {
	tree := parse.New(name, r.funcs)
	tree.Mode = parse.ParseComments | parse.SkipFuncCheck
	treeSet := make(map[string]*parse.Tree)
	parsedTree, err := tree.Parse(string(data), "", "", treeSet, r.funcs)
	if err != nil {
		return fmt.Errorf("internal error: could not parse comments: %w", err)
	}

	var docstrings []string
	comments, ok := findRootComment(parsedTree.Root)
	if ok {
		docstrings = append(docstrings, comments...)
	}

	if len(docstrings) > 0 {
		r.docstrings[name] = comments
	}

	for asset, parsedTree := range treeSet {
		var docstrings []string
		comments, ok := findRootComment(parsedTree.Root)
		if ok {
			docstrings = append(docstrings, comments...)
		}
		if len(docstrings) > 0 {
			r.docstrings[asset] = append(r.docstrings[asset], comments...)
		}
	}

	return nil
}
