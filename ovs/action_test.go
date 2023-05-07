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
			a:   All(),
			out: "all",
		},
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

func TestMultipath(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "set multipath OK",
			a:      Multipath("symmetric_l3l4+udp", 1024, "hrw", 2, 0, "reg0"),
			action: "multipath(symmetric_l3l4+udp,1024,hrw,2,0,reg0)",
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

func TestMove(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "move OK",
			a:      Move("nw_src", "nw_dst"),
			action: "move:nw_src->nw_dst",
		},
		{
			desc: "both empty",
			a:    Move("", ""),
			err:  errMoveEmpty,
		},
		{
			desc: "src empty",
			a:    Move("", "nw_dst"),
			err:  errMoveEmpty,
		},
		{
			desc: "dst empty",
			a:    Move("nw_src", ""),
			err:  errMoveEmpty,
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

func TestOutputField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "output field OK",
			a:      OutputField("in_port"),
			action: "output:in_port",
		},
		{
			desc: "empty field",
			a:    OutputField(""),
			err:  errOutputFieldEmpty,
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

func TestLearn(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc: "learn ok",
			a: Learn(&LearnedFlow{
				DeleteLearned:  true,
				FinHardTimeout: 10,
				Matches:        []Match{DataLinkType(0x800)},
				Actions:        []Action{OutputField("in_port"), Load("2", "tp_dst")},
			}),
			action: `learn(priority=0,dl_type=0x0800,table=0,idle_timeout=0,fin_hard_timeout=10,hard_timeout=0,delete_learned,output:in_port,load:2->tp_dst)`,
		},
		{
			desc: "learn ok",
			a: Learn(&LearnedFlow{
				DeleteLearned:  true,
				FinHardTimeout: 10,
				HardTimeout:    30,
				Limit:          10,
				Matches:        []Match{DataLinkType(0x800)},
				Actions:        []Action{OutputField("in_port"), Load("2", "tp_dst")},
			}),
			action: `learn(priority=0,dl_type=0x0800,table=0,idle_timeout=0,fin_hard_timeout=10,hard_timeout=30,limit=10,delete_learned,output:in_port,load:2->tp_dst)`,
		},
		{
			desc: "prohibited learned action, mod_tp_dst",
			a: Learn(&LearnedFlow{
				DeleteLearned:  true,
				FinHardTimeout: 10,
				Matches:        []Match{DataLinkType(0x800)},
				Actions:        []Action{ModTransportDestinationPort(1)},
			}),
			err: errInvalidLearnedActions,
		},
		{
			desc: "nil learned flow",
			a:    Learn(nil),
			err:  errLearnedNil,
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

func TestPopField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "pop OK",
			a:      Pop("NXM_OF_IN_PORT[]"),
			action: "pop:NXM_OF_IN_PORT[]",
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

func TestPushField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "push OK",
			a:      Push("NXM_NX_REG0[]"),
			action: "push:NXM_NX_REG0[]",
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

func TestDecTTLField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "dec_ttl OK",
			a:      DecTTL(),
			action: "dec_ttl",
		},
		{
			desc:   "dec_ttl 1 id",
			a:      DecTTL(1),
			action: "dec_ttl(1)",
		},
		{
			desc:   "dec_ttl 2 ids",
			a:      DecTTL(1, 2),
			action: "dec_ttl(1,2)",
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

func TestControllerField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "controller plan",
			a:      Controller(0, "", 0, "", false),
			action: "controller",
		},
		{
			desc:   "controller userdata",
			a:      Controller(0, "", 0, "00.00.00.04.00.00.00.00", false),
			action: "controller(userdata=00.00.00.04.00.00.00.00)",
		},
		{
			desc:   "controller max_len",
			a:      Controller(10, "", 0, "", false),
			action: "controller:10",
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

func TestCTClearField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "ct_clear ok",
			a:      CTClear(),
			action: "ct_clear",
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

func TestGroupField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "group ok",
			a:      Group(1),
			action: "group:1",
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

func TestBundleField(t *testing.T) {
	var tests = []struct {
		desc   string
		a      Action
		action string
		err    error
	}{
		{
			desc:   "bundle ok",
			a:      Bundle("eth_src", 0, "active_backup", 149),
			action: "bundle(eth_src,0,active_backup,ofport,members:149)",
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
		{
			a: Move("nw_src", "nw_dst"),
			s: `ovs.Move("nw_src", "nw_dst")`,
		},
		{
			a: OutputField("in_port"),
			s: `ovs.OutputField("in_port")`,
		},
		{
			a: Learn(&LearnedFlow{
				DeleteLearned:  true,
				FinHardTimeout: 10,
				HardTimeout:    30,
				Limit:          10,
				Matches:        []Match{DataLinkType(0x800)},
				Actions:        []Action{OutputField("in_port")},
			}),
			s: `ovs.Learn(&ovs.LearnedFlow{Priority:0, InPort:0, Matches:[]ovs.Match{ovs.DataLinkType(0x0800)}, Table:0, IdleTimeout:0, Cookie:0x0, Actions:[]ovs.Action{ovs.OutputField("in_port")}, DeleteLearned:true, FinHardTimeout:10, HardTimeout:30, Limit:10})`,
		},
		{
			a: Push("NXM_NX_REG0[]"),
			s: `ovs.Push("NXM_NX_REG0[]")`,
		},
		{
			a: Pop("NXM_NX_REG0[]"),
			s: `ovs.Pop("NXM_NX_REG0[]")`,
		},
		{
			a: DecTTL(1, 2),
			s: `ovs.DecTTL(1, 2)`,
		},
		{
			a: DecTTL(),
			s: `ovs.DecTTL()`,
		},
		{
			a: CTClear(),
			s: `ovs.CTClear()`,
		},
		{
			a: Group(5),
			s: `ovs.Group(5)`,
		},
		{
			a: Controller(0, "", 0, "00.00.00.0c.00.00.00.00.00.19.00.10.80.00.08.06.0e.27.b1.82.65.0c.00.00.00.19.00.18.80.00.34.10.fe.80.00.00.00.00.00.00.0c.27.b1.ff.fe.82.65.0c.00.19.00.18.80.00.3e.10.fe.80.00.00.00.00.00.00.0c.27.b1.ff.fe.82.65.0c.00.19.00.10.80.00.42.06.0e.27.b1.82.65.0c.00.00.00.1c.00.18.00.20.00.00.00.00.00.00.00.01.1c.04.00.01.1e.04.00.00.00.00.00.19.00.10.00.01.15.08.00.00.00.01.00.00.00.01.ff.ff.00.10.00.00.23.20.00.0e.ff.f8.25.00.00.00", false),
			s: `ovs.Controller(userdata=00.00.00.0c.00.00.00.00.00.19.00.10.80.00.08.06.0e.27.b1.82.65.0c.00.00.00.19.00.18.80.00.34.10.fe.80.00.00.00.00.00.00.0c.27.b1.ff.fe.82.65.0c.00.19.00.18.80.00.3e.10.fe.80.00.00.00.00.00.00.0c.27.b1.ff.fe.82.65.0c.00.19.00.10.80.00.42.06.0e.27.b1.82.65.0c.00.00.00.1c.00.18.00.20.00.00.00.00.00.00.00.01.1c.04.00.01.1e.04.00.00.00.00.00.19.00.10.00.01.15.08.00.00.00.01.00.00.00.01.ff.ff.00.10.00.00.23.20.00.0e.ff.f8.25.00.00.00)`,
		},
		{
			a: Controller(10, "", 0, "", false),
			s: `ovs.Controller(max_len=10)`,
		},
		{
			a: Bundle("eth_src", 0, "active_backup", 149),
			s: `ovs.Bundle(eth_src,0,active_backup,ofport,members:149)`,
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

func Test_formatIntArr(t *testing.T) {
	type arg struct {
		arr []int
		sep string
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{
			"empty arr",
			arg{arr: []int{}, sep: ","},
			"",
		},
		{
			"1 item",
			arg{arr: []int{1}, sep: ","},
			"1",
		},
		{
			"2 item",
			arg{arr: []int{1, 2}, sep: ","},
			"1,2",
		},
		{
			"2 item with space",
			arg{arr: []int{1, 2}, sep: ", "},
			"1, 2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatIntArr(tt.arg.arr, tt.arg.sep); got != tt.want {
				t.Errorf("formatIntArr() = %v, want %v", got, tt.want)
			}
		})
	}
}
