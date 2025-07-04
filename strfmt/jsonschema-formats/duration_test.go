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

package formats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestDuration(t *testing.T) {
	pp := Duration(0)

	err := pp.UnmarshalText([]byte("0ms"))
	require.NoError(t, err)
	err = pp.UnmarshalText([]byte("yada"))
	require.Error(t, err)

	orig := "2ms"
	b := []byte(orig)
	bj := []byte("\"" + orig + "\"")

	err = pp.UnmarshalText(b)
	require.NoError(t, err)

	err = pp.UnmarshalText([]byte("three week"))
	require.Error(t, err)

	err = pp.UnmarshalText([]byte("9999999999999999999999999999999999999999999999999999999 weeks"))
	require.Error(t, err)

	txt, err := pp.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, orig, string(txt))

	err = pp.UnmarshalJSON(bj)
	require.NoError(t, err)
	assert.EqualValues(t, orig, pp.String())

	err = pp.UnmarshalJSON([]byte("yada"))
	require.Error(t, err)

	err = pp.UnmarshalJSON([]byte(`"12 parsecs"`))
	require.Error(t, err)

	err = pp.UnmarshalJSON([]byte(`"12 y"`))
	require.Error(t, err)

	b, err = pp.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, bj, b)

	dur := Duration(42)
	bsonData, err := bson.Marshal(&dur)
	require.NoError(t, err)

	var durCopy Duration
	err = bson.Unmarshal(bsonData, &durCopy)
	require.NoError(t, err)
	assert.Equal(t, dur, durCopy)
}

func testDurationParser(t *testing.T, toParse string, expected time.Duration) {
	t.Helper()

	r, e := ParseDuration(toParse)
	require.NoError(t, e)
	assert.Equal(t, expected, r)
}

func TestDurationParser_Failed(t *testing.T) {
	_, e := ParseDuration("45 wekk")
	require.Error(t, e)
}

func TestIsDuration_Failed(t *testing.T) {
	e := IsDuration("45 weeekks")
	assert.False(t, e)
}

func testDurationSQLScanner(t *testing.T, dur time.Duration) {
	t.Helper()

	values := []interface{}{int64(dur), float64(dur)}
	for _, value := range values {
		var result Duration
		err := result.Scan(value)
		require.NoError(t, err)
		assert.Equal(t, dur, time.Duration(result))

		// And the other way around
		resv, erv := result.Value()
		require.NoError(t, erv)
		assert.EqualValues(t, value, resv)

	}
}

func TestDurationScanner_Nil(t *testing.T) {
	var result Duration
	err := result.Scan(nil)
	require.NoError(t, err)
	assert.EqualValues(t, 0, time.Duration(result))

	err = result.Scan("1 ms")
	require.Error(t, err)
}

func TestDurationParser(t *testing.T) {
	testcases := map[string]time.Duration{
		// parse the short forms without spaces
		"1ns": 1 * time.Nanosecond,
		"1us": 1 * time.Microsecond,
		"1µs": 1 * time.Microsecond,
		"1ms": 1 * time.Millisecond,
		"1s":  1 * time.Second,
		"1m":  1 * time.Minute,
		"1h":  1 * time.Hour,
		"1hr": 1 * time.Hour,
		"1d":  24 * time.Hour,
		"1w":  7 * 24 * time.Hour,
		"1wk": 7 * 24 * time.Hour,

		// parse the long forms without spaces
		"1nanoseconds":  1 * time.Nanosecond,
		"1nanos":        1 * time.Nanosecond,
		"1microseconds": 1 * time.Microsecond,
		"1micros":       1 * time.Microsecond,
		"1millis":       1 * time.Millisecond,
		"1milliseconds": 1 * time.Millisecond,
		"1second":       1 * time.Second,
		"1sec":          1 * time.Second,
		"1min":          1 * time.Minute,
		"1minute":       1 * time.Minute,
		"1hour":         1 * time.Hour,
		"1day":          24 * time.Hour,
		"1week":         7 * 24 * time.Hour,

		// parse the short forms with spaces
		"1  ns": 1 * time.Nanosecond,
		"1  us": 1 * time.Microsecond,
		"1  µs": 1 * time.Microsecond,
		"1  ms": 1 * time.Millisecond,
		"1  s":  1 * time.Second,
		"1  m":  1 * time.Minute,
		"1  h":  1 * time.Hour,
		"1  hr": 1 * time.Hour,
		"1  d":  24 * time.Hour,
		"1  w":  7 * 24 * time.Hour,
		"1  wk": 7 * 24 * time.Hour,

		// parse the long forms without spaces
		"1  nanoseconds":  1 * time.Nanosecond,
		"1  nanos":        1 * time.Nanosecond,
		"1  microseconds": 1 * time.Microsecond,
		"1  micros":       1 * time.Microsecond,
		"1  millis":       1 * time.Millisecond,
		"1  milliseconds": 1 * time.Millisecond,
		"1  second":       1 * time.Second,
		"1  sec":          1 * time.Second,
		"1  min":          1 * time.Minute,
		"1  minute":       1 * time.Minute,
		"1  hour":         1 * time.Hour,
		"1  day":          24 * time.Hour,
		"1  week":         7 * 24 * time.Hour,
	}

	for str, dur := range testcases {
		t.Run(str, func(t *testing.T) {
			testDurationParser(t, str, dur)

			// negative duration
			testDurationParser(t, "-"+str, -dur)
			testDurationParser(t, "- "+str, -dur)

			testDurationSQLScanner(t, dur)
		})
	}
}

func TestIsDuration_Caveats(t *testing.T) {
	// This works too
	e := IsDuration("45 weeks")
	assert.True(t, e)

	// This works too
	e = IsDuration("45 weekz")
	assert.True(t, e)

	// This works too
	e = IsDuration("12 hours")
	assert.True(t, e)

	// This works too
	e = IsDuration("12 minutes")
	assert.True(t, e)

	// This does not work
	e = IsDuration("12 phours")
	assert.False(t, e)
}

func TestDeepCopyDuration(t *testing.T) {
	dur := Duration(42)
	in := &dur

	out := new(Duration)
	in.DeepCopyInto(out)
	assert.Equal(t, in, out)

	out2 := in.DeepCopy()
	assert.Equal(t, in, out2)

	var inNil *Duration
	out3 := inNil.DeepCopy()
	assert.Nil(t, out3)
}
