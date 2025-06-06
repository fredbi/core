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

package mangling

import (
	"fmt"
	"io"
	"testing"
)

func BenchmarkToXXXName(b *testing.B) {
	samples := []string{
		"sample text",
		"sample-text",
		"sample_text",
		"sampleText",
		"sample 2 Text",
		"findThingById",
		"日本語sample 2 Text",
		"日本語findThingById",
		"findTHINGSbyID",
	}
	m := Make()

	b.Run("ToGoName", benchmarkFunc(m.ToGoName, samples))
	b.Run("ToVarName", benchmarkFunc(m.ToVarName, samples))
	b.Run("ToFileName", benchmarkFunc(m.ToFileName, samples))
	b.Run("ToCommandName", benchmarkFunc(m.ToCommandName, samples))
	b.Run("ToHumanNameLower", benchmarkFunc(m.ToHumanNameLower, samples))
	b.Run("ToHumanNameTitle", benchmarkFunc(m.ToHumanNameTitle, samples))
}

func benchmarkFunc(fn func(string) string, samples []string) func(*testing.B) {
	return func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		var res string
		for i := 0; i < b.N; i++ {
			res = fn(samples[i%len(samples)])
		}

		fmt.Fprintln(io.Discard, res)
	}
}
