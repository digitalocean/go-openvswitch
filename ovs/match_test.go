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
	"fmt"
	"net"
	"reflect"
	"testing"
)

func TestMatchDataLink(t *testing.T) {
	var tests = []struct {
		desc    string
		m       Match
		out     string
		invalid bool
	}{
		{
			desc:    "source hardware address invalid",
			m:       DataLinkSource("foo"),
			invalid: true,
		},
		{
			desc:    "source hardware address invalid length",
			m:       DataLinkSource("de:ad:be:ef:de:ad:be:ef"),
			invalid: true,
		},
		{
			desc:    "destination hardware address invalid",
			m:       DataLinkDestination("foo"),
			invalid: true,
		},
		{
			desc:    "destination hardware address invalid length",
			m:       DataLinkDestination("de:ad:be:ef:de:ad:be:ef"),
			invalid: true,
		},
		{
			desc:    "source wildcard address invalid",
			m:       DataLinkSource("de:ad:be:ef:de:ad/foo"),
			invalid: true,
		},
		{
			desc:    "source wildcard address invalid length",
			m:       DataLinkSource("de:ad:be:ef:de:ad/00:11:22:33:44:55:66:77"),
			invalid: true,
		},
		{
			desc:    "destination wildcard address invalid",
			m:       DataLinkDestination("de:ad:be:ef:de:ad/foo"),
			invalid: true,
		},
		{
			desc:    "destination wildcard address invalid length",
			m:       DataLinkDestination("de:ad:be:ef:de:ad/00:11:22:33:44:55:66:77"),
			invalid: true,
		},
		{
			desc: "source hardware address",
			m:    DataLinkSource("de:ad:be:ef:de:ad"),
			out:  "dl_src=de:ad:be:ef:de:ad",
		},
		{
			desc: "destination hardware address",
			m:    DataLinkDestination("de:ad:be:ef:de:ad"),
			out:  "dl_dst=de:ad:be:ef:de:ad",
		},
		{
			desc: "source hardware address and wildcard",
			m:    DataLinkSource("de:ad:be:ef:de:ad/ff:ff:ff:ff:ff:ff"),
			out:  "dl_src=de:ad:be:ef:de:ad/ff:ff:ff:ff:ff:ff",
		},
		{
			desc: "destination hardware address and wildcard",
			m:    DataLinkDestination("de:ad:be:ef:de:ad/ff:ff:ff:ff:ff:ff"),
			out:  "dl_dst=de:ad:be:ef:de:ad/ff:ff:ff:ff:ff:ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchDataLinkType(t *testing.T) {
	var tests = []struct {
		desc      string
		etherType uint16
		out       string
	}{
		{
			desc:      "ARP",
			etherType: 0x0806,
			out:       "dl_type=0x0806",
		},
		{
			desc:      "decimal 10",
			etherType: 10,
			out:       "dl_type=0x000a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := DataLinkType(tt.etherType).MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchDataLinkVLAN(t *testing.T) {
	var tests = []struct {
		desc    string
		vlan    int
		out     string
		invalid bool
	}{
		{
			desc:    "negative VLAN",
			vlan:    -1,
			invalid: true,
		},
		{
			desc:    "too large VLAN",
			vlan:    5000,
			invalid: true,
		},
		{
			desc: "no VLAN",
			vlan: VLANNone,
			out:  "dl_vlan=0xffff",
		},
		{
			desc: "VLAN 10",
			vlan: 10,
			out:  "dl_vlan=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := DataLinkVLAN(tt.vlan).MarshalText()
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchDataLinkVLANPCP(t *testing.T) {
	var tests = []struct {
		desc    string
		vlanPCP int
		out     string
		invalid bool
	}{
		{
			desc:    "too small VLAN PCP",
			vlanPCP: -1,
			invalid: true,
		},
		{
			desc:    "too large VLAN PCP",
			vlanPCP: 8,
			invalid: true,
		},
		{
			desc:    "minimum VLAN PCP",
			vlanPCP: 0,
			out:     "dl_vlan_pcp=0",
		},
		{
			desc:    "maximum VLAN PCP",
			vlanPCP: 7,
			out:     "dl_vlan_pcp=7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := DataLinkVLANPCP(tt.vlanPCP).MarshalText()
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchIPv4AddressOrCIDR(t *testing.T) {
	var tests = []struct {
		desc    string
		m       Match
		out     string
		invalid bool
	}{
		{
			desc:    "network source invalid",
			m:       NetworkSource("foo"),
			invalid: true,
		},
		{
			desc:    "network destination invalid",
			m:       NetworkDestination("foo"),
			invalid: true,
		},
		{
			desc:    "ARP source protocol invalid",
			m:       ARPSourceProtocolAddress("foo"),
			invalid: true,
		},
		{
			desc:    "ARP target protocol invalid",
			m:       ARPTargetProtocolAddress("foo"),
			invalid: true,
		},
		{
			desc:    "network source IPv6 address (invalid)",
			m:       NetworkSource("2001:db8::1"),
			invalid: true,
		},
		{
			desc:    "network destination IPv6 address (invalid)",
			m:       NetworkDestination("2001:db8::1"),
			invalid: true,
		},
		{
			desc:    "ARP source protocol IPv6 address (invalid)",
			m:       ARPSourceProtocolAddress("2001:db8::1"),
			invalid: true,
		},
		{
			desc:    "ARP target protocol IPv6 address (invalid)",
			m:       ARPTargetProtocolAddress("2001:db8::1"),
			invalid: true,
		},
		{
			desc:    "network source IPv6 CIDR block (invalid)",
			m:       NetworkSource("2001:db8::1/128"),
			invalid: true,
		},
		{
			desc:    "network destination IPv6 CIDR block (invalid)",
			m:       NetworkDestination("2001:db8::1/128"),
			invalid: true,
		},
		{
			desc:    "ARP source protocol IPv6 CIDR block (invalid)",
			m:       ARPSourceProtocolAddress("2001:db8::1/128"),
			invalid: true,
		},
		{
			desc:    "ARP target protocol IPv6 CIDR block (invalid)",
			m:       ARPTargetProtocolAddress("2001:db8::1/128"),
			invalid: true,
		},
		{
			desc: "network source IPv4 address",
			m:    NetworkSource("192.168.1.1"),
			out:  "nw_src=192.168.1.1",
		},
		{
			desc: "network destination IPv4 address",
			m:    NetworkDestination("192.168.1.1"),
			out:  "nw_dst=192.168.1.1",
		},
		{
			desc: "ARP source protocol IPv4 address",
			m:    ARPSourceProtocolAddress("192.168.1.1"),
			out:  "arp_spa=192.168.1.1",
		},
		{
			desc: "ARP target protocol IPv4 address",
			m:    ARPTargetProtocolAddress("192.168.1.1"),
			out:  "arp_tpa=192.168.1.1",
		},
		{
			desc: "network source IPv4 CIDR",
			m:    NetworkSource("192.168.1.0/24"),
			out:  "nw_src=192.168.1.0/24",
		},
		{
			desc: "network destination IPv4 CIDR",
			m:    NetworkDestination("192.168.1.0/24"),
			out:  "nw_dst=192.168.1.0/24",
		},
		{
			desc: "ARP source protocol IPv4 CIDR",
			m:    ARPSourceProtocolAddress("192.168.1.0/24"),
			out:  "arp_spa=192.168.1.0/24",
		},
		{
			desc: "ARP target protocol IPv4 CIDR",
			m:    ARPTargetProtocolAddress("192.168.1.0/24"),
			out:  "arp_tpa=192.168.1.0/24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchIPv6AddressOrCIDR(t *testing.T) {
	var tests = []struct {
		desc    string
		m       Match
		out     string
		invalid bool
	}{
		{
			desc:    "source invalid",
			m:       IPv6Source("foo"),
			invalid: true,
		},
		{
			desc:    "destination invalid",
			m:       IPv6Destination("foo"),
			invalid: true,
		},
		{
			desc:    "neighbor discovery target invalid",
			m:       NeighborDiscoveryTarget("foo"),
			invalid: true,
		},
		{
			desc:    "source IPv4 address (invalid)",
			m:       IPv6Source("192.168.1.1"),
			invalid: true,
		},
		{
			desc:    "destination IPv4 address (invalid)",
			m:       IPv6Destination("192.168.1.1"),
			invalid: true,
		},
		{
			desc:    "neighbor discovery target IPv4 address (invalid)",
			m:       NeighborDiscoveryTarget("192.168.1.1"),
			invalid: true,
		},
		{
			desc:    "source IPv4 CIDR block (invalid)",
			m:       IPv6Source("192.168.1.0/24"),
			invalid: true,
		},
		{
			desc:    "destination IPv4 CIDR block (invalid)",
			m:       IPv6Destination("192.168.1.0/24"),
			invalid: true,
		},
		{
			desc:    "neighbor discovery target IPv4 CIDR block (invalid)",
			m:       NeighborDiscoveryTarget("192.168.1.0/24"),
			invalid: true,
		},
		{
			desc: "network source IPv6 address",
			m:    IPv6Source("2001:db8::1"),
			out:  "ipv6_src=2001:db8::1",
		},
		{
			desc: "network destination IPv6 address",
			m:    IPv6Destination("2001:db8::1"),
			out:  "ipv6_dst=2001:db8::1",
		},
		{
			desc: "neighbor discovery target IPv6 address",
			m:    NeighborDiscoveryTarget("2001:db8::1"),
			out:  "nd_target=2001:db8::1",
		},
		{
			desc: "network source IPv6 CIDR",
			m:    IPv6Source("2001:db8::1/128"),
			out:  "ipv6_src=2001:db8::1/128",
		},
		{
			desc: "network destination IPv6 CIDR",
			m:    IPv6Destination("2001:db8::1/128"),
			out:  "ipv6_dst=2001:db8::1/128",
		},
		{
			desc: "neighbor discovery target IPv6 CIDR",
			m:    NeighborDiscoveryTarget("2001:db8::1/128"),
			out:  "nd_target=2001:db8::1/128",
		},
		{
			desc: "network source IPv6 CIDR (fixed bug: matches original address)",
			m:    IPv6Source("2001:db8::a001/124"),
			out:  "ipv6_src=2001:db8::a001/124",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchICMPType(t *testing.T) {
	var tests = []struct {
		desc string
		typ  uint8
		out  string
	}{
		{
			desc: "echo reply",
			typ:  0,
			out:  "icmp_type=0",
		},
		{
			desc: "destination unreachable",
			typ:  3,
			out:  "icmp_type=3",
		},
		{
			desc: "echo",
			typ:  8,
			out:  "icmp_type=8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := ICMPType(tt.typ).MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchICMPCode(t *testing.T) {
	var tests = []struct {
		desc string
		code uint8
		out  string
	}{
		{
			desc: "host unreachable",
			code: 1,
			out:  "icmp_code=1",
		},
		{
			desc: "protocol unreachable",
			code: 2,
			out:  "icmp_code=2",
		},
		{
			desc: "port unreachable",
			code: 3,
			out:  "icmp_code=3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := ICMPCode(tt.code).MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchICMP6Type(t *testing.T) {
	var tests = []struct {
		desc string
		typ  uint8
		out  string
	}{
		{
			desc: "destination unreachable",
			typ:  1,
			out:  "icmpv6_type=1",
		},
		{
			desc: "neighbor solicitation",
			typ:  135,
			out:  "icmpv6_type=135",
		},
		{
			desc: "neighbor advertisement",
			typ:  136,
			out:  "icmpv6_type=136",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := ICMP6Type(tt.typ).MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchICMP6Code(t *testing.T) {
	var tests = []struct {
		desc string
		code uint8
		out  string
	}{
		{
			desc: "no route to destination",
			code: 0,
			out:  "icmpv6_code=0",
		},
		{
			desc: "address unreachable",
			code: 3,
			out:  "icmpv6_code=3",
		},
		{
			desc: "port unreachable",
			code: 4,
			out:  "icmpv6_code=4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := ICMP6Code(tt.code).MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchNetworkProtocol(t *testing.T) {
	var tests = []struct {
		desc string
		num  uint8
		out  string
	}{
		{
			desc: "ICMP",
			num:  1,
			out:  "nw_proto=1",
		},
		{
			desc: "TCP",
			num:  6,
			out:  "nw_proto=6",
		},
		{
			desc: "ICMPv6",
			num:  58,
			out:  "nw_proto=58",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := NetworkProtocol(tt.num).MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchConjunctionID(t *testing.T) {
	var tests = []struct {
		desc string
		num  uint32
		out  string
	}{
		{
			desc: "ID 1",
			num:  1,
			out:  "conj_id=1",
		},
		{
			desc: "ID 11111",
			num:  11111,
			out:  "conj_id=11111",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := ConjunctionID(tt.num).MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchConnectionTrackingState(t *testing.T) {
	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "new connection",
			m: ConnectionTrackingState(
				SetState(CTStateNew),
			),
			out: "ct_state=+new",
		},
		{
			desc: "connection in tracker that is not new",
			m: ConnectionTrackingState(
				SetState(CTStateNew),
				UnsetState(CTStateTracked),
			),
			out: "ct_state=+new-trk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchTCPFlags(t *testing.T) {
	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "syn flag set",
			m: TCPFlags(
				SetTCPFlag(TCPFlagSYN),
			),
			out: "tcp_flags=+syn",
		},
		{
			desc: "syn flag set, ack flag not set",
			m: TCPFlags(
				SetTCPFlag(TCPFlagSYN),
				UnsetTCPFlag(TCPFlagACK),
			),
			out: "tcp_flags=+syn-ack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchEthernetHardwareAddress(t *testing.T) {
	var tests = []struct {
		desc    string
		m       Match
		out     string
		invalid bool
	}{
		{
			desc:    "ARP source hardware address invalid length",
			m:       ARPSourceHardwareAddress(mustParseMAC("de:ad:be:ef:de:ad:be:ef")),
			invalid: true,
		},
		{
			desc:    "ARP target hardware address invalid length",
			m:       ARPTargetHardwareAddress(mustParseMAC("de:ad:be:ef:de:ad:be:ef")),
			invalid: true,
		},
		{
			desc:    "ND source link layer address invalid length",
			m:       NeighborDiscoverySourceLinkLayer(mustParseMAC("de:ad:be:ef:de:ad:be:ef")),
			invalid: true,
		},
		{
			desc:    "ND target link layer address invalid length",
			m:       NeighborDiscoveryTargetLinkLayer(mustParseMAC("de:ad:be:ef:de:ad:be:ef")),
			invalid: true,
		},
		{
			desc: "ARP source hardware address",
			m:    ARPSourceHardwareAddress(mustParseMAC("de:ad:be:ef:de:ad")),
			out:  "arp_sha=de:ad:be:ef:de:ad",
		},
		{
			desc: "ARP target hardware address",
			m:    ARPTargetHardwareAddress(mustParseMAC("de:ad:be:ef:de:ad")),
			out:  "arp_tha=de:ad:be:ef:de:ad",
		},
		{
			desc: "ND source link layer address",
			m:    NeighborDiscoverySourceLinkLayer(mustParseMAC("de:ad:be:ef:de:ad")),
			out:  "nd_sll=de:ad:be:ef:de:ad",
		},
		{
			desc: "ND target link layer address",
			m:    NeighborDiscoveryTargetLinkLayer(mustParseMAC("de:ad:be:ef:de:ad")),
			out:  "nd_tll=de:ad:be:ef:de:ad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil && !tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchTransport(t *testing.T) {
	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "source port 80",
			m:    TransportSourcePort(80),
			out:  "tp_src=80",
		},
		{
			desc: "source port 65535",
			m:    TransportSourcePort(65535),
			out:  "tp_src=65535",
		},
		{
			desc: "destination port 22",
			m:    TransportDestinationPort(22),
			out:  "tp_dst=22",
		},
		{
			desc: "destination port 8080",
			m:    TransportDestinationPort(8080),
			out:  "tp_dst=8080",
		},
		{
			desc: "source port range 16/0xfff0 (16-31)",
			m:    TransportSourceMaskedPort(0x10, 0xfff0),
			out:  "tp_src=0x0010/0xfff0",
		},
		{
			desc: "destination port range 16/0xfff0 (16-31)",
			m:    TransportDestinationMaskedPort(0x10, 0xfff0),
			out:  "tp_dst=0x0010/0xfff0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchTransportPortRange(t *testing.T) {
	var tests = []struct {
		desc string
		pr   TransportPortRanger
		m    []Match
	}{
		{
			desc: "destination port range 16-31",
			pr:   TransportDestinationPortRange(16, 31),
			m: []Match{
				TransportDestinationMaskedPort(0x10, 0xfff0),
			},
		},
		{
			desc: "source port range 16-31",
			pr:   TransportSourcePortRange(16, 31),
			m: []Match{
				TransportSourceMaskedPort(0x10, 0xfff0),
			},
		},
		{
			desc: "destination port range 16-32",
			pr:   TransportDestinationPortRange(16, 32),
			m: []Match{
				TransportDestinationMaskedPort(0x10, 0xfff0),
				TransportDestinationMaskedPort(0x20, 0xffff),
			},
		},
		{
			desc: "source port range 16-32",
			pr:   TransportSourcePortRange(16, 32),
			m: []Match{
				TransportSourceMaskedPort(0x10, 0xfff0),
				TransportSourceMaskedPort(0x20, 0xffff),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			m, err := tt.pr.MaskedPorts()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if want, got := tt.m, m; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Match:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchVLANTCI(t *testing.T) {
	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "TCI 10, no mask",
			m:    VLANTCI(10, 0),
			out:  "vlan_tci=0x000a",
		},
		{
			desc: "TCI 0x1000, mask 0x1000 (any VLAN)",
			m:    VLANTCI(0x1000, 0x1000),
			out:  "vlan_tci=0x1000/0x1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchVLANTCI1(t *testing.T) {
	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "TCI1 10, no mask",
			m:    VLANTCI1(0, 0),
			out:  "vlan_tci1=0x0000",
		},
		{
			desc: "TCI1 0x1000, mask 0x1000 (any VLAN)",
			m:    VLANTCI1(0x1000, 0x1000),
			out:  "vlan_tci1=0x1000/0x1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchIPv6Label(t *testing.T) {
	var tests = []struct {
		desc    string
		m       Match
		out     string
		invalid bool
	}{
		{
			desc: "Label 10, no mask",
			m:    IPv6Label(10, 0),
			out:  "ipv6_label=0x0000a",
		},
		{
			desc: "Label 0x1000, mask 0xfffff",
			m:    IPv6Label(0x1000, 0xfffff),
			out:  "ipv6_label=0x01000/0xfffff",
		},
		{
			desc:    "Label uses more than the lower 20 bits",
			m:       IPv6Label(0x100000, 0x000fffff),
			invalid: true,
		},
		{
			desc:    "Mask uses more than the lower 20 bits",
			m:       IPv6Label(0x010000, 0x00ffffff),
			invalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if (err != nil) != tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchARPOP(t *testing.T) {
	var tests = []struct {
		desc    string
		m       Match
		out     string
		invalid bool
	}{
		{
			desc: "Arp op 1",
			m:    ArpOp(1),
			out:  "arp_op=1",
		},
		{
			desc: "Arp op 2",
			m:    ArpOp(2),
			out:  "arp_op=2",
		},
		{
			desc:    "Arp op 0",
			m:       ArpOp(0),
			invalid: true,
		},
		{
			desc:    "Arp op 5",
			m:       ArpOp(5),
			invalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if (err != nil) != tt.invalid {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchConnectionTrackingMark(t *testing.T) {
	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "Mark 10, no mask",
			m:    ConnectionTrackingMark(10, 0),
			out:  "ct_mark=0x0000000a",
		},
		{
			desc: "Mark 0x1000, mask 0x1000",
			m:    ConnectionTrackingMark(0x1000, 0x1000),
			out:  "ct_mark=0x00001000/0x00001000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchConnectionTrackingZone(t *testing.T) {
	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "Zone 1",
			m:    ConnectionTrackingZone(1),
			out:  "ct_zone=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchTunnelID(t *testing.T) {
	var negOne int32 = -1

	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "tunnel ID 0xa",
			m:    TunnelID(0xa),
			out:  "tun_id=0xa",
		},
		{
			desc: "tunnel ID max 64 bit",
			m:    TunnelID(0xffffffffffffffff),
			out:  "tun_id=0xffffffffffffffff",
		},
		{
			desc: "tunnel ID with contiguous mask",
			m:    TunnelIDWithMask(0xa0, 0xf0),
			out:  "tun_id=0xa0/0xf0",
		},
		{
			desc: "tunnel ID with arbitrary mask",
			m:    TunnelIDWithMask(0xa0, 0x5a),
			out:  "tun_id=0xa0/0x5a",
		},
		{
			desc: "tunnel ID with -1 mask is all 1s bit mask",
			m:    TunnelIDWithMask(0xa0, uint64(negOne)),
			out:  "tun_id=0xa0/0xffffffffffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchMetadata(t *testing.T) {
	var tests = []struct {
		desc string
		m    Match
		out  string
	}{
		{
			desc: "metadata 0xa",
			m:    Metadata(0xa),
			out:  "metadata=0xa",
		},
		{
			desc: "metadata max 64 bit",
			m:    Metadata(0xffffffffffffffff),
			out:  "metadata=0xffffffffffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := tt.m.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMatchGoString(t *testing.T) {
	var tests = []struct {
		m Match
		s string
	}{
		{
			m: DataLinkSource("de:ad:be:ef:de:ad"),
			s: `ovs.DataLinkSource("de:ad:be:ef:de:ad")`,
		},
		{
			m: DataLinkDestination("de:ad:be:ef:de:ad"),
			s: `ovs.DataLinkDestination("de:ad:be:ef:de:ad")`,
		},
		{
			m: DataLinkType(0x0806),
			s: `ovs.DataLinkType(0x0806)`,
		},
		{
			m: DataLinkVLAN(10),
			s: `ovs.DataLinkVLAN(10)`,
		},
		{
			m: DataLinkVLAN(VLANNone),
			s: `ovs.DataLinkVLAN(ovs.VLANNone)`,
		},
		{
			m: NetworkSource("192.168.1.1"),
			s: `ovs.NetworkSource("192.168.1.1")`,
		},
		{
			m: NetworkDestination("192.168.1.1"),
			s: `ovs.NetworkDestination("192.168.1.1")`,
		},
		{
			m: NetworkProtocol(255),
			s: `ovs.NetworkProtocol(255)`,
		},
		{
			m: IPv6Source("2001:db8::1"),
			s: `ovs.IPv6Source("2001:db8::1")`,
		},
		{
			m: IPv6Destination("2001:db8::1"),
			s: `ovs.IPv6Destination("2001:db8::1")`,
		},
		{
			m: ICMPType(10),
			s: `ovs.ICMPType(10)`,
		},
		{
			m: ICMPCode(1),
			s: `ovs.ICMPCode(1)`,
		},
		{
			m: ICMP6Type(136),
			s: `ovs.ICMP6Type(136)`,
		},
		{
			m: ICMP6Code(2),
			s: `ovs.ICMP6Code(2)`,
		},
		{
			m: NeighborDiscoveryTarget("2001:db8::1"),
			s: `ovs.NeighborDiscoveryTarget("2001:db8::1")`,
		},
		{
			m: NeighborDiscoverySourceLinkLayer(mustParseMAC("de:ad:be:ef:de:ad")),
			s: `ovs.NeighborDiscoverySourceLinkLayer(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad})`,
		},
		{
			m: NeighborDiscoveryTargetLinkLayer(mustParseMAC("de:ad:be:ef:de:ad")),
			s: `ovs.NeighborDiscoveryTargetLinkLayer(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad})`,
		},
		{
			m: ARPSourceHardwareAddress(mustParseMAC("de:ad:be:ef:de:ad")),
			s: `ovs.ARPSourceHardwareAddress(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad})`,
		},
		{
			m: ARPTargetHardwareAddress(mustParseMAC("de:ad:be:ef:de:ad")),
			s: `ovs.ARPTargetHardwareAddress(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad})`,
		},
		{
			m: ARPSourceProtocolAddress("192.168.1.1"),
			s: `ovs.ARPSourceProtocolAddress("192.168.1.1")`,
		},
		{
			m: ARPTargetProtocolAddress("192.168.1.1"),
			s: `ovs.ARPTargetProtocolAddress("192.168.1.1")`,
		},
		{
			m: TransportSourcePort(80),
			s: `ovs.TransportSourcePort(80)`,
		},
		{
			m: TransportDestinationPort(80),
			s: `ovs.TransportDestinationPort(80)`,
		},
		{
			m: TransportSourceMaskedPort(0x10, 0xfff0),
			s: `ovs.TransportSourceMaskedPort(0x10, 0xfff0)`,
		},
		{
			m: TransportDestinationMaskedPort(0x10, 0xfff0),
			s: `ovs.TransportDestinationMaskedPort(0x10, 0xfff0)`,
		},
		{
			m: VLANTCI(10, 0),
			s: `ovs.VLANTCI(0x000a, 0x0000)`,
		},
		{
			m: VLANTCI(0x1000, 0x1000),
			s: `ovs.VLANTCI(0x1000, 0x1000)`,
		},
		{
			m: VLANTCI1(10, 0),
			s: `ovs.VLANTCI1(0x000a, 0x0000)`,
		},
		{
			m: VLANTCI1(0x1000, 0x1000),
			s: `ovs.VLANTCI1(0x1000, 0x1000)`,
		},
		{
			m: ConnectionTrackingState(
				SetState(CTStateNew),
				UnsetState(CTStateEstablished),
			),
			s: `ovs.ConnectionTrackingState("+new", "-est")`,
		},
		{
			m: TCPFlags(
				SetTCPFlag(TCPFlagSYN),
				UnsetTCPFlag(TCPFlagACK),
			),
			s: `ovs.TCPFlags("+syn", "-ack")`,
		},
		{
			m: TunnelID(0xa),
			s: `ovs.TunnelID(0xa)`,
		},
		{
			m: TunnelIDWithMask(0xa, 0x0f),
			s: `ovs.TunnelIDWithMask(0xa, 0xf)`,
		},
		{
			m: ConjunctionID(123),
			s: `ovs.ConjunctionID(123)`,
		},
		{
			m: ARPOperation(2),
			s: `ovs.ARPOperation(2)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if want, got := tt.s, tt.m.GoString(); want != got {
				t.Fatalf("unexpected Match Go syntax:\n- want: %v\n-  got: %v", want, got)
			}
		})
	}
}

func TestMatchARPOperation(t *testing.T) {
	var tests = []struct {
		desc string
		oper uint16
		out  string
	}{
		{
			desc: "request",
			oper: 1,
			out:  "arp_op=1",
		},
		{
			desc: "response",
			oper: 2,
			out:  "arp_op=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out, err := ARPOperation(tt.oper).MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, string(out); want != got {
				t.Fatalf("unexpected Match output:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

// mustParseMAC is a helper to parse a hardware address from a string using
// net.ParseMAC, that panic on failure.
func mustParseMAC(addr string) net.HardwareAddr {
	mac, err := net.ParseMAC(addr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %q as a hardware address: %v", addr, err))
	}

	return mac
}
