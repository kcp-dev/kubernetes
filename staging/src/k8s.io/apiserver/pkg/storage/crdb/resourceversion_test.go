/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package crdb

import (
	"strconv"
	"testing"

	"github.com/cockroachdb/apd"
)

func newFromStringOrDie(input string) *apd.Decimal {
	d, c, e := apd.NewFromString(input)
	if c.Any() {
		panic(c.String())
	}
	if e != nil {
		panic(e)
	}
	return d
}

func TestToResourceVersion(t *testing.T) {
	var testCases = []struct {
		name      string
		input     *apd.Decimal
		output    int64
		expectErr bool
	}{
		{
			name:   "no logical clock",
			input:  newFromStringOrDie("1640995200000000001"), // datum + 1
			output: 16,
		},
		{
			name:   "large physical clock",
			input:  newFromStringOrDie(strconv.Itoa(1<<20 + 1640995200000000000)),
			output: 1 << 24,
		},
		{
			name:   "with logical clock",
			input:  newFromStringOrDie("1640995200000000001.0000000001"),
			output: 17,
		},
		{
			name:      "with physical clock overflow",
			input:     newFromStringOrDie("2217455952303423488.0000000000"),
			expectErr: true,
		},
		{
			name:      "with logical clock overflow",
			input:     newFromStringOrDie("1640995200000000001.0000000016"),
			expectErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := toResourceVersion(testCase.input)
			if err == nil && testCase.expectErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if err != nil && !testCase.expectErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
			}
			if actual != testCase.output {
				t.Errorf("%s: expected %d but got %d", testCase.name, testCase.output, actual)
			}
		})
	}
}

func TestToHybridLogicalClock(t *testing.T) {
	var testCases = []struct {
		name      string
		input     int64
		output    *apd.Decimal
		expectErr bool
	}{
		{
			name:   "no logical clock",
			input:  16,
			output: newFromStringOrDie("1640995200000000001"),
		},
		{
			name:   "large physical clock",
			input:  1 << 24,
			output: newFromStringOrDie(strconv.Itoa(1<<20 + 1640995200000000000)),
		},
		{
			name:   "with logical clock",
			input:  17,
			output: newFromStringOrDie("1640995200000000001.0000000001"),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := toHybridLogicalClock(testCase.input)
			if err == nil && testCase.expectErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if err != nil && !testCase.expectErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
			}
			if actual.Cmp(testCase.output) != 0 {
				t.Errorf("%s: expected %s but got %s", testCase.name, testCase.output.String(), actual.String())
			}
		})
	}
}

func TestResourceVersionRoundTripping(t *testing.T) {
	var testCases = []struct {
		name  string
		input *apd.Decimal
	}{
		{
			name:  "no logical clock",
			input: newFromStringOrDie("1640995200000000001"),
		},
		{
			name:  "large physical clock",
			input: newFromStringOrDie(strconv.Itoa(1<<20 + 1640995200000000000)),
		},
		{
			name:  "with logical clock",
			input: newFromStringOrDie("1640995200000000001.0000000001"),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rv, err := toResourceVersion(testCase.input)
			if err != nil {
				t.Fatalf("error getting rv: %v", err)
			}
			hlc, err := toHybridLogicalClock(rv)
			if err != nil {
				t.Fatalf("error getting hlc: %v", err)
			}
			if hlc.Cmp(testCase.input) != 0 {
				t.Fatalf("invalid round-tripped hlc: before: %s, after: %s", testCase.input.String(), hlc.String())
			}
		})
	}
}