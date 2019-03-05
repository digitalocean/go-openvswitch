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

func Test_parseMatch(t *testing.T) {
	var tests = []struct {
		desc    string
		s       string
		final   string
		m       Match
		invalid bool
	}{
		{
			s:       "foo=bar",
			invalid: true,
		},
		{
			s:       "arp_sha=foo",
			invalid: true,
		},
		{
			s: "arp_sha=de:ad:be:ef:de:ad",
			m: ARPSourceHardwareAddress(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}),
		},
		{
			s:       "arp_tha=foo",
			invalid: true,
		},
		{
			s: "arp_tha=de:ad:be:ef:de:ad",
			m: ARPTargetHardwareAddress(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}),
		},
		{
			s: "arp_spa=192.168.1.1",
			m: ARPSourceProtocolAddress("192.168.1.1"),
		},
		{
			s: "arp_tpa=192.168.1.1",
			m: ARPTargetProtocolAddress("192.168.1.1"),
		},
		{
			s:       "ct_state=+hi",
			invalid: true,
		},
		{
			s: "ct_state=+trk-new",
			m: ConnectionTrackingState(
				SetState(CTStateTracked),
				UnsetState(CTStateNew),
			),
		},
		{
			s:       "tcp_flags=+omg",
			invalid: true,
		},
		{
			s: "tcp_flags=+syn-ack",
			m: TCPFlags(
				SetTCPFlag(TCPFlagSYN),
				UnsetTCPFlag(TCPFlagACK),
			),
		},
		{
			s: "dl_src=de:ad:be:ef:de:ad",
			m: DataLinkSource("de:ad:be:ef:de:ad"),
		},
		{
			s: "dl_dst=de:ad:be:ef:de:ad",
			m: DataLinkDestination("de:ad:be:ef:de:ad"),
		},
		{
			s:       "dl_vlan=foo",
			invalid: true,
		},
		{
			s:       "dl_vlan=0xff",
			invalid: true,
		},
		{
			s: "dl_vlan=10",
			m: DataLinkVLAN(10),
		},
		{
			s: "dl_vlan=0xffff",
			m: DataLinkVLAN(VLANNone),
		},
		{
			s:       "dl_vlan_pcp=foo",
			invalid: true,
		},
		{
			s:       "dl_vlan_pcp=0x0f",
			invalid: true,
		},
		{
			s: "dl_vlan_pcp=0",
			m: DataLinkVLANPCP(0),
		},
		{
			s: "dl_vlan_pcp=7",
			m: DataLinkVLANPCP(7),
		},
		{
			s:       "dl_type=foo",
			invalid: true,
		},
		{
			s: "dl_type=0x0806",
			m: DataLinkType(0x0806),
		},
		{
			s:       "icmp_type=256",
			invalid: true,
		},
		{
			s: "icmp_type=1",
			m: ICMPType(1),
		},
		{
			s: "icmp_code=1",
			m: ICMPCode(1),
		},
		{
			s: "ipv6_src=2001:db8::1",
			m: IPv6Source("2001:db8::1"),
		},
		{
			s: "ipv6_dst=2001:db8::1",
			m: IPv6Destination("2001:db8::1"),
		},
		{
			s: "icmpv6_type=135",
			m: ICMP6Type(135),
		},
		{
			s: "icmpv6_code=3",
			m: ICMP6Code(3),
		},
		{
			s:       "nd_sll=foo",
			invalid: true,
		},
		{
			s: "nd_sll=de:ad:be:ef:de:ad",
			m: NeighborDiscoverySourceLinkLayer(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}),
		},
		{
			s: "nd_tll=de:ad:be:ef:de:ad",
			m: NeighborDiscoveryTargetLinkLayer(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}),
		},
		{
			s: "nd_target=2001:db8::1",
			m: NeighborDiscoveryTarget("2001:db8::1"),
		},
		{
			s: "nw_src=192.168.1.1",
			m: NetworkSource("192.168.1.1"),
		},
		{
			s: "nw_dst=192.168.1.1",
			m: NetworkDestination("192.168.1.1"),
		},
		{
			s:       "nw_proto=256",
			invalid: true,
		},
		{
			s: "nw_proto=1",
			m: NetworkProtocol(1),
		},
		{
			s:       "tp_dst=65536",
			invalid: true,
		},
		{
			s: "tp_dst=80",
			m: TransportDestinationPort(80),
		},
		{
			s:       "tp_src=65536",
			invalid: true,
		},
		{
			s: "tp_src=80",
			m: TransportSourcePort(80),
		},
		{
			s:       "vlan_tci=",
			invalid: true,
		},
		{
			s:       "vlan_tci=foo",
			invalid: true,
		},
		{
			s:     "vlan_tci=10",
			final: "vlan_tci=0x000a",
			m:     VLANTCI(10, 0),
		},
		{
			s: "vlan_tci=0x000a",
			m: VLANTCI(10, 0),
		},
		{
			s:       "vlan_tci=10/foo",
			invalid: true,
		},
		{
			s:     "vlan_tci=10/10",
			final: "vlan_tci=0x000a/0x000a",
			m:     VLANTCI(10, 10),
		},
		{
			s: "vlan_tci=0x1000/0x1000",
			m: VLANTCI(0x1000, 0x1000),
		},
		{
			s:       "vlan_tci=10/10/10",
			invalid: true,
		},
		{
			s:       "vlan_tci1=",
			invalid: true,
		},
		{
			s:       "vlan_tci1=foo",
			invalid: true,
		},
		{
			s:     "vlan_tci1=10",
			final: "vlan_tci1=0x000a",
			m:     VLANTCI1(10, 0),
		},
		{
			s: "vlan_tci1=0x000a",
			m: VLANTCI1(10, 0),
		},
		{
			s:       "vlan_tci1=10/foo",
			invalid: true,
		},
		{
			s:     "vlan_tci1=10/10",
			final: "vlan_tci1=0x000a/0x000a",
			m:     VLANTCI1(10, 10),
		},
		{
			s: "vlan_tci1=0x1000/0x1000",
			m: VLANTCI1(0x1000, 0x1000),
		},
		{
			s:       "vlan_tci1=10/10/10",
			invalid: true,
		},
		{
			s:       "ipv6_label=",
			invalid: true,
		},
		{
			s:       "ipv6_label=foo",
			invalid: true,
		},
		{
			s:     "ipv6_label=10",
			final: "ipv6_label=0x0000a",
			m:     IPv6Label(10, 0),
		},
		{
			s: "ipv6_label=0x0000a",
			m: IPv6Label(10, 0),
		},
		{
			s:       "ipv6_label=10/foo",
			invalid: true,
		},
		{
			s:     "ipv6_label=10/10",
			final: "ipv6_label=0x0000a/0x0000a",
			m:     IPv6Label(10, 10),
		},
		{
			s: "ipv6_label=0x01000/0x01000",
			m: IPv6Label(0x1000, 0x1000),
		},
		{
			s:       "ipv6_label=10/10/10",
			invalid: true,
		},
		{
			s:       "arp_op=",
			invalid: true,
		},
		{
			s:       "arp_op=foo",
			invalid: true,
		},
		{
			s:     "arp_op=2",
			final: "arp_op=2",
			m:     ArpOp(2),
		},
		{
			s:       "ct_mark=",
			invalid: true,
		},
		{
			s:       "ct_mark=foo",
			invalid: true,
		},
		{
			s:     "ct_mark=10",
			final: "ct_mark=0x0000000a",
			m:     ConnectionTrackingMark(10, 0),
		},
		{
			s: "ct_mark=0x0000000a",
			m: ConnectionTrackingMark(10, 0),
		},
		{
			s:       "ct_mark=10/foo",
			invalid: true,
		},
		{
			s:     "ct_mark=10/10",
			final: "ct_mark=0x0000000a/0x0000000a",
			m:     ConnectionTrackingMark(10, 10),
		},
		{
			s: "ct_mark=0x00001000/0x00001000",
			m: ConnectionTrackingMark(0x1000, 0x1000),
		},
		{
			s:       "ct_mark=10/10/10",
			invalid: true,
		},
		{
			s: "ct_zone=1",
			m: ConnectionTrackingZone(1),
		},
		{
			s:       "ct_zone=",
			invalid: true,
		},
		{
			s:       "ct_zone=1/1",
			invalid: true,
		},
		{
			s:       "tun_id=",
			invalid: true,
		},
		{
			s:       "tun_id=xyzzy",
			invalid: true,
		},
		{
			s:     "tun_id=0",
			final: "tun_id=0x0",
			m:     TunnelID(0),
		},
		{
			s:     "tun_id=1",
			final: "tun_id=0x1",
			m:     TunnelID(1),
		},
		{
			s: "tun_id=0x135d",
			m: TunnelID(4957),
		},
		{
			s:     "tun_id=0x000000000000000a",
			final: "tun_id=0xa",
			m:     TunnelID(10),
		},
		{
			s:     "tun_id=0x000000000000000a/00000000000000002",
			final: "tun_id=0xa/0x2",
			m:     TunnelIDWithMask(10, 2),
		},
		{
			s: "conj_id=123",
			m: ConjunctionID(123),
		},
		{
			s:       "conj_id=nope",
			invalid: true,
		},
		{
			desc:    "tp_dst out of range 65536/0xffe0",
			s:       "tp_dst=65536/0xffe0",
			invalid: true,
		},
		{
			desc:    "tp_dst out of range 0x10000/0xffe0",
			s:       "tp_dst=0x10000/0xffe0",
			invalid: true,
		},
		{
			desc:    "tp_dst out of range 0xea60/0x10000",
			s:       "tp_dst=0xea60/0x10000",
			invalid: true,
		},
		{
			desc: "tp_dst 0xea60/0xffe0",
			s:    "tp_dst=0xea60/0xffe0",
			m:    TransportDestinationMaskedPort(0xea60, 0xffe0),
		},
		{
			desc:    "tp_dst 0xea60/0xffe0/0xdddd",
			s:       "tp_dst=0xea60/0xffe0/0xdddd",
			invalid: true,
		},
		{
			desc:    "tp_src out of range 65536/0xffe0",
			s:       "tp_src=65536/0xffe0",
			invalid: true,
		},
		{
			desc:    "tp_src out of range 0x10000/0xffe0",
			s:       "tp_src=0x10000/0xffe0",
			invalid: true,
		},
		{
			desc:    "tp_src out of range 0xea60/0x10000",
			s:       "tp_src=0xea60/0x10000",
			invalid: true,
		},
		{
			desc: "tp_src 0xea60/0xffe0",
			s:    "tp_src=0xea60/0xffe0",
			m:    TransportSourceMaskedPort(0xea60, 0xffe0),
		},
		{
			desc:    "tp_src 0xea60/0xffe0/0xdddd",
			s:       "tp_src=0xea60/0xffe0/0xdddd",
			invalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			ss := strings.Split(tt.s, "=")
			if len(ss) != 2 {
				t.Fatalf("malformed match: %q", tt.s)
			}

			m, err := parseMatch(ss[0], ss[1])
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.invalid {
				return
			}

			if !reflect.DeepEqual(tt.m, m) {
				t.Fatalf("unexpected matcher:\n- want: %q\n- got: %q",
					tt.m, m)
			}

			s, err := m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// In some cases, we may want to check a different final
			// output instead of the initial input (e.g. in the case
			// of a match that may be in decimal or hexadecimal).
			want := tt.s
			if tt.final != "" {
				want = tt.final
			}

			if got := string(s); want != got {
				t.Fatalf("unexpected match:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}
