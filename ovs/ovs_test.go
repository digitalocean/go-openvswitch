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
	"errors"
	"testing"
)

func TestIsPortNotExist(t *testing.T) {
	var tests = []struct {
		desc string
		err  error
		ok   bool
	}{
		{
			desc: "not type Error",
			err:  errors.New("foo"),
		},
		{
			desc: "type Error, wrong Error.Out",
			err: &Error{
				Out: []byte("bar"),
				Err: errors.New("exit status 1"),
			},
		},
		{
			desc: "type Error, wrong Error.Err",
			err: &Error{
				Out: []byte("ovs-vsctl: no port named foo"),
				Err: errors.New("exit status foo"),
			},
		},
		{
			desc: "ok",
			err: &Error{
				Out: []byte("ovs-vsctl: no port named foo"),
				Err: errors.New("exit status 1"),
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if want, got := tt.ok, IsPortNotExist(tt.err); want != got {
				t.Fatalf("unexpected IsPortNotExist(%v):\n- want: %v\n-  got: %v",
					tt.err, want, got)
			}
		})
	}
}

// errStr is a helper to return the string form of an error, even if the
// error is nil.
func errStr(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
