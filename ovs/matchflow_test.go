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
	"testing"
)

func TestMatchFlowMarshalText(t *testing.T) {
	var tests = []struct {
		desc string
		f    *MatchFlow
		s    string
		err  error
	}{
		{
			desc: "empty",
			f:    &MatchFlow{Table: AnyTable},
			err: &MatchFlowError{
				Err: errEmptyMatchFlow,
			},
		},
		{
			desc: "Flow with cookie=10/-1, in any table",
			f: &MatchFlow{
				Cookie: 10,
				Table:  AnyTable,
			},
			s: "cookie=0x000000000000000a/-1",
		},
		{
			desc: "Flow with cookie=0x1/0xf, in any table",
			f: &MatchFlow{
				Cookie:     0x1,
				CookieMask: 0xf,
				Table:      AnyTable,
			},
			s: "cookie=0x0000000000000001/0x000000000000000f",
		},
		{
			desc: "Flow with in_port=LOCAL",
			f: &MatchFlow{
				InPort: PortLOCAL,
			},
			s: "in_port=LOCAL,table=0",
		},
		{
			desc: "ARP Flow",
			f: &MatchFlow{
				Protocol: ProtocolARP,
				Matches: []Match{
					ARPTargetHardwareAddress(
						net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
					),
					ARPTargetProtocolAddress("169.254.0.0/16"),
				},
				Table: 1,
			},
			s: "arp,arp_tha=aa:bb:cc:dd:ee:ff,arp_tpa=169.254.0.0/16,table=1",
		},
		{
			desc: "ICMPv4 Flow",
			f: &MatchFlow{
				Protocol: ProtocolICMPv4,
				Matches: []Match{
					ICMPType(3),
					ICMPCode(1),
					DataLinkSource("00:11:22:33:44:55"),
				},
			},
			s: "icmp,icmp_type=3,icmp_code=1,dl_src=00:11:22:33:44:55,table=0",
		},
		{
			desc: "ICMPv6 Flow",
			f: &MatchFlow{
				Protocol: ProtocolICMPv6,
				InPort:   74,
				Matches: []Match{
					ICMP6Type(135),
					IPv6Source("fe80:aaaa:bbbb:cccc:dddd::1/124"),
					NeighborDiscoverySourceLinkLayer(
						net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
					),
				},
				Table: 0,
			},
			s: "icmp6,in_port=74,icmpv6_type=135,ipv6_src=fe80:aaaa:bbbb:cccc:dddd::1/124,nd_sll=00:11:22:33:44:55,table=0",
		},
		{
			desc: "ICMPv6 Type and Code Flow",
			f: &MatchFlow{
				Protocol: ProtocolICMPv6,
				InPort:   74,
				Matches: []Match{
					ICMP6Type(1),
					ICMP6Code(3),
					IPv6Source("fe80:aaaa:bbbb:cccc:dddd::1/124"),
				},
				Table: 0,
			},
			s: "icmp6,in_port=74,icmpv6_type=1,icmpv6_code=3,ipv6_src=fe80:aaaa:bbbb:cccc:dddd::1/124,table=0",
		},
		{
			desc: "IPv4 Flow",
			f: &MatchFlow{
				Protocol: ProtocolIPv4,
				InPort:   31,
				Matches: []Match{
					DataLinkSource("00:11:22:33:44:55"),
					NetworkSource("10.0.0.1"),
				},
				Table: 0,
			},
			s: "ip,in_port=31,dl_src=00:11:22:33:44:55,nw_src=10.0.0.1,table=0",
		},
		{
			desc: "IPv6 Flow",
			f: &MatchFlow{
				Protocol: ProtocolIPv6,
				Matches: []Match{
					DataLinkDestination("01:02:03:04:05:06"),
					IPv6Destination("fe80::abcd:1"),
				},
				Table: 1,
			},
			s: "ipv6,dl_dst=01:02:03:04:05:06,ipv6_dst=fe80::abcd:1,table=1",
		},
		{
			desc: "TCPv4 Flow",
			f: &MatchFlow{
				Protocol: ProtocolTCPv4,
				InPort:   72,
				Matches: []Match{
					TransportDestinationPort(995),
				},
				Table: 0,
			},
			s: "tcp,in_port=72,tp_dst=995,table=0",
		},
		{
			desc: "TCPv6 Flow",
			f: &MatchFlow{
				Protocol: ProtocolTCPv6,
				InPort:   15,
				Matches: []Match{
					TransportDestinationPort(465),
				},
				Table: 0,
			},
			s: "tcp6,in_port=15,tp_dst=465,table=0",
		},
		{
			desc: "UDPv4 Flow",
			f: &MatchFlow{
				Protocol: ProtocolUDPv4,
				InPort:   33,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table: 0,
			},
			s: "udp,in_port=33,tp_dst=80,table=0",
		},
		{
			desc: "UDPv6 Flow",
			f: &MatchFlow{
				Protocol: ProtocolUDPv6,
				InPort:   49,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table: 0,
			},
			s: "udp6,in_port=49,tp_dst=80,table=0",
		},
		{
			desc: "IPv4 SSH conntrack flow",
			f: &MatchFlow{
				Protocol: ProtocolTCPv4,
				Matches: []Match{
					ConnectionTrackingState(
						SetState(CTStateTracked),
						SetState(CTStateNew),
					),
					NetworkDestination("192.0.2.1"),
					TransportDestinationPort(22),
				},
				Table: 45,
			},
			s: "tcp,ct_state=+trk+new,nw_dst=192.0.2.1,tp_dst=22,table=45",
		},
		{
			desc: "TCP Flag Flow",
			f: &MatchFlow{
				Protocol: ProtocolTCPv4,
				Matches: []Match{
					TCPFlags(
						SetTCPFlag(TCPFlagSYN),
						SetTCPFlag(TCPFlagACK),
					),
					NetworkDestination("192.0.2.1"),
					TransportDestinationPort(22),
				},
				Table: 45,
			},
			s: "tcp,tcp_flags=+syn+ack,nw_dst=192.0.2.1,tp_dst=22,table=45",
		},
		{
			desc: "TP port range flow",
			f: &MatchFlow{
				Protocol: ProtocolUDPv4,
				InPort:   33,
				Matches: []Match{
					NetworkDestination("192.0.2.1"),
					TransportDestinationMaskedPort(0xea60, 0xffe0),
				},
				Table: 55,
			},
			s: "udp,in_port=33,nw_dst=192.0.2.1,tp_dst=0xea60/0xffe0,table=55",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			b, err := tt.f.MarshalText()
			if want, got := tt.err, err; !matchFlowErrorEqual(want, got) {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}

			if want, got := tt.s, string(b); want != got {
				t.Fatalf("unexpected Flow text:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

// matchFlowErrorEqual determines if two possible MatchFlowErrors are equal.
func matchFlowErrorEqual(a error, b error) bool {
	// Special case: both nil is OK
	if a == nil && b == nil {
		return true
	}

	fa, ok := a.(*MatchFlowError)
	if !ok {
		return false
	}

	fb, ok := b.(*MatchFlowError)
	if !ok {
		return false
	}

	// Zero out Str field for comparison
	fa.Str = ""
	fb.Str = ""

	return reflect.DeepEqual(fa, fb)
}
