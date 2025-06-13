package models

import (
	"reflect"

	"github.com/fredbi/core/swag/mangling"
)

// experimental: should be moved to some other internal package

type SettingWithLocation struct {
	Name   string
	Type   string
	Path   []string
	Values []string
}

type Settings struct {
	Flags      []SettingWithLocation
	Extensions []SettingWithLocation
}

/*
func dot(index []int) string {
	if len(index) == 0 {
		return ""
	}
	var w strings.Builder
	w.WriteString(strconv.Itoa(index[0]))

	for _, i := range index[1:] {
		w.WriteString("." + strconv.Itoa(i))
	}

	return w.String()
}
*/

func ResolveSettings(o any) Settings {
	v := reflect.ValueOf(o)
	t := v.Type()
	m := mangling.Make(mangling.WithAdditionalInitialisms("CI"))
	fields := reflect.VisibleFields(t)
	flags := make([]SettingWithLocation, 0, len(fields))
	extensions := make([]SettingWithLocation, 0, len(fields))

	for _, field := range fields {
		if !field.IsExported() {
			continue
		}

		if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Map {
			continue
		}

		if field.Type.Kind() == reflect.Struct {
			/*
				section, ok := field.Tag.Lookup("section")
				if !ok {
					continue
				}
				parent := sections[dot(field.Index[:len(field.Index)-1])]
				sections[dot(field.Index)] = path.Join(parent, section)

			*/

			continue
		}

		name := m.Dasherize(field.Name)
		/*
			pth := make([]string, 0, 10)
			idx := dot(field.Index[:len(field.Index)-1])
			section, ok := sections[idx]
			if ok {
				pth = append(pth, section)
			}
		*/

		st := field.Type.String()
		//values := make([]string, 0, 5)
		flags = append(flags, SettingWithLocation{Name: name, Type: st})
		extensions = append(extensions, SettingWithLocation{Name: "x-go-" + name, Type: st})
	}

	return Settings{
		Flags:      flags,
		Extensions: extensions,
	}
}
