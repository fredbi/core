package repo

import (
	"fmt"
	"slices"
	"strings"
	"text/template"
	"text/template/parse"
)

// findDependencies finds template findDependencies from a given node in the template AST.
func findDependencies(n parse.Node) []string {
	const sensibleDependenciesAllocs = 5
	deps := make([]string, 0, sensibleDependenciesAllocs)
	depMap := make(map[string]bool)

	if n == nil {
		return deps
	}

	switch node := n.(type) {
	case *parse.ListNode:
		if node != nil && node.Nodes != nil {
			for _, nn := range node.Nodes {
				for _, dep := range findDependencies(nn) {
					depMap[dep] = true
				}
			}
		}
	case *parse.IfNode:
		for _, dep := range findDependencies(node.List) {
			depMap[dep] = true
		}
		for _, dep := range findDependencies(node.ElseList) {
			depMap[dep] = true
		}

	case *parse.RangeNode:
		for _, dep := range findDependencies(node.List) {
			depMap[dep] = true
		}
		for _, dep := range findDependencies(node.ElseList) {
			depMap[dep] = true
		}

	case *parse.WithNode:
		for _, dep := range findDependencies(node.List) {
			depMap[dep] = true
		}
		for _, dep := range findDependencies(node.ElseList) {
			depMap[dep] = true
		}

	case *parse.TemplateNode:
		depMap[node.Name] = true
	}

	for dep := range depMap {
		deps = append(deps, dep)
	}

	return slices.Clip(deps)
}

// flattenDependencies flattens the graph of dependencies into a flat index.
func (r *Repository) flattenDependencies(tpl *template.Template, dependencies map[string]bool) map[string]bool {
	if dependencies == nil {
		dependencies = make(map[string]bool)
	}

	deps := findDependencies(tpl.Root)

	for _, d := range deps {
		if _, found := dependencies[d]; !found {

			dependencies[d] = true

			if tt := r.templates[d]; tt != nil {
				dependencies = r.flattenDependencies(tt, dependencies)
			}
		}

		dependencies[d] = true

	}

	return dependencies
}

// addDependencies resolves all dependencies and add them to the parse tree of the dependent template.
func (r *Repository) addDependencies(tpl *template.Template) (*template.Template, error) {
	name := tpl.Name()

	deps := r.flattenDependencies(tpl, nil)

	for dep := range deps {
		if dep == "" {
			continue
		}

		inner := tpl.Lookup(dep)
		if inner != nil {
			continue
		}

		// check if we have it stored in some other place
		cached, found := r.templates[dep]

		// still don't have it, return an error
		if !found {
			return tpl, fmt.Errorf("could not find template dependency %q: %w", dep, ErrTemplateRepo)
		}

		// add it to the parse tree
		var err error
		tpl, err = tpl.AddParseTree(dep, cached.Tree)
		if err != nil {
			return tpl, fmt.Errorf("dependency error: %w", err)
		}
	}

	return tpl.Lookup(name), nil
}

// resolveDependencies resolves and checks all dependencies for loaded templates.
func (r *Repository) resolveDependencies() error {
	for name, tpl := range r.templates {
		if err := r.resolveDependenciesFor(name, tpl); err != nil {
			return err
		}
	}

	return nil
}

// resolveDependenciesFor resolves and checks dependencies for a single template.
func (r *Repository) resolveDependenciesFor(name string, tpl *template.Template) error {
	resolved, err := r.addDependencies(tpl)
	if err != nil {
		return fmt.Errorf("could not resolve dependency for template %q: %w", name, err)
	}

	r.templates[name] = resolved

	return nil
}

// findRootComment looks for the first comment in a template source file or the first comment in a list of nodes.
func findRootComment(n parse.Node) ([]string, bool) {
	switch node := n.(type) {
	case *parse.CommentNode:
		return []string{stripCommentMarks(node.Text)}, true
	case *parse.ListNode:
		var concat []string
		for _, sub := range node.Nodes {
			if comment, ok := findRootComment(sub); ok {
				concat = append(concat, comment...)
			}
		}

		return concat, len(concat) > 0
	default:
		return nil, false
	}
}

// stripCommentMarks strips the comment marks from template comments.
func stripCommentMarks(text string) string {
	return commentsStripper.Replace(text)
}

var commentsStripper = strings.NewReplacer("/*", "", "*/", "")
