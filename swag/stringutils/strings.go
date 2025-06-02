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
	"slices"
	"strings"
)

// ContainsStrings searches a slice of strings for a case-sensitive match
//
// Now equivalent to the standard library [slice.Contains].
func ContainsStrings(collection []string, item string) bool {
	return slices.Contains(collection, item)
}

// Subset returns true if all elements of subset are present in the collection.
//
// It always returns true for empty subsets.
func Subset(collection []string, subset []string) bool {
	for _, item := range subset {
		if !slices.Contains(collection, item) {
			return false
		}
	}

	return true
}

// ContainsStringsCI searches a slice of strings for a case-insensitive match
func ContainsStringsCI(collection []string, item string) bool {
	return slices.ContainsFunc(collection, func(e string) bool {
		return strings.EqualFold(e, item)
	})
}

// Subset returns true if all elements of subset are present in the collection, with a case-insensitive match.
//
// It always returns true for empty subsets.
func SubsetCI(collection []string, subset []string) bool {
	for _, item := range subset {
		if !ContainsStringsCI(collection, item) {
			return false
		}
	}

	return true
}

func MapContains[Map ~map[K]V, K comparable, V any](m Map, items ...K) bool {
	if len(items) == 0 {
		return true
	}
	if len(m) == 0 {
		return false
	}

	for _, item := range items {
		if _, ok := m[item]; ok {
			return true
		}
	}

	return false
}
