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
	"testing"
)

func TestActionConstants(t *testing.T) {
	var tests = []struct {
		a   Action
		out string
	}{
		{
			a:   Drop(),
			out: "drop",
		},
		{
			a:   Flood(),
			out: "flood",
		},
		{
			a:   InPort(),
			out: "in_port",
		},
		{
			a:   Local(),
			out: "local",
		},
		{
			a:   Normal(),
			out: "normal",
		},
		{
			a:   StripVLAN(),
			out: "strip_vlan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.out, func(t *testing.T) {
			out, err := tt.a.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionCT(t *testing.T) {
	var tests = []struct {
		desc   string
		args   string
		action string
		err    error
	}{
		{
			desc: "no arguments",
			err:  errCTNoArguments,
		},
		{
			desc:   "OK",
			args:   "commit,exec(set_field:1->ct_label,set_field:1->ct_mark)",
			action: "ct(commit,exec(set_field:1->ct_label,set_field:1->ct_mark))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := ConnectionTracking(tt.args).MarshalText()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.action, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionModDataLink(t *testing.T) {
	var tests = []struct {
		desc    string
		a       Action
		out     string
		invalid bool
	}{
		{
			desc:    "destination too short",
			a:       ModDataLinkDestination(net.HardwareAddr{0xde}),
			invalid: true,
		},
		{
			desc:    "destination too long",
			a:       ModDataLinkDestination(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe}),
			invalid: true,
		},
		{
			desc:    "source too short",
			a:       ModDataLinkSource(net.HardwareAddr{0xde}),
			invalid: true,
		},
		{
			desc:    "source too long",
			a:       ModDataLinkSource(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe}),
			invalid: true,
		},
		{
			desc: "destination OK",
			a:    ModDataLinkDestination(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}),
			out:  "mod_dl_dst:de:ad:be:ef:de:ad",
		},
		{
			desc: "source OK",
			a:    ModDataLinkSource(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}),
			out:  "mod_dl_src:de:ad:be:ef:de:ad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := tt.a.MarshalText()
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionModNetwork(t *testing.T) {
	var tests = []struct {
		desc    string
		a       Action
		out     string
		invalid bool
	}{
		{
			desc:    "destination bad",
			a:       ModNetworkDestination(net.IP{0xff}),
			invalid: true,
		},
		{
			desc:    "destination IPv6",
			a:       ModNetworkDestination(net.ParseIP("2001:db8::1")),
			invalid: true,
		},
		{
			desc:    "source bad",
			a:       ModNetworkSource(net.IP{0xff}),
			invalid: true,
		},
		{
			desc:    "source IPv6",
			a:       ModNetworkSource(net.ParseIP("2001:db8::1")),
			invalid: true,
		},
		{
			desc: "destination OK",
			a:    ModNetworkDestination(net.IPv4(192, 168, 1, 1)),
			out:  "mod_nw_dst:192.168.1.1",
		},
		{
			desc: "source OK",
			a:    ModNetworkSource(net.IPv4(192, 168, 1, 1)),
			out:  "mod_nw_src:192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := tt.a.MarshalText()
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionModTransportPort(t *testing.T) {
	var tests = []struct {
		desc string
		a    Action
		out  string
	}{
		{
			desc: "destination port OK",
			a:    ModTransportDestinationPort(65535),
			out:  "mod_tp_dst:65535",
		},
		{
			desc: "source port OK",
			a:    ModTransportSourcePort(65535),
			out:  "mod_tp_src:65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, _ := tt.a.MarshalText()

			if want, got := tt.out, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionModVLANVID(t *testing.T) {
	var tests = []struct {
		desc   string
		vid    int
		action string
		err    error
	}{
		{
			desc: "VLAN VID too small",
			vid:  -1,
			err:  errInvalidVLANVID,
		},
		{
			desc: "VLAN VID too large",
			vid:  4096,
			err:  errInvalidVLANVID,
		},
		{
			desc:   "OK",
			vid:    10,
			action: "mod_vlan_vid:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := ModVLANVID(tt.vid).MarshalText()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.action, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionOutput(t *testing.T) {
	var tests = []struct {
		desc   string
		port   int
		action string
		err    error
	}{
		{
			desc: "port -1",
			port: -1,
			err:  errOutputNegativePort,
		},
		{
			desc:   "port 10",
			port:   10,
			action: "output:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := Output(tt.port).MarshalText()

			if want, got := errStr(tt.err), errStr(err); want != got {
				t.Fatalf("unexpected error:\n- want: %q\n-  got: %q",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.action, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionResubmit(t *testing.T) {
	var tests = []struct {
		desc   string
		port   int
		table  int
		action string
		err    error
	}{
		{
			desc: "both port and table zero",
			err:  errResubmitPortTableZero,
		},
		{
			desc:   "port zero",
			table:  1,
			action: "resubmit(,1)",
		},
		{
			desc:   "table zero",
			port:   1,
			action: "resubmit(1,)",
		},
		{
			desc:   "both port and table non-zero",
			port:   1,
			table:  2,
			action: "resubmit(1,2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := Resubmit(tt.port, tt.table).MarshalText()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.action, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionResubmitPort(t *testing.T) {
	var tests = []struct {
		desc   string
		port   int
		action string
		err    error
	}{
		{
			desc: "invalid port",
			port: -1,
			err:  errResubmitPortInvalid,
		},
		{
			desc:   "port zero",
			port:   0,
			action: "resubmit:0",
		},
		{
			desc:   "port 1",
			port:   1,
			action: "resubmit:1",
		},
		{
			desc:   "max port (0xfffffeff)",
			port:   0xfffffeff,
			action: "resubmit:4294967039",
		},
		{
			desc: "max port+1 (0xfffffeff)",
			port: 0xffffff00,
			err:  errResubmitPortInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := ResubmitPort(tt.port).MarshalText()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.action, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionLoadSetField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc: "set field both empty",
			a:    SetField("", ""),
			err:  errLoadSetFieldZero,
		},
		{
			desc: "set field value empty",
			a:    SetField("", "arp_spa"),
			err:  errLoadSetFieldZero,
		},
		{
			desc: "set field field empty",
			a:    SetField("192.168.1.1", ""),
			err:  errLoadSetFieldZero,
		},
		{
			desc: "load both empty",
			a:    Load("", ""),
			err:  errLoadSetFieldZero,
		},
		{
			desc: "load value empty",
			a:    Load("", "NXM_OF_ARP_OP[]"),
			err:  errLoadSetFieldZero,
		},
		{
			desc: "load field empty",
			a:    Load("0x2", ""),
			err:  errLoadSetFieldZero,
		},
		{
			desc:   "set field OK",
			a:      SetField("192.168.1.1", "arp_spa"),
			action: "set_field:192.168.1.1->arp_spa",
		},
		{
			desc:   "load OK",
			a:      Load("0x2", "NXM_OF_ARP_OP[]"),
			action: "load:0x2->NXM_OF_ARP_OP[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := tt.a.MarshalText()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.action, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestSetTunnel(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "set tunnel OK",
			a:      SetTunnel(0xa),
			action: "set_tunnel:0xa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := tt.a.MarshalText()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.action, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestConjunction(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "set conjunction 1/2",
			a:      Conjunction(123, 1, 2),
			action: "conjunction(123,1/2)",
		},
		{
			desc:   "set conjunction 2/2",
			a:      Conjunction(123, 2, 2),
			action: "conjunction(123,2/2)",
		},
		{
			desc: "set conjunction 3/2",
			a:    Conjunction(123, 3, 2),
			err:  errDimensionTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			action, err := tt.a.MarshalText()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.action, string(action); want != got {
				t.Fatalf("unexpected Action:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestActionGoString(t *testing.T) {
	tests := []struct {
		a Action
		s string
	}{
		{
			a: Drop(),
			s: `ovs.Drop()`,
		},
		{
			a: Flood(),
			s: `ovs.Flood()`,
		},
		{
			a: Local(),
			s: `ovs.Local()`,
		},
		{
			a: Normal(),
			s: `ovs.Normal()`,
		},
		{
			a: StripVLAN(),
			s: `ovs.StripVLAN()`,
		},
		{
			a: ConnectionTracking("commit"),
			s: `ovs.ConnectionTracking("commit")`,
		},
		{
			a: ModDataLinkDestination(mustParseMAC("de:ad:be:ef:de:ad")),
			s: `ovs.ModDataLinkDestination(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad})`,
		},
		{
			a: ModDataLinkSource(mustParseMAC("de:ad:be:ef:de:ad")),
			s: `ovs.ModDataLinkSource(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad})`,
		},
		{
			a: ModNetworkDestination(net.IPv4(192, 168, 1, 1)),
			s: `ovs.ModNetworkDestination(net.IPv4(192, 168, 1, 1))`,
		},
		{
			a: ModNetworkSource(net.IPv4(192, 168, 1, 1)),
			s: `ovs.ModNetworkSource(net.IPv4(192, 168, 1, 1))`,
		},
		{
			a: ModVLANVID(10),
			s: `ovs.ModVLANVID(10)`,
		},
		{
			a: Output(1),
			s: `ovs.Output(1)`,
		},
		{
			a: Resubmit(0, 10),
			s: `ovs.Resubmit(0, 10)`,
		},
		{
			a: Load("0x2", "NXM_OF_ARP_OP[]"),
			s: `ovs.Load("0x2", "NXM_OF_ARP_OP[]")`,
		},
		{
			a: SetField("192.168.1.1", "arp_spa"),
			s: `ovs.SetField("192.168.1.1", "arp_spa")`,
		},
		{
			a: SetTunnel(10),
			s: `ovs.SetTunnel(0xa)`,
		},
		{
			a: Conjunction(123, 1, 2),
			s: `ovs.Conjunction(123, 1, 2)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if want, got := tt.s, tt.a.GoString(); want != got {
				t.Fatalf("unexpected Action Go syntax:\n- want: %v\n-  got: %v", want, got)
			}
		})
	}
}
