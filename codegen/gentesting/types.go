package gentest

type symbolsIndex map[ExportedSymbolKind][]symbol

func (i symbolsIndex) NamesForKind(kind ExportedSymbolKind) []string {
	list, ok := i[kind]
	if !ok {
		return nil
	}

	values := make([]string, 0, len(list))
	for _, element := range list {
		values = append(values, element.String())
	}

	return values
}

type pair struct {
	Ident  string
	Target string
}

type ExportedSymbolKind string

const (
	noSymbol    ExportedSymbolKind = ""
	SymbolConst ExportedSymbolKind = "const"
	SymbolVar   ExportedSymbolKind = "var"
	SymbolType  ExportedSymbolKind = "type"
	SymbolFunc  ExportedSymbolKind = "func"
)

func (k ExportedSymbolKind) String() string {
	return string(k)
}

type symbol struct {
	Kind  ExportedSymbolKind
	Pkg   string
	Ident string
}

func (s symbol) String() string {
	return s.Pkg + "." + s.Ident
}
