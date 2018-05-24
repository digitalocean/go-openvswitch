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
	"bytes"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	var tests = []struct {
		desc    string
		options []OptionFunc
		c       *Client
	}{
		{
			desc: "no options",
			c: &Client{
				flags:      make([]string, 0),
				ofctlFlags: make([]string, 0),
				debug:      false,
			},
		},
		{
			desc:    "Timeout(2)",
			options: []OptionFunc{Timeout(2)},
			c: &Client{
				flags:      []string{"--timeout=2"},
				ofctlFlags: make([]string, 0),
				debug:      false,
			},
		},
		{
			desc:    "Debug(true)",
			options: []OptionFunc{Debug(true)},
			c: &Client{
				flags:      make([]string, 0),
				ofctlFlags: make([]string, 0),
				debug:      true,
			},
		},
		{
			desc:    "FlowFormat(FlowFormatNXMTableID)",
			options: []OptionFunc{FlowFormat(FlowFormatNXMTableID)},
			c: &Client{
				flags:      make([]string, 0),
				ofctlFlags: []string{"--flow-format=NXM+table_id"},
			},
		},
		{
			desc:    "Protocols([]string{ProtocolOpenFlow14})",
			options: []OptionFunc{Protocols([]string{ProtocolOpenFlow14})},
			c: &Client{
				flags:      make([]string, 0),
				ofctlFlags: []string{"--protocols=OpenFlow14"},
			},
		},
		{
			desc: "Timeout(5), Debug(true)",
			options: []OptionFunc{
				Timeout(5),
				Debug(true),
			},
			c: &Client{
				flags:      []string{"--timeout=5"},
				ofctlFlags: make([]string, 0),
				debug:      true,
			},
		},
		{
			desc: "Sudo()",
			options: []OptionFunc{
				Sudo(),
			},
			c: &Client{
				flags:      make([]string, 0),
				ofctlFlags: make([]string, 0),
				sudo:       true,
			},
		},
		{
			desc: "SetSSLParam(pkey, cert, cacert)",
			options: []OptionFunc{
				SetSSLParam("privkey.pem", "cert.pem", "cacert.pem"),
			},
			c: &Client{
				flags:      make([]string, 0),
				ofctlFlags: []string{"--private-key=privkey.pem", "--certificate=cert.pem", "--ca-cert=cacert.pem"},
			},
		},
		{
			desc: "SetTCPParam(addr)",
			options: []OptionFunc{
				SetTCPParam("127.0.0.1:6640"),
			},
			c: &Client{
				flags:      []string{"--db=tcp:127.0.0.1:6640"},
				ofctlFlags: make([]string, 0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			c := New(tt.options...)

			if want, got := tt.c.flags, c.flags; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Client.flags:\n- want: %v\n-  got: %v",
					want, got)
			}

			if want, got := tt.c.debug, c.debug; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Client.debug:\n- want: %v\n-  got: %v",
					want, got)
			}
			if want, got := tt.c.sudo, c.sudo; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Client.sudo:\n- want: %v\n-  got: %v",
					want, got)
			}

			if want, got := tt.c.ofctlFlags, c.ofctlFlags; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Client.ofctlFlags:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func Test_shellPipe(t *testing.T) {
	b := bytes.TrimSpace([]byte(`
foo
bar
baz
`))

	// stdin pipe must be consumed.  This test will hang if broken.
	buf := bytes.NewBuffer(b)
	out, err := shellPipe(buf, "cat", "-")
	if err != nil {
		t.Fatalf("failed to pipe to cat: %v", err)
	}

	if want, got := b, out; !bytes.Equal(want, got) {
		t.Fatalf("unexpected bytes:\n- want: %v\n-  got: %v",
			want, got)
	}
}

// testClient creates a new Client with the specified OptionFuncs applied and
// using the specified ExecFunc.
func testClient(options []OptionFunc, fn ExecFunc) *Client {
	options = append(options, Exec(fn))
	c := New(options...)
	return c
}
