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
	"strconv"
	"testing"
)

func TestFlowMarshalText(t *testing.T) {
	var tests = []struct {
		desc string
		f    *Flow
		s    string
		err  error
	}{
		{
			desc: "empty Flow, need Actions to be valid",
			f:    &Flow{},
			err: &FlowError{
				Err: errNoActions,
			},
		},
		{
			desc: "invalid Flow with actions=drop,output:1",
			f: &Flow{
				Actions: []Action{
					Drop(),
					Output(1),
				},
			},
			err: &FlowError{
				Err: errActionsWithDrop,
			},
		},
		{
			desc: "Flow with actions=drop",
			f: &Flow{
				Actions: []Action{Drop()},
			},
			s: "priority=0,table=0,idle_timeout=0,actions=drop",
		},
		{
			desc: "Flow with cookie=10",
			f: &Flow{
				Cookie:  10,
				Actions: []Action{Drop()},
			},
			s: "priority=0,table=0,idle_timeout=0,cookie=0x000000000000000a,actions=drop",
		},
		{
			desc: "Flow with in_port=LOCAL",
			f: &Flow{
				Priority: 2005,
				InPort:   PortLOCAL,
				Actions:  []Action{Resubmit(0, 1)},
			},
			s: "priority=2005,in_port=LOCAL,table=0,idle_timeout=0,actions=resubmit(,1)",
		},
		{
			desc: "ARP Flow",
			f: &Flow{
				Priority: 1005,
				Protocol: ProtocolARP,
				Matches: []Match{
					ARPTargetHardwareAddress(
						net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
					),
					ARPTargetProtocolAddress("169.254.0.0/16"),
				},
				Table:       1,
				IdleTimeout: 0,
				Actions:     []Action{Output(64)},
			},
			s: "priority=1005,arp,arp_tha=aa:bb:cc:dd:ee:ff,arp_tpa=169.254.0.0/16,table=1,idle_timeout=0,actions=output:64",
		},
		{
			desc: "ICMPv4 Flow",
			f: &Flow{
				Priority: 1500,
				Protocol: ProtocolICMPv4,
				Matches: []Match{
					ICMPType(3),
					ICMPCode(1),
					DataLinkSource("00:11:22:33:44:55"),
				},
				Actions: []Action{
					Resubmit(0, 1),
				},
			},
			s: "priority=1500,icmp,icmp_type=3,icmp_code=1,dl_src=00:11:22:33:44:55,table=0,idle_timeout=0,actions=resubmit(,1)",
		},
		{
			desc: "ICMPv6 Flow",
			f: &Flow{
				Priority: 2024,
				Protocol: ProtocolICMPv6,
				InPort:   74,
				Matches: []Match{
					ICMP6Type(135),
					IPv6Source("fe80:aaaa:bbbb:cccc:dddd::1/124"),
					NeighborDiscoverySourceLinkLayer(
						net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
					),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions: []Action{
					ModVLANVID(10),
					Resubmit(0, 1),
				},
			},
			s: "priority=2024,icmp6,in_port=74,icmpv6_type=135,ipv6_src=fe80:aaaa:bbbb:cccc:dddd::1/124,nd_sll=00:11:22:33:44:55,table=0,idle_timeout=0,actions=mod_vlan_vid:10,resubmit(,1)",
		},
		{
			desc: "ICMPv6 Type and Code Flow",
			f: &Flow{
				Priority: 2024,
				Protocol: ProtocolICMPv6,
				InPort:   74,
				Matches: []Match{
					ICMP6Type(1),
					ICMP6Code(3),
					IPv6Source("fe80:aaaa:bbbb:cccc:dddd::1/124"),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions: []Action{
					ModVLANVID(10),
					Resubmit(0, 1),
				},
			},
			s: "priority=2024,icmp6,in_port=74,icmpv6_type=1,icmpv6_code=3,ipv6_src=fe80:aaaa:bbbb:cccc:dddd::1/124,table=0,idle_timeout=0,actions=mod_vlan_vid:10,resubmit(,1)",
		},
		{
			desc: "IPv4 Flow",
			f: &Flow{
				Priority: 2020,
				Protocol: ProtocolIPv4,
				InPort:   31,
				Matches: []Match{
					DataLinkSource("00:11:22:33:44:55"),
					NetworkSource("10.0.0.1"),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions: []Action{
					ModVLANVID(20),
					Resubmit(0, 1),
				},
			},
			s: "priority=2020,ip,in_port=31,dl_src=00:11:22:33:44:55,nw_src=10.0.0.1,table=0,idle_timeout=0,actions=mod_vlan_vid:20,resubmit(,1)",
		},
		{
			desc: "IPv6 Flow",
			f: &Flow{
				Priority: 1020,
				Protocol: ProtocolIPv6,
				Matches: []Match{
					DataLinkDestination("01:02:03:04:05:06"),
					IPv6Destination("fe80::abcd:1"),
				},
				Table:       1,
				IdleTimeout: 0,
				Actions: []Action{
					StripVLAN(),
					Output(69),
				},
			},
			s: "priority=1020,ipv6,dl_dst=01:02:03:04:05:06,ipv6_dst=fe80::abcd:1,table=1,idle_timeout=0,actions=strip_vlan,output:69",
		},
		{
			desc: "TCPv4 Flow",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolTCPv4,
				InPort:   72,
				Matches: []Match{
					TransportDestinationPort(995),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
			s: "priority=3000,tcp,in_port=72,tp_dst=995,table=0,idle_timeout=0,actions=drop",
		},
		{
			desc: "TCPv6 Flow",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolTCPv6,
				InPort:   15,
				Matches: []Match{
					TransportDestinationPort(465),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
			s: "priority=3000,tcp6,in_port=15,tp_dst=465,table=0,idle_timeout=0,actions=drop",
		},
		{
			desc: "UDPv4 Flow",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolUDPv4,
				InPort:   33,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
			s: "priority=3000,udp,in_port=33,tp_dst=80,table=0,idle_timeout=0,actions=drop",
		},
		{
			desc: "UDPv6 Flow",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolUDPv6,
				InPort:   49,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
			s: "priority=3000,udp6,in_port=49,tp_dst=80,table=0,idle_timeout=0,actions=drop",
		},
		{
			desc: "IPv4 SSH conntrack flow",
			f: &Flow{
				Priority: 4010,
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
				Actions: []Action{
					ConnectionTracking("commit,exec(set_field:1->ct_label,set_field:1->ct_mark)"),
					Resubmit(0, 1),
				},
			},
			s: "priority=4010,tcp,ct_state=+trk+new,nw_dst=192.0.2.1,tp_dst=22,table=45,idle_timeout=0,actions=ct(commit,exec(set_field:1->ct_label,set_field:1->ct_mark)),resubmit(,1)",
		},
		{
			desc: "TCP flags flow",
			f: &Flow{
				Priority: 4020,
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
				Actions: []Action{
					Resubmit(0, 1),
				},
			},
			s: "priority=4020,tcp,tcp_flags=+syn+ack,nw_dst=192.0.2.1,tp_dst=22,table=45,idle_timeout=0,actions=resubmit(,1)",
		},
		{
			desc: "Conjunction flow",
			f: &Flow{
				Priority: 400,
				Protocol: ProtocolIPv4,
				Matches: []Match{
					NetworkDestination("192.0.2.1"),
				},
				Table: 45,
				Actions: []Action{
					Conjunction(123, 1, 2),
				},
			},
			s: "priority=400,ip,nw_dst=192.0.2.1,table=45,idle_timeout=0,actions=conjunction(123,1/2)",
		},
		{
			desc: "TP Port Range",
			f: &Flow{
				InPort: 72,
				Matches: []Match{
					TransportSourceMaskedPort(0xea60, 0xffe0),
					TransportDestinationMaskedPort(60000, 0xffe0),
				},
				Table:   55,
				Actions: []Action{Drop()},
			},
			s: "priority=0,in_port=72,tp_src=0xea60/0xffe0,tp_dst=0xea60/0xffe0,table=55,idle_timeout=0,actions=drop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			b, err := tt.f.MarshalText()
			if want, got := tt.err, err; !flowErrorEqual(want, got) {
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

func TestFlowUnmarshalText(t *testing.T) {
	var tests = []struct {
		desc string
		s    string
		f    *Flow
		err  error
	}{
		{
			desc: "empty Flow string, need actions fields",
			err: &FlowError{
				Err: errNoActions,
			},
		},
		{
			desc: "Flow matchers in the actions field",
			s:    "actions=drop,priority=10",
			err: &FlowError{
				Err: errInvalidActions,
			},
		},
		{
			desc: "Flow string with malformed resubmit action, no comma",
			s:    "priority=10,actions=resubmit(",
			err: &FlowError{
				Err: errInvalidActions,
			},
		},
		{
			desc: "Flow string with malformed resubmit action, with comma",
			s:    "priority=10,actions=resubmit(,",
			err: &FlowError{
				Err: errInvalidActions,
			},
		},
		{
			desc: "Flow string with malformed resubmit action, without ending parenthesis",
			s:    "priority=10,actions=resubmit(,1",
			err: &FlowError{
				Err: errInvalidActions,
			},
		},
		{
			desc: "Flow string with actions=drop,output:1",
			s:    "priority=0,table=0,idle_timeout=0,actions=drop,output:1",
			err: &FlowError{
				Err: errActionsWithDrop,
			},
		},
		{
			desc: "Flow string with invalid priority integer",
			s:    "priority=foo,actions=drop",
			err: &FlowError{
				Err: &strconv.NumError{
					Func: "ParseInt",
					Num:  "foo",
					Err:  strconv.ErrSyntax,
				},
			},
		},
		{
			desc: "Flow string with invalid cookie integer",
			s:    "priority=10,table=0,cookie=foo,actions=drop",
			err: &FlowError{
				Err: &strconv.NumError{
					Func: "ParseUint",
					Num:  "foo",
					Err:  strconv.ErrSyntax,
				},
			},
		},
		{
			desc: "Flow string with invalid idle_timeout integer",
			s:    "priority=10,idle_timeout=foo,actions=drop",
			err: &FlowError{
				Err: &strconv.NumError{
					Func: "ParseInt",
					Num:  "foo",
					Err:  strconv.ErrSyntax,
				},
			},
		},
		{
			desc: "Flow string with invalid table integer",
			s:    "priority=10,table=foo,actions=drop",
			err: &FlowError{
				Err: &strconv.NumError{
					Func: "ParseInt",
					Num:  "foo",
					Err:  strconv.ErrSyntax,
				},
			},
		},
		{
			desc: "Flow string with invalid in_port integer",
			s:    "priority=10,in_port=foo,table=0,actions=drop",
			err: &FlowError{
				Err: &strconv.NumError{
					Func: "ParseInt",
					Num:  "foo",
					Err:  strconv.ErrSyntax,
				},
			},
		},
		{
			desc: "Flow string with actions key without =value",
			s:    "priority=10,actions",
			err: &FlowError{
				Err: errNoActions,
			},
		},
		{
			desc: "Flow string with actions key with '=' but empty value",
			s:    "priority=10,actions=",
			err: &FlowError{
				Err: errNoActions,
			},
		},
		{
			desc: "Flow with actions=drop",
			s:    "priority=0,table=0,idle_timeout=0,actions=drop",
			f: &Flow{
				Actions: []Action{Drop()},
			},
		},
		{
			desc: "Flow with cookie=10",
			s:    "priority=0,table=0,idle_timeout=0,cookie=10,actions=drop",
			f: &Flow{
				Cookie:  10,
				Actions: []Action{Drop()},
			},
		},
		{
			desc: "Flow with hex cookie=0xff",
			s:    "priority=0,table=0,idle_timeout=0,cookie=0xff,actions=drop",
			f: &Flow{
				Cookie:  255,
				Actions: []Action{Drop()},
			},
		},
		{
			desc: "Flow with hex cookie padded left",
			s:    "priority=0,table=0,idle_timeout=0,cookie=0x00000000000000ff,actions=drop",
			f: &Flow{
				Cookie:  255,
				Actions: []Action{Drop()},
			},
		},
		{
			desc: "Flow with hex cookie padded right",
			s:    "priority=0,table=0,idle_timeout=0,cookie=0xff00000000000000,actions=drop",
			f: &Flow{
				Cookie:  0xff00000000000000,
				Actions: []Action{Drop()},
			},
		},
		{
			desc: "Flow with in_port=LOCAL",
			s:    "priority=2005,in_port=LOCAL,table=0,idle_timeout=0,actions=resubmit(,1)",
			f: &Flow{
				Priority: 2005,
				InPort:   PortLOCAL,
				Actions:  []Action{Resubmit(0, 1)},
			},
		},
		{
			desc: "ARP Flow",
			s:    "priority=1005,arp,arp_tha=aa:bb:cc:dd:ee:ff,arp_tpa=169.254.0.0/16,table=1,idle_timeout=0,actions=output:64",
			f: &Flow{
				Priority: 1005,
				Protocol: ProtocolARP,
				Matches: []Match{
					ARPTargetHardwareAddress(
						net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
					),
					ARPTargetProtocolAddress("169.254.0.0/16"),
				},
				Table:       1,
				IdleTimeout: 0,
				Actions:     []Action{Output(64)},
			},
		},
		{
			desc: "ICMPv4 Flow",
			s:    "priority=1500,icmp,icmp_type=3,icmp_code=1,dl_src=00:11:22:33:44:55,table=0,idle_timeout=0,actions=resubmit(,1)",
			f: &Flow{
				Priority: 1500,
				Protocol: ProtocolICMPv4,
				Matches: []Match{
					ICMPType(3),
					ICMPCode(1),
					DataLinkSource("00:11:22:33:44:55"),
				},
				Actions: []Action{
					Resubmit(0, 1),
				},
			},
		},
		{
			desc: "ICMPv6 Flow",
			s:    "priority=2024,icmp6,in_port=74,icmpv6_type=135,ipv6_src=fe80:aaaa:bbbb:cccc:dddd::1/124,nd_sll=00:11:22:33:44:55,table=0,idle_timeout=0,actions=mod_vlan_vid:10,resubmit(,1)",
			f: &Flow{
				Priority: 2024,
				Protocol: ProtocolICMPv6,
				InPort:   74,
				Matches: []Match{
					ICMP6Type(135),
					IPv6Source("fe80:aaaa:bbbb:cccc:dddd::1/124"),
					NeighborDiscoverySourceLinkLayer(
						net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
					),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions: []Action{
					ModVLANVID(10),
					Resubmit(0, 1),
				},
			},
		},
		{
			desc: "ICMPv6 Type and Code Flow",
			s:    "priority=2024,icmp6,in_port=74,icmpv6_type=1,icmpv6_code=3,ipv6_src=fe80:aaaa:bbbb:cccc:dddd::1/124,table=0,idle_timeout=0,actions=mod_vlan_vid:10,resubmit(,1)",
			f: &Flow{
				Priority: 2024,
				Protocol: ProtocolICMPv6,
				InPort:   74,
				Matches: []Match{
					ICMP6Type(1),
					ICMP6Code(3),
					IPv6Source("fe80:aaaa:bbbb:cccc:dddd::1/124"),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions: []Action{
					ModVLANVID(10),
					Resubmit(0, 1),
				},
			},
		},
		{
			desc: "IPv4 Flow",
			s:    "priority=2020,ip,in_port=31,dl_src=00:11:22:33:44:55,nw_src=10.0.0.1,table=0,idle_timeout=0,actions=mod_vlan_vid:20,resubmit(,1)",
			f: &Flow{
				Priority: 2020,
				Protocol: ProtocolIPv4,
				InPort:   31,
				Matches: []Match{
					DataLinkSource("00:11:22:33:44:55"),
					NetworkSource("10.0.0.1"),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions: []Action{
					ModVLANVID(20),
					Resubmit(0, 1),
				},
			},
		},
		{
			desc: "IPv6 Flow",
			s:    "priority=1020,ipv6,dl_dst=01:02:03:04:05:06,ipv6_dst=fe80::abcd:1,table=1,idle_timeout=0,actions=strip_vlan,output:69",
			f: &Flow{
				Priority: 1020,
				Protocol: ProtocolIPv6,
				Matches: []Match{
					DataLinkDestination("01:02:03:04:05:06"),
					IPv6Destination("fe80::abcd:1"),
				},
				Table:       1,
				IdleTimeout: 0,
				Actions: []Action{
					StripVLAN(),
					Output(69),
				},
			},
		},
		{
			desc: "TCPv4 Flow",
			s:    "priority=3000,tcp,in_port=72,tp_dst=995,table=0,idle_timeout=0,actions=drop",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolTCPv4,
				InPort:   72,
				Matches: []Match{
					TransportDestinationPort(995),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
		},
		{
			desc: "TCPv6 Flow",
			s:    "priority=3000,tcp6,in_port=15,tp_dst=465,table=0,idle_timeout=0,actions=drop",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolTCPv6,
				InPort:   15,
				Matches: []Match{
					TransportDestinationPort(465),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
		},
		{
			desc: "UDPv4 Flow",
			s:    "priority=3000,udp,in_port=33,tp_dst=80,table=0,idle_timeout=0,actions=drop",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolUDPv4,
				InPort:   33,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
		},
		{
			desc: "UDPv6 Flow",
			s:    "priority=3000,udp6,in_port=49,tp_dst=80,table=0,idle_timeout=0,actions=drop",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolUDPv6,
				InPort:   49,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
		},
		{
			desc: "IPv4 SSH conntrack flow",
			s:    "priority=4010,tcp,ct_state=+trk+new,nw_dst=192.0.2.1,tp_dst=22,table=45,idle_timeout=0,actions=ct(commit,exec(set_field:1->ct_label,set_field:1->ct_mark)),resubmit(,1)",
			f: &Flow{
				Priority: 4010,
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
				Actions: []Action{
					ConnectionTracking("commit,exec(set_field:1->ct_label,set_field:1->ct_mark)"),
					Resubmit(0, 1),
				},
			},
		},
		{
			desc: "Flow generated by ovs-ofctl dump-flows",
			s:    " cookie=0x0, duration=9215.748s, table=0, n_packets=6, n_bytes=480, idle_age=9206, hard_age=65535, priority=820,in_port=LOCAL actions=mod_vlan_vid:10,output:1",
			f: &Flow{
				Priority: 820,
				InPort:   PortLOCAL,
				Matches:  []Match{},
				Table:    0,
				Actions: []Action{
					ModVLANVID(10),
					Output(1),
				},
			},
		},
		{
			desc: "CT classifier flow generated by ovs-ofctl dump-flows",
			s:    " cookie=0x0, duration=1121991.329s, table=50, n_packets=0, n_bytes=0, priority=110,ip,dl_src=f1:f2:f3:f4:f5:f6 actions=ct(table=51)",
			f: &Flow{
				Priority: 110,
				Protocol: ProtocolIPv4,
				Matches: []Match{
					DataLinkSource("f1:f2:f3:f4:f5:f6"),
				},
				Table: 50,
				Actions: []Action{
					ConnectionTracking("table=51"),
				},
			},
		},
		{
			desc: "CT default flow generated by ovs-ofctl dump-flows",
			s:    " cookie=0x0, duration=83229.846s, table=51, n_packets=3, n_bytes=234, priority=101,ct_state=+new+rel+trk,ip actions=ct(commit,table=65)",
			f: &Flow{
				Priority: 101,
				Protocol: ProtocolIPv4,
				Matches: []Match{
					ConnectionTrackingState(
						SetState(CTStateNew),
						SetState(CTStateRelated),
						SetState(CTStateTracked),
					),
				},
				Table: 51,
				Actions: []Action{
					ConnectionTracking("commit,table=65"),
				},
			},
		},
		{
			desc: "CT ingress user flow generated by ovs-ofctl dump-flows",
			s:    " cookie=0x0, duration=920420.008s, table=55, n_packets=0, n_bytes=0, priority=1010,ct_state=+new+trk,tcp,dl_dst=f1:f2:f3:f4:f5:f6,tp_dst=80 actions=ct(commit,table=65,exec(load:0x1fb5fce->NXM_NX_CT_MARK[]))",
			f: &Flow{
				Priority: 1010,
				Protocol: ProtocolTCPv4,
				Matches: []Match{
					ConnectionTrackingState(
						SetState(CTStateNew),
						SetState(CTStateTracked),
					),
					DataLinkDestination("f1:f2:f3:f4:f5:f6"),
					TransportDestinationPort(80),
				},
				Table: 55,
				Actions: []Action{
					ConnectionTracking("commit,table=65,exec(load:0x1fb5fce->NXM_NX_CT_MARK[])"),
				},
			},
		},
		{
			desc: "TCP Flags flow generated by ovs-ofctl dump-flows",
			s:    " cookie=0x0, duration=13.265s, table=12, n_packets=0, n_bytes=0, idle_age=13, priority=1010,tcp,tcp_flags=+syn-psh+ack actions=resubmit(,13)",
			f: &Flow{
				Priority: 1010,
				Protocol: ProtocolTCPv4,
				Matches: []Match{
					TCPFlags(
						SetTCPFlag(TCPFlagSYN),
						UnsetTCPFlag(TCPFlagPSH),
						SetTCPFlag(TCPFlagACK),
					),
				},
				Table: 12,
				Actions: []Action{
					Resubmit(0, 13),
				},
			},
		},
		{
			desc: "External service ingress flow generated by ovs-ofctl dump-flows",
			s:    " cookie=0xa, duration=1381314.983s, table=65, n_packets=0, n_bytes=0, priority=4040,ip,dl_dst=f1:f2:f3:f4:f5:f6,nw_src=169.254.169.254,nw_dst=169.254.0.0/16 actions=output:19",
			f: &Flow{
				Priority: 4040,
				Protocol: ProtocolIPv4,
				Matches: []Match{
					DataLinkDestination("f1:f2:f3:f4:f5:f6"),
					NetworkSource("169.254.169.254"),
					NetworkDestination("169.254.0.0/16"),
				},
				Table:  65,
				Cookie: 10,
				Actions: []Action{
					Output(19),
				},
			},
		},
		{
			desc: "TP Port Range",
			s:    "priority=3000,tcp,in_port=72,tp_src=0xea60/0xffe0,tp_dst=0xea60/0xffe0,table=0,idle_timeout=0,actions=drop",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolTCPv4,
				InPort:   72,
				Matches: []Match{
					TransportSourceMaskedPort(60000, 0xffe0),
					TransportDestinationMaskedPort(0xea60, 0xffe0),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			f := new(Flow)
			err := f.UnmarshalText([]byte(tt.s))

			// Need temporary strings to avoid nil pointer dereference
			// panics when checking Error method.
			if want, got := tt.err, err; !flowErrorEqual(want, got) {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.f, f; !flowsEqual(want, got) {
				t.Fatalf("unexpected Flow:\n- want: %#v\n-  got: %#v",
					want, got)
			}
		})
	}
}

func TestFlowMatchFlow(t *testing.T) {
	var tests = []struct {
		desc string
		f    *Flow
		m    *MatchFlow
		err  error
	}{
		{
			desc: "Flow with actions=drop",
			f: &Flow{
				Actions: []Action{Drop()},
			},
			m: &MatchFlow{},
		},
		{
			desc: "Flow with cookie=10",
			f: &Flow{
				Cookie:  10,
				Actions: []Action{Drop()},
			},
			m: &MatchFlow{
				Cookie: 10,
			},
		},
		{
			desc: "Flow with in_port=LOCAL",
			f: &Flow{
				Priority: 2005,
				InPort:   PortLOCAL,
				Actions:  []Action{Resubmit(0, 1)},
			},
			m: &MatchFlow{
				InPort: PortLOCAL,
			},
		},
		{
			desc: "ARP Flow",
			f: &Flow{
				Priority: 1005,
				Protocol: ProtocolARP,
				Matches: []Match{
					ARPTargetHardwareAddress(
						net.HardwareAddr{0x04, 0x01, 0x41, 0xa6, 0xb8, 0x01},
					),
					ARPTargetProtocolAddress("169.254.0.0/16"),
				},
				Table:       1,
				IdleTimeout: 0,
				Actions:     []Action{Output(64)},
			},
			m: &MatchFlow{
				Protocol: ProtocolARP,
				Matches: []Match{
					ARPTargetHardwareAddress(
						net.HardwareAddr{0x04, 0x01, 0x41, 0xa6, 0xb8, 0x01},
					),
					ARPTargetProtocolAddress("169.254.0.0/16"),
				},
				Table: 1,
			},
		},
		{
			desc: "ICMPv4 Flow",
			f: &Flow{
				Priority: 1500,
				Protocol: ProtocolICMPv4,
				Matches: []Match{
					DataLinkSource("00:11:22:33:44:55"),
				},
				Actions: []Action{
					Resubmit(0, 1),
				},
			},
			m: &MatchFlow{
				Protocol: ProtocolICMPv4,
				Matches: []Match{
					DataLinkSource("00:11:22:33:44:55"),
				},
			},
		},
		{
			desc: "ICMPv6 Flow",
			f: &Flow{
				Priority: 2024,
				Protocol: ProtocolICMPv6,
				InPort:   74,
				Matches: []Match{
					ICMPType(135),
					IPv6Source("fe80:aaaa:bbbb:cccc:dddd::1/124"),
					NeighborDiscoverySourceLinkLayer(
						net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
					),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions: []Action{
					ModVLANVID(10),
					Resubmit(0, 1),
				},
			},
			m: &MatchFlow{
				Protocol: ProtocolICMPv6,
				InPort:   74,
				Matches: []Match{
					ICMPType(135),
					IPv6Source("fe80:aaaa:bbbb:cccc:dddd::1/124"),
					NeighborDiscoverySourceLinkLayer(
						net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
					),
				},
				Table: 0,
			},
		},
		{
			desc: "IPv4 Flow",
			f: &Flow{
				Priority: 2020,
				Protocol: ProtocolIPv4,
				InPort:   31,
				Matches: []Match{
					DataLinkSource("00:11:22:33:44:55"),
					NetworkSource("10.0.0.1"),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions: []Action{
					ModVLANVID(20),
					Resubmit(0, 1),
				},
			},
			m: &MatchFlow{
				Protocol: ProtocolIPv4,
				InPort:   31,
				Matches: []Match{
					DataLinkSource("00:11:22:33:44:55"),
					NetworkSource("10.0.0.1"),
				},
				Table: 0,
			},
		},
		{
			desc: "IPv6 Flow",
			f: &Flow{
				Priority: 1020,
				Protocol: ProtocolIPv6,
				Matches: []Match{
					DataLinkDestination("01:02:03:04:05:06"),
					IPv6Destination("fe80::abcd:1"),
				},
				Table:       1,
				IdleTimeout: 0,
				Actions: []Action{
					StripVLAN(),
					Output(69),
				},
			},
			m: &MatchFlow{
				Protocol: ProtocolIPv6,
				Matches: []Match{
					DataLinkDestination("01:02:03:04:05:06"),
					IPv6Destination("fe80::abcd:1"),
				},
				Table: 1,
			},
		},
		{
			desc: "TCPv4 Flow",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolTCPv4,
				InPort:   72,
				Matches: []Match{
					TransportDestinationPort(995),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
			m: &MatchFlow{
				Protocol: ProtocolTCPv4,
				InPort:   72,
				Matches: []Match{
					TransportDestinationPort(995),
				},
				Table: 0,
			},
		},
		{
			desc: "TCPv6 Flow",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolTCPv6,
				InPort:   15,
				Matches: []Match{
					TransportDestinationPort(465),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
			m: &MatchFlow{
				Protocol: ProtocolTCPv6,
				InPort:   15,
				Matches: []Match{
					TransportDestinationPort(465),
				},
				Table: 0,
			},
		},
		{
			desc: "UDPv4 Flow",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolUDPv4,
				InPort:   33,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
			m: &MatchFlow{
				Protocol: ProtocolUDPv4,
				InPort:   33,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table: 0,
			},
		},
		{
			desc: "UDPv6 Flow",
			f: &Flow{
				Priority: 3000,
				Protocol: ProtocolUDPv6,
				InPort:   49,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table:       0,
				IdleTimeout: 0,
				Actions:     []Action{Drop()},
			},
			m: &MatchFlow{
				Protocol: ProtocolUDPv6,
				InPort:   49,
				Matches: []Match{
					TransportDestinationPort(80),
				},
				Table: 0,
			},
		},
		{
			desc: "IPv4 SSH conntrack flow",
			f: &Flow{
				Priority: 4010,
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
				Actions: []Action{
					ConnectionTracking("commit,exec(set_field:1->ct_label,set_field:1->ct_mark)"),
					Resubmit(0, 1),
				},
			},
			m: &MatchFlow{
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
		},
		{
			desc: "TCP Flags Flow",
			f: &Flow{
				Priority: 4010,
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
				Actions: []Action{
					Resubmit(0, 1),
				},
			},
			m: &MatchFlow{
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			want, got := tt.m, tt.f.MatchFlow()
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected MatchFlow:\n- want: %#v\n-  got: %#v",
					want, got)
			}
		})
	}
}

// flowsEqual determines if two possible Flows are equal.
func flowsEqual(a *Flow, b *Flow) bool {
	// Special case: both nil is OK
	if a == nil && b == nil {
		return true
	}

	am, err := a.marshalMatches()
	if err != nil {
		panic(fmt.Sprintf("unexpected error parsing matches: %v", err))
	}
	bm, err := b.marshalMatches()
	if err != nil {
		panic(fmt.Sprintf("unexpected error parsing matches: %v", err))
	}

	if !reflect.DeepEqual(am, bm) {
		return false
	}

	aa, err := a.marshalActions()
	if err != nil {
		panic(fmt.Sprintf("unexpected error parsing actions: %v", err))
	}
	ba, err := b.marshalActions()
	if err != nil {
		panic(fmt.Sprintf("unexpected error parsing actions: %v", err))
	}

	if !reflect.DeepEqual(aa, ba) {
		return false
	}

	// Since functions cannot be compared, nil them for final check
	a.Matches = nil
	b.Matches = nil
	a.Actions = nil
	b.Actions = nil

	return reflect.DeepEqual(a, b)
}

// flowErrorEqual determines if two possible FlowErrors are equal.
func flowErrorEqual(a error, b error) bool {
	// Special case: both nil is OK
	if a == nil && b == nil {
		return true
	}

	fa, ok := a.(*FlowError)
	if !ok {
		return false
	}

	fb, ok := b.(*FlowError)
	if !ok {
		return false
	}

	// Zero out Str field for comparison
	fa.Str = ""
	fb.Str = ""

	return reflect.DeepEqual(fa, fb)
}
