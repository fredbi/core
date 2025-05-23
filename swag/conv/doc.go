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

// Package conv exposes utilities to convert types.
//
// The Convert and Format families of functions are essentially a shorthand to [strconv] functions,
// using the decimal representation of numbers.
//
// Features:
//
//   - from string representation to value ("Convert*") and reciprocally ("Format*")
//   - from pointer to value ([Value]) and reciprocally ([Pointer])
//   - from slice of values to slice of pointers ([PointerSlice]) and reciprocally ([ValueSlice])
//   - from map of values to map of pointers ([PointerMap]) and reciprocally ([ValueMap])
package conv
