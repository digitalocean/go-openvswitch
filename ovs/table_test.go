// Copyright 2017 DigitalOcean.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ovs

import (
	"reflect"
	"strconv"
	"testing"
)

func TestTableUnmarshalText(t *testing.T) {
	var tests = []struct {
		desc string
		s    string
		tb   *Table
		err  error
	}{
		{
			desc: "empty string",
			err:  ErrInvalidTable,
		},
		{
			desc: "too few fields",
			s: `
				0: classifier: wild=0x3fffff, max=1000000, active=0
				               lookup=0,
			`,
			err: ErrInvalidTable,
		},
		{
			desc: "too many fields",
			s: `
				1: table1 : wild=0x3fffff, max=1000000, active=0
				            lookup=0, matched=0, foo=0
			`,
			err: ErrInvalidTable,
		},
		{
			desc: "invalid integer ID",
			s: `
				foo: classifier: wild=0x3fffff, max=1000000, active=0
				               lookup=0, matched=0
			`,
			err: &strconv.NumError{
				Func: "ParseInt",
				Num:  "foo",
				Err:  strconv.ErrSyntax,
			},
		},
		{
			desc: "broken key=value pair",
			s: `
				0: classifier: wild 0x3fffff, max=1000000, active=0
				               lookup=0, matched=0
			`,
			err: ErrInvalidTable,
		},
		{
			desc: "invalid integer max",
			s: `
				0: classifier: wild=0x3fffff, max=foo, active=0
				               lookup=0, matched=0
			`,
			err: &strconv.NumError{
				Func: "ParseUint",
				Num:  "foo",
				Err:  strconv.ErrSyntax,
			},
		},
		{
			desc: "OK classifier table",
			s: `
				0: classifier: wild=0x3fffff, max=1000000, active=1
				               lookup=2, matched=3
			`,
			tb: &Table{
				ID:      0,
				Name:    "classifier",
				Wild:    "0x3fffff",
				Max:     1000000,
				Active:  1,
				Lookup:  2,
				Matched: 3,
			},
		},
		{
			desc: "OK table",
			s: `
				1: table1 : wild=0x3fffff, max=1000000, active=1
				            lookup=2, matched=3
			`,
			tb: &Table{
				ID:      1,
				Name:    "table1",
				Wild:    "0x3fffff",
				Max:     1000000,
				Active:  1,
				Lookup:  2,
				Matched: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tb := new(Table)
			err := tb.UnmarshalText([]byte(tt.s))

			if want, got := errStr(tt.err), errStr(err); want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.tb, tb; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Table:\n- want: %#v\n-  got: %#v",
					want, got)
			}
		})
	}
}
