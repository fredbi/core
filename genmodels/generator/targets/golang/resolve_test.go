package golang

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fredbi/core/swag/mangling"
	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestResolveSettings(t *testing.T) {
	var o GenOptions
	s := ResolveSettings(o)

	jazon, err := json.MarshalIndent(s, "", "  ")
	require.NoError(t, err)
	t.Log(string(jazon))
}

func TestMarshalSetting(t *testing.T) {
	var (
		o GenOptions
		d map[string]any
	)
	m := mangling.Make(mangling.WithAdditionalInitialisms("CI"))
	//fromMap := mapstructure.DecodeHookFuncValue(mapstructure.RecursiveStructToMapHookFunc().(func(from reflect.Value, to reflect.Value) (any, error)))
	//index := make(map[string]string, 20)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &d,
		TagName: "section",
		DecodeHook: mapstructure.DecodeHookFuncType(func(from reflect.Type, to reflect.Type, data any) (any, error) {
			t.Logf("%v, %v, %#v", from, to, data)

			return data, nil
		}),
	})
	require.NoError(t, err)
	require.NoError(t, decoder.Decode(o))

	// rewrite map with mangled keys
	var rewriteMap func(d any) any
	rewriteMap = func(d any) any {
		asMap, ok := d.(map[string]any)
		if !ok {
			return d
		}

		rewritten := make(map[string]any, len(asMap))
		for k, v := range asMap {
			rewritten[m.ToVarName(k)] = rewriteMap(v)
		}

		return rewritten
	}
	e := rewriteMap(d)

	/*
		jazon, err := json.MarshalIndent(e, "", "  ")
		require.NoError(t, err)
		t.Log(string(jazon))
	*/

	y, err := yaml.Marshal(e)
	require.NoError(t, err)
	t.Log(string(y))

}
