// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainsStringsCI(t *testing.T) {
	list := []string{"hello", "world", "and", "such"}

	assert.True(t, ContainsStringsCI(list, "hELLo"))
	assert.True(t, ContainsStringsCI(list, "world"))
	assert.True(t, ContainsStringsCI(list, "AND"))
	assert.False(t, ContainsStringsCI(list, "nuts"))
}

func TestContainsStrings(t *testing.T) {
	list := []string{"hello", "world", "and", "such"}

	assert.True(t, ContainsStrings(list, "hello"))
	assert.False(t, ContainsStrings(list, "hELLo"))
	assert.True(t, ContainsStrings(list, "world"))
	assert.False(t, ContainsStrings(list, "World"))
	assert.True(t, ContainsStrings(list, "and"))
	assert.False(t, ContainsStrings(list, "AND"))
	assert.False(t, ContainsStrings(list, "nuts"))
}

type mapWrapper map[string]any

func TestMapContains(t *testing.T) {
	m := make(map[string]any)
	m["x"] = struct{}{}
	m["y"] = struct{}{}

	require.True(t, MapContains(m, "x", "y", "z"))
	require.False(t, MapContains(m, "a", "z"))

	w := make(mapWrapper)
	w["x"] = struct{}{}
	w["y"] = struct{}{}

	require.True(t, MapContains(w, "x", "y", "z"))
	require.False(t, MapContains(w, "a", "z"))
}
