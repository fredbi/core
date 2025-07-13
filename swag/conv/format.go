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

package conv

import (
	"strconv"
)

// FormatInteger turns an integer type into a string.
func FormatInteger[T Signed](value T) string {
	return strconv.FormatInt(int64(value), 10)
}

// FormatUinteger turns an unsigned integer type into a string.
func FormatUinteger[T Unsigned](value T) string {
	return strconv.FormatUint(uint64(value), 10)
}

// FormatFloat turns a floating point numerical value into a string.
func FormatFloat[T Float](value T) string {
	return strconv.FormatFloat(float64(value), 'g', -1, bitsize(value))
}

// FormatBool turns a boolean into a string.
func FormatBool(value bool) string {
	return strconv.FormatBool(value)
}

// AppendInteger appends the decimal representation of an integer to a slice of bytes.
func AppendInteger[T Signed](dst []byte, value T) []byte {
	return strconv.AppendInt(dst, int64(value), 10)
}

// AppendUinteger appends the decimal representation of an unsigned integer to a slice of bytes.
func AppendUinteger[T Unsigned](dst []byte, value T) []byte {
	return strconv.AppendUint(dst, uint64(value), 10)
}

// AppendFloat appends the decimal representation of a floating point number to a slice of bytes.
func AppendFloat[T Float](dst []byte, value T) []byte {
	return strconv.AppendFloat(dst, float64(value), 'g', -1, bitsize(value))
}

// AppendBool appends the text representation of a boolean to a slice of bytes.
func AppendBool(dst []byte, value bool) []byte {
	return strconv.AppendBool(dst, value)
}
