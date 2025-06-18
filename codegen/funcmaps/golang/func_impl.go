package golang

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/fredbi/core/mangling/inflect"
	"github.com/fredbi/core/swag/stringutils"
)

func hasSecureScheme(arg []string) bool {
	return stringutils.SubsetCI(arg, []string{"https", "wss"})
}

func hasInsecureScheme(arg []string) bool {
	return stringutils.SubsetCI(arg, []string{"http", "ws"})
}

func mediaMime(orig string) string {
	const maxMimeParts = 2
	return strings.SplitN(orig, ";", maxMimeParts)[0]
}

/*
func mediaGoName(media string) string {
	return pascalize(strings.ReplaceAll(media, "*", "Star"))
}
*/

/*
func importAlias(pkg string) string {
	_, k := path.Split(pkg)
	return k
}
*/

func padComment(str string, pads ...string) string {
	// pads specifes padding to indent multi line comments.Defaults to one space
	pad := " "
	lines := strings.Split(str, "\n")
	if len(pads) > 0 {
		pad = strings.Join(pads, "")
	}
	return (strings.Join(lines, "\n//"+pad))
}

func blockComment(str string) string {
	return strings.ReplaceAll(str, "*/", "[*]/")
}

/*
	func pascalize(arg string) string {
		runes := []rune(arg)
		switch len(runes) {
		case 0:
			return "Empty"
		case 1: // handle special case when we have a single rune that is not handled by ToGoName
			switch runes[0] {
			case '+', '-', '#', '_', '*', '/', '=': // those cases are handled differently than swag utility
				return prefixForName(arg)
			}
		}
		return swag.ToGoName(swag.ToGoName(arg)) // want to remove spaces
	}
*/
func escapeBackticks(arg string) string {
	return strings.ReplaceAll(arg, "`", "`+\"`\"+`")
}

func prefixForName(arg string) string {
	first := []rune(arg)[0]
	if len(arg) == 0 || unicode.IsLetter(first) {
		return ""
	}
	switch first {
	case '+':
		return "Plus"
	case '-':
		return "Minus"
	case '#':
		return "HashTag"
	case '*':
		return "Asterisk"
	case '/':
		return "ForwardSlash"
	case '=':
		return "EqualSign"
		// other cases ($,@ etc..) handled by ToGoName
	}
	return "Nr"
}

/*
func replaceSpecialChar(in rune) string {
	switch in {
	case '.':
		return "-Dot-"
	case '+':
		return "-Plus-"
	case '-':
		return "-Dash-"
	case '#':
		return "-Hashtag-"
	}
	return string(in)
}

func cleanupEnumVariant(in string) string {
	replaced := ""
	for _, char := range in {
		replaced += replaceSpecialChar(char)
	}
	return replaced
}
*/

func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("expected even number of arguments, got %d: %w", len(values), ErrFuncMap)
	}
	const sensibleAllocs = 2
	dict := make(map[string]any, len(values)/sensibleAllocs)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %+v: %w", values[i], ErrFuncMap)
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func isInteger(arg any) bool {
	// is integer determines if a value may be represented by an integer
	switch val := arg.(type) {
	case int8, int16, int32, int, int64, uint8, uint16, uint32, uint, uint64:
		return true
	case *int8, *int16, *int32, *int, *int64, *uint8, *uint16, *uint32, *uint, *uint64:
		v := reflect.ValueOf(arg)
		return !v.IsNil()
	case float64:
		return math.Round(val) == val
	case *float64:
		return val != nil && math.Round(*val) == *val
	case float32:
		return math.Round(float64(val)) == float64(val)
	case *float32:
		return val != nil && math.Round(float64(*val)) == float64(*val)
	case string:
		_, err := strconv.ParseInt(val, 10, 64)
		return err == nil
	case *string:
		if val == nil {
			return false
		}
		_, err := strconv.ParseInt(*val, 10, 64)
		return err == nil
	default:
		return false
	}
}

func httpStatus(code int) string {
	if name := http.StatusText(code); name != "" {
		return name
	}
	// non-standard codes deserve some name
	return fmt.Sprintf("Status %d", code)
}

func gt0(in *int64) bool {
	// gt0 returns true if the *int64 points to a value > 0
	// NOTE: plain {{ gt .MinProperties 0 }} just refuses to work normally
	// with a pointer
	return in != nil && *in > 0
}

const mdNewLine = "</br>"

var mdNewLineReplacer = strings.NewReplacer("\r\n", mdNewLine, "\n", mdNewLine, "\r", mdNewLine)

func markdownBlock(in string) string {
	in = strings.TrimSpace(in)

	return mdNewLineReplacer.Replace(in)
}

func asJSON(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func asPrettyJSON(data any) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func pluralizeFirstWord(arg string) string {
	sentence := strings.Split(arg, " ")
	if len(sentence) == 1 {
		return inflect.Pluralize(arg)
	}

	return inflect.Pluralize(sentence[0]) + " " + strings.Join(sentence[1:], " ")
}

func dropPackage(str string) string {
	parts := strings.Split(str, ".")
	return parts[len(parts)-1]
}

// return true if the GoType str contains pkg. For example "model.MyType" -> true, "MyType" -> false
func containsPkgStr(str string) bool {
	dropped := dropPackage(str)
	return dropped != str
}

func padSurround(entry, padWith string, i, ln int) string {
	res := make([]string, 0, ln)
	if i > 0 {
		for range i {
			res = append(res, padWith)
		}
	}
	res = append(res, entry)
	tot := ln - i - 1
	for range tot {
		res = append(res, padWith)
	}
	return strings.Join(res, ",")
}

func arrayInitializer(data any) (string, error) {
	// arrayInitializer constructs a Go literal initializer from any literals.
	// e.g. []any{"a", "b"} is transformed in {"a","b",}
	// e.g. map[string]any{ "a": "x", "b": "y"} is transformed in {"a":"x","b":"y",}.
	//
	// NOTE: this is currently used to construct simple slice intializers for default values.
	// This allows for nicer slice initializers for slices of primitive types and avoid systematic use for json.Unmarshal().
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(string(b), "}", ",}"), "[", "{"), "]", ",}"), "{,}", "{}"), nil
}

func generateImports(imports map[string]string) string {
	if len(imports) == 0 {
		return ""
	}
	result := make([]string, 0, len(imports))
	for k, v := range imports {
		_, name := path.Split(v)
		if name != k {
			result = append(result, fmt.Sprintf("\t%s %q", k, v))
		} else {
			result = append(result, fmt.Sprintf("\t%q", v))
		}
	}
	sort.Strings(result)
	return strings.Join(result, "\n")
}
