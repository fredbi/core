package jsonschema

import (
	"bytes"
	"testing"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/jsonschema/overlay"
	"github.com/stretchr/testify/require"
)

func TestOverlay(t *testing.T) {
	t.Run("with valid schema overlay", func(t *testing.T) {
		const jazon = `{
			"overlay": "1.0.0",
			"info": {
			  "title": "test overlay",
			  "version": "v1.2.3-rc",
			  "x-info": {
			    "scope": "testing"
				}
			},
			"extends": "./baseSchema.json",
			"actions": [
				{
			    "description": "remove author_id from base schema",
			    "target":  "$.properties.author_id",
			    "remove": true,
			    "x-action": "propagate" 
				},
				{
			    "description": "add author and author_id properties to book",
			    "target":  "$.properties.book[*]",
			    "update": {
			      "properties": {
			        "author": {
			          "type": "string",
			          "minLength": 1
			        },
			        "author_id": {
			          "type": "string",
			          "format": "uuid"
			        }
			      }
			    },
			    "additionalKey": {"licensed": false}
				}
			],
			"x-overlay": true,
			"otherKey": 345
		}`

		t.Run("should decode", func(t *testing.T) {
			o := MakeOverlay()

			r := bytes.NewReader([]byte(jazon))
			require.NoError(t, o.Decode(r))

			t.Run("should encode as the original", func(t *testing.T) {
				var w bytes.Buffer
				require.NoError(t, o.Encode(&w))

				require.JSONEq(t, jazon, w.String())
			})

			t.Run("should hold expected extends field", func(t *testing.T) {
				require.Equal(t, "./baseSchema.json", o.Extends())
			})

			t.Run("should hold expected info object", func(t *testing.T) {
				require.Equal(t, overlay.Version10, o.Version())

				info := o.Info()
				require.NotEmpty(t, info)
				require.Equal(t, "test overlay", info.Title())
				require.Equal(t, "v1.2.3-rc", info.Version())

				ext := info.Extensions()
				require.True(t, ext.Has("x-info"))
			})

			t.Run("should hold expected actions", func(t *testing.T) {
				i := 0
				for action := range o.Actions() {
					switch i {
					case 0:
						t.Run("on first action (remove)", func(t *testing.T) {
							require.Equal(t, "remove author_id from base schema", action.Description())
							require.Equal(t, "$.properties.author_id", action.Target().String())
							require.True(t, action.Remove())
							doc := action.Update()
							require.True(t, doc.IsEmpty())
							ext := action.Extensions()
							require.NotEmpty(t, ext)
							require.True(t, ext.Has("x-action"))
						})
					case 1:
						t.Run("on second action (update)", func(t *testing.T) {
							require.Equal(t, "add author and author_id properties to book", action.Description())
							require.Equal(t, "$.properties.book[*]", action.Target().String())
							require.False(t, action.Remove())

							doc := action.Update()
							require.NotEqual(t, json.EmptyDocument, doc)

							update, err := doc.MarshalJSON()
							require.NoError(t, err)

							const expectedUpdate = `{
			          "properties": {
			            "author": {
			            "type": "string",
			            "minLength": 1
			            },
			            "author_id": {
			              "type": "string",
			              "format": "uuid"
			            }
			          }
			        }`
							require.JSONEq(t, expectedUpdate, string(update))

							ext := action.Extensions()
							require.Empty(t, ext)
						})
					}
					i++
				}
				require.Equal(t, 2, i)
			})

			t.Run("should hold extension", func(t *testing.T) {
				ext := o.Extensions()
				require.NotEmpty(t, ext)
				require.True(t, ext.Has("x-overlay"))
			})
		})

		t.Run("should unmarshal", func(t *testing.T) {
			o := MakeOverlay()
			require.NoError(t, o.UnmarshalJSON([]byte(jazon)))
			t.Run("should marshal as the original", func(t *testing.T) {
				output, err := o.MarshalJSON()
				require.NoError(t, err)

				require.JSONEq(t, jazon, string(output))
			})
		})
	})
}
