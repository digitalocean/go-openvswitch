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
	"net"
	"reflect"
	"strings"
	"testing"
)

func Test_actionParser(t *testing.T) {
	var tests = []struct {
		name    string
		in      string
		raw     []string
		invalid bool
	}{
		{
			name:    "invalid action",
			in:      "strip_vlan,resubmit(",
			invalid: true,
		},
		{
			name: "one action",
			in:   "strip_vlan",
			raw: []string{
				"strip_vlan",
			},
		},
		{
			name: "two actions",
			in:   "strip_vlan,resubmit(,1)",
			raw: []string{
				"strip_vlan",
				"resubmit(,1)",
			},
		},
		{
			name: "action with nested parentheses",
			in:   "strip_vlan,resubmit(,1),ct(commit,exec(set_field:1->ct_label,set_field:1->ct_mark))",
			raw: []string{
				"strip_vlan",
				"resubmit(,1)",
				"ct(commit,exec(set_field:1->ct_label,set_field:1->ct_mark))",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newActionParser(strings.NewReader(tt.in))
			actions, raw, err := p.Parse()
			if err != nil {
				if tt.invalid {
					return
				}

				t.Fatalf("unexpected error during parsing: %v", err)
			}

			if want, got := tt.raw, raw; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected raw actions:\n- want: %v\n-  got: %v",
					want, got)
			}

			as, err := (&Flow{Actions: actions}).marshalActions()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := raw, as; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected actions after parsing:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func Test_parseAction(t *testing.T) {
	var tests = []struct {
		desc    string
		s       string
		final   string
		a       Action
		invalid bool
	}{
		{
			s:       "foo",
			invalid: true,
		},
		{
			s: "drop",
			a: Drop(),
		},
		{
			s: "flood",
			a: Flood(),
		},
		{
			s: "in_port",
			a: InPort(),
		},
		{
			s: "local",
			a: Local(),
		},
		{
			s: "LOCAL",
			a: Local(),
		},
		{
			s: "normal",
			a: Normal(),
		},
		{
			s: "NORMAL",
			a: Normal(),
		},
		{
			s: "strip_vlan",
			a: StripVLAN(),
		},
		{
			s:       "ct()",
			invalid: true,
		},
		{
			s: "ct(commit)",
			a: ConnectionTracking("commit"),
		},
		{
			s:       "mod_dl_dst:foo",
			invalid: true,
		},
		{
			s: "mod_dl_dst:de:ad:be:ef:de:ad",
			a: ModDataLinkDestination(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}),
		},
		{
			s: "mod_dl_src:de:ad:be:ef:de:ad",
			a: ModDataLinkSource(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}),
		},
		{
			s:       "mod_nw_dst:foo",
			invalid: true,
		},
		{
			s:       "mod_nw_dst:2001:db8::1",
			invalid: true,
		},
		{
			s: "mod_nw_dst:192.168.1.1",
			a: ModNetworkDestination(net.IPv4(192, 168, 1, 1)),
		},
		{
			s:       "mod_nw_src:foo",
			invalid: true,
		},
		{
			s:       "mod_nw_src:2001:db8::1",
			invalid: true,
		},
		{
			s: "mod_nw_src:192.168.1.1",
			a: ModNetworkSource(net.IPv4(192, 168, 1, 1)),
		},
		{
			s:       "mod_tp_dst:foo",
			invalid: true,
		},
		{
			s:       "mod_tp_dst:-1",
			invalid: true,
		},
		{
			s:       "mod_tp_dst:65536",
			invalid: true,
		},
		{
			s: "mod_tp_dst:65535",
			a: ModTransportDestinationPort(65535),
		},
		{
			s:       "mod_tp_src:foo",
			invalid: true,
		},
		{
			s:       "mod_tp_src:-1",
			invalid: true,
		},
		{
			s:       "mod_tp_src:65536",
			invalid: true,
		},
		{
			s: "mod_tp_src:65535",
			a: ModTransportSourcePort(65535),
		},
		{
			s:       "mod_vlan_vid:foo",
			invalid: true,
		},
		{
			s: "mod_vlan_vid:10",
			a: ModVLANVID(10),
		},
		{
			s:       "output:foo",
			invalid: true,
		},
		{
			s: "output:1",
			a: Output(1),
		},
		{
			s:       "resubmit(foo,)",
			invalid: true,
		},
		{
			s:       "resubmit(,bar)",
			invalid: true,
		},
		{
			s:       "resubmit(foo,bar)",
			invalid: true,
		},
		{
			s: "resubmit:4",
			a: ResubmitPort(4),
		},
		{
			s: "resubmit(1,)",
			a: Resubmit(1, 0),
		},
		{
			s: "resubmit(,2)",
			a: Resubmit(0, 2),
		},
		{
			s: "resubmit(1,2)",
			a: Resubmit(1, 2),
		},
		{
			s: "resubmit(,25)",
			a: Resubmit(0, 25),
		},
		{
			s:       "load:->NXM_OF_ARP_OP[]",
			invalid: true,
		},
		{
			s:       "load:0x2->",
			invalid: true,
		},
		{
			s: "load:0x2->NXM_OF_ARP_OP[]",
			a: Load("0x2", "NXM_OF_ARP_OP[]"),
		},
		{
			s:       "set_field:->arp_spa",
			invalid: true,
		},
		{
			s:       "set_field:192.168.1.1->",
			invalid: true,
		},
		{
			s: "set_field:192.168.1.1->arp_spa",
			a: SetField("192.168.1.1", "arp_spa"),
		},
		{
			s: "conjunction(123,1/2)",
			a: Conjunction(123, 1, 2),
		},
		{
			s: "conjunction(123,2/2)",
			a: Conjunction(123, 2, 2),
		},
		{
			s:       "conjunction(123,3/2)",
			invalid: true,
		},
		{
			s:       "conjunxxxxx(123,3/2)",
			invalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			a, err := parseAction(tt.s)
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.invalid {
				return
			}

			s, err := a.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Special case: LOCAL and NORMAL are converted to
			// the lower case counterpart by this package for
			// consistency.
			want := tt.s
			switch want {
			case "LOCAL", "NORMAL":
				want = strings.ToLower(want)
			}

			if tt.final != "" {
				want = tt.final
			}

			if got := string(s); want != got {
				t.Fatalf("unexpected action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}
