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
	"testing"
)

func Test_UnmarshalText(t *testing.T) {
	testcases := []struct {
		name            string
		output          string
		datapathActions DataPathActions
		flowActions     []string
	}{
		{
			name: "action output port",
			output: `Flow: tcp,in_port=3,vlan_tci=0x0000,dl_src=00:00:00:00:00:00,dl_dst=00:00:00:00:00:00,nw_src=192.0.2.2,nw_dst=0.0.0.0,nw_tos=0,nw_ecn=0,nw_ttl=0,tp_src=0,tp_dst=22,tcp_flags=0

bridge("br0")
-------------
 0. ip,in_port=3,nw_src=192.0.2.0/24, priority 32768
    resubmit(,2)
 2. tcp,tp_dst=22, priority 32768
    output:1

Final flow: unchanged
Megaflow: recirc_id=0,tcp,in_port=3,nw_src=192.0.2.0/24,nw_frag=no,tp_dst=22
Datapath actions: 1`,
			datapathActions: NewDataPathActions("1"),
			flowActions: []string{
				"resubmit(,2)",
				"output:1",
			},
		},
		{
			name: "in_port is LOCAL",
			output: `Flow: tcp,in_port=LOCAL,vlan_tci=0x0000,dl_src=00:00:00:00:00:00,dl_dst=00:00:00:00:00:00,nw_src=192.0.2.2,nw_dst=0.0.0.0,nw_tos=0,nw_ecn=0,nw_ttl=0,tp_src=0,tp_dst=22,tcp_flags=0

bridge("br0")
-------------
 0. ip,in_port=LOCAL,nw_src=192.0.2.0/24, priority 32768
    resubmit(,2)
 2. tcp,tp_dst=22, priority 32768
    output:1

Final flow: unchanged
Megaflow: recirc_id=0,tcp,in_port=LOCAL,nw_src=192.0.2.0/24,nw_frag=no,tp_dst=22
Datapath actions: 1`,
			datapathActions: NewDataPathActions("1"),
			flowActions: []string{
				"resubmit(,2)",
				"output:1",
			},
		},
		{
			name: "popvlan and output port",
			output: `Flow: tcp,in_port=3,vlan_tci=0x0000,dl_src=00:00:00:00:00:00,dl_dst=00:00:00:00:00:00,nw_src=192.0.2.2,nw_dst=0.0.0.0,nw_tos=0,nw_ecn=0,nw_ttl=0,tp_src=0,tp_dst=22,tcp_flags=0

bridge("br0")
-------------
 0. ip,in_port=3,nw_src=192.0.2.0/24, priority 32768
    resubmit(,2)
 2. tcp,tp_dst=22, priority 32768
    output:1

Final flow: unchanged
Megaflow: recirc_id=0,tcp,in_port=3,nw_src=192.0.2.0/24,nw_frag=no,tp_dst=22
Datapath actions: popvlan,1`,
			datapathActions: NewDataPathActions("popvlan,1"),
			flowActions: []string{
				"resubmit(,2)",
				"output:1",
			},
		},
		{
			name: "pushvlan and output port",
			output: `Flow: tcp,in_port=3,vlan_tci=0x0000,dl_src=00:00:00:00:00:00,dl_dst=00:00:00:00:00:00,nw_src=192.0.2.2,nw_dst=0.0.0.0,nw_tos=0,nw_ecn=0,nw_ttl=0,tp_src=0,tp_dst=22,tcp_flags=0

bridge("br0")
-------------
 0. ip,in_port=3,nw_src=192.0.2.0/24, priority 32768
    resubmit(,2)
 2. tcp,tp_dst=22, priority 32768
    output:1

Final flow: unchanged
Megaflow: recirc_id=0,tcp,in_port=3,nw_src=192.0.2.0/24,nw_frag=no,tp_dst=22
Datapath actions: push_vlan(vid=20,pcp=0),4`,
			datapathActions: NewDataPathActions("push_vlan(vid=20,pcp=0),4"),
			flowActions: []string{
				"resubmit(,2)",
				"output:1",
			},
		},
		{
			name: "drop",
			output: `Flow: tcp,in_port=3,vlan_tci=0x0000,dl_src=00:00:00:00:00:00,dl_dst=00:00:00:00:00:00,nw_src=192.0.2.2,nw_dst=0.0.0.0,nw_tos=0,nw_ecn=0,nw_ttl=0,tp_src=0,tp_dst=22,tcp_flags=0

bridge("br0")
-------------
 0. ip,in_port=3,nw_src=192.0.2.0/24, priority 32768
    resubmit(,2)
 2. tcp,tp_dst=22, priority 32768
    output:1

Final flow: unchanged
Megaflow: recirc_id=0,tcp,in_port=3,nw_src=192.0.2.0/24,nw_frag=no,tp_dst=22
Datapath actions: drop`,
			datapathActions: NewDataPathActions("drop"),
			flowActions: []string{
				"resubmit(,2)",
				"output:1",
			},
		},

		{
			name: "connection tracker trace with 3 legs",
			output: ` Flow: icmp,in_port=4,dl_vlan=2,dl_vlan_pcp=0,vlan_tci1=0x0000,dl_src=10:0e:7e:be:fc:40,dl_dst=3c:fd:fe:b6:fb:50,nw_src=10.126.86.66,nw_dst=10.39.144.8,nw_tos=0,nw_ecn=0,nw_ttl=0,icmp_type=8,icmp_code=0

bridge("br0")
-------------
 0. ip,in_port=4,dl_vlan=2,nw_dst=10.39.144.8, priority 900, cookie 0x1dfd9000410000
    resubmit(,25)
25. ip,in_port=4,dl_vlan=2,nw_dst=10.39.144.8, priority 2020, cookie 0x1dfd9000410000
    pop_vlan
    set_field:fe:00:00:00:01:01->eth_src
    set_field:a6:c1:a7:15:a4:3d->eth_dst
    resubmit(,28)
28. priority 100
    resubmit(,35)
35. priority 100
    resubmit(,45)
45. priority 100
    resubmit(,50)
50. ip,dl_dst=a6:c1:a7:15:a4:3d, priority 110, cookie 0x1dfd9000500000
    ct(table=51)
    drop
     -> A clone of the packet is forked to recirculate. The forked pipeline will be resumed at table 51.
     -> Sets the packet to an untracked state, and clears all the conntrack fields.
Final flow: icmp,in_port=4,vlan_tci=0x0000,dl_src=fe:00:00:00:01:01,dl_dst=a6:c1:a7:15:a4:3d,nw_src=10.126.86.66,nw_dst=10.39.144.8,nw_tos=0,nw_ecn=0,nw_ttl=0,icmp_type=8,icmp_code=0
Megaflow: recirc_id=0,eth,ip,tun_id=0,in_port=4,dl_vlan=2,dl_vlan_pcp=0,dl_src=10:0e:7e:be:fc:40,dl_dst=3c:fd:fe:b6:fb:50,nw_src=10.64.0.0/10,nw_dst=10.39.144.8,nw_frag=no
Datapath actions: set(eth(src=fe:00:00:00:01:01,dst=a6:c1:a7:15:a4:3d)),pop_vlan,ct,recirc(0x908)
===============================================================================
recirc(0x908) - resume conntrack with default ct_state=trk|new (use --ct-next to customize)
===============================================================================
Flow: recirc_id=0x908,ct_state=new|trk,eth,icmp,in_port=4,vlan_tci=0x0000,dl_src=fe:00:00:00:01:01,dl_dst=a6:c1:a7:15:a4:3d,nw_src=10.126.86.66,nw_dst=10.39.144.8,nw_tos=0,nw_ecn=0,nw_ttl=0,icmp_type=8,icmp_code=0
bridge("br0")
-------------
    thaw
        Resuming from table 51
51. priority 200
    resubmit(,55)
55. ct_state=+new+trk,icmp,dl_dst=a6:c1:a7:15:a4:3d, priority 1000, cookie 0x1dfd9000500000
    ct(commit,table=60,exec(set_field:0x1dfd90->ct_mark))
    set_field:0x1dfd90->ct_mark
     -> A clone of the packet is forked to recirculate. The forked pipeline will be resumed at table 60.
     -> Sets the packet to an untracked state, and clears all the conntrack fields.
Final flow: recirc_id=0x908,eth,icmp,in_port=4,vlan_tci=0x0000,dl_src=fe:00:00:00:01:01,dl_dst=a6:c1:a7:15:a4:3d,nw_src=10.126.86.66,nw_dst=10.39.144.8,nw_tos=0,nw_ecn=0,nw_ttl=0,icmp_type=8,icmp_code=0
Megaflow: recirc_id=0x908,ct_state=+new-est-rel-rpl+trk,ct_mark=0,eth,icmp,in_port=4,dl_dst=a6:c1:a7:15:a4:3d,nw_frag=no
Datapath actions: ct(commit,mark=0x1dfd90/0xffffffff),recirc(0x909)
===============================================================================
recirc(0x909) - resume conntrack with default ct_state=trk|new (use --ct-next to customize)
===============================================================================
Flow: recirc_id=0x909,ct_state=new|trk,ct_mark=0x1dfd90,eth,icmp,in_port=4,vlan_tci=0x0000,dl_src=fe:00:00:00:01:01,dl_dst=a6:c1:a7:15:a4:3d,nw_src=10.126.86.66,nw_dst=10.39.144.8,nw_tos=0,nw_ecn=0,nw_ttl=0,icmp_type=8,icmp_code=0
bridge("br0")
-------------
    thaw
        Resuming from table 60
60. priority 100
    resubmit(,62)
62. priority 100
    resubmit(,65)
65. ip,vlan_tci=0x0000/0x1fff,dl_dst=a6:c1:a7:15:a4:3d,nw_dst=10.39.144.8, priority 1000, cookie 0x1dfd9000400000
    output:30
Final flow: unchanged
Megaflow: recirc_id=0x909,eth,ip,tun_id=0,in_port=4,vlan_tci=0x0000/0x1fff,dl_dst=a6:c1:a7:15:a4:3d,nw_src=10.64.0.0/10,nw_dst=10.39.144.8,nw_frag=no
Datapath actions: 7`,
			datapathActions: NewDataPathActions("7"),
			flowActions: []string{
				"resubmit(,25)",
				"pop_vlan",
				"set_field:fe:00:00:00:01:01->eth_src",
				"set_field:a6:c1:a7:15:a4:3d->eth_dst",
				"resubmit(,28)",
				"resubmit(,35)",
				"resubmit(,45)",
				"resubmit(,50)",
				"ct(table=51)",
				"drop",
				"recirc",
				"resubmit(,55)",
				"ct(commit,table=60,exec(set_field:0x1dfd90->ct_mark))",
				"set_field:0x1dfd90->ct_mark",
				"recirc",
				"resubmit(,62)",
				"resubmit(,65)",
				"output:30",
			},
		},
		{
			name: "connection tracker trace with 2 legs",
			output: `Flow: ct_mark=0x1e240,ip,in_port=6,vlan_tci=0x0000,dl_src=56:03:b3:97:ac:c8,dl_dst=4a:72:d2:56:78:d1,nw_src=10.36.96.36,nw_dst=10.36.96.37,nw_proto=0,nw_tos=0,nw_ecn=0,nw_ttl=0

bridge("br0")
-------------
 0. ip,in_port=6,dl_src=56:03:b3:97:ac:c8,nw_src=10.36.96.36, priority 2000, cookie 0x1e24001800000
    resubmit(,25)
25. ip,vlan_tci=0x0000/0x1fff,nw_src=10.36.96.36, priority 2000, cookie 0x1e24001800000
    push_vlan:0x8100
    set_field:4118->vlan_vid
    resubmit(,25)
25. ip,dl_vlan=22,nw_dst=10.36.96.37, priority 2020, cookie 0x1d97c01800000
    pop_vlan
    resubmit(,28)
28. ip,in_port=6, priority 110, cookie 0x1e24001900000
    ct(table=30)
    drop
     -> A clone of the packet is forked to recirculate. The forked pipeline will be resumed at table 30.

Final flow: ip,in_port=6,vlan_tci=0x0000,dl_src=56:03:b3:97:ac:c8,dl_dst=4a:72:d2:56:78:d1,nw_src=10.36.96.36,nw_dst=10.36.96.37,nw_proto=0,nw_tos=0,nw_ecn=0,nw_ttl=0
Megaflow: recirc_id=0,eth,ip,in_port=6,vlan_tci=0x0000/0x1fff,dl_src=56:03:b3:97:ac:c8,dl_dst=00:00:00:00:00:00/01:00:00:00:00:00,nw_src=10.36.96.36,nw_dst=10.36.96.37,nw_proto=0,nw_frag=no
Datapath actions: ct,recirc(0x2)

===============================================================================
recirc(0x2) - resume conntrack with ct_state=est|trk
===============================================================================

Flow: recirc_id=0x2,ct_state=est|trk,ct_mark=0x1e240,eth,ip,in_port=6,vlan_tci=0x0000,dl_src=56:03:b3:97:ac:c8,dl_dst=4a:72:d2:56:78:d1,nw_src=10.36.96.36,nw_dst=10.36.96.37,nw_proto=0,nw_tos=0,nw_ecn=0,nw_ttl=0

bridge("br0")
-------------
    thaw
        Resuming from table 30
30. ct_state=+est+trk,ct_mark=0x1e240,in_port=6, priority 220, cookie 0x1e24001900000
    resubmit(,35)
35. priority 100
    resubmit(,45)
45. priority 100
    resubmit(,50)
50. priority 100
    resubmit(,60)
60. ip,in_port=6,dl_src=56:03:b3:97:ac:c8,nw_src=10.36.96.36, priority 1020, cookie 0x1e24001800000
    resubmit(,62)
62. ip,tun_id=0,nw_dst=10.36.96.0/20, priority 1000
    resubmit(,67)
67. conj_id=2, priority 1500, cookie 0x200020000
    set_field:0x2->tun_id
    resubmit(,68)
68. ip,tun_id=0x2,nw_dst=10.36.96.37, priority 1500, cookie 0x200020000
    set_field:10.39.129.11->tun_dst
    output:9
     -> output to native tunnel
     >> native tunnel routing failed

Final flow: recirc_id=0x2,ct_state=est|trk,ct_mark=0x1e240,eth,ip,tun_src=0.0.0.0,tun_dst=10.39.129.11,tun_ipv6_src=::,tun_ipv6_dst=::,tun_gbp_id=0,tun_gbp_flags=0,tun_tos=0,tun_ttl=0,tun_flags=0,in_port=6,vlan_tci=0x0000,dl_src=56:03:b3:97:ac:c8,dl_dst=4a:72:d2:56:78:d1,nw_src=10.36.96.36,nw_dst=10.36.96.37,nw_proto=0,nw_tos=0,nw_ecn=0,nw_ttl=0
Megaflow: recirc_id=0x2,ct_state=+est+trk,ct_mark=0x1e240,eth,ip,tun_id=0,tun_dst=0.0.0.0,in_port=6,dl_src=56:03:b3:97:ac:c8,dl_dst=4a:72:d2:56:78:d1,nw_src=10.36.96.36,nw_dst=10.36.96.37,nw_ecn=0,nw_frag=no
Datapath actions: drop`,
			datapathActions: NewDataPathActions("drop"),
			flowActions: []string{
				"resubmit(,25)",
				"push_vlan:0x8100",
				"set_field:4118->vlan_vid",
				"resubmit(,25)",
				"pop_vlan",
				"resubmit(,28)",
				"ct(table=30)",
				"drop",
				"recirc",
				"resubmit(,35)",
				"resubmit(,45)",
				"resubmit(,50)",
				"resubmit(,60)",
				"resubmit(,62)",
				"resubmit(,67)",
				"set_field:0x2->tun_id",
				"resubmit(,68)",
				"set_field:10.39.129.11->tun_dst",
				"output:9",
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			pt := &ProtoTrace{}
			err := pt.UnmarshalText([]byte(testcase.output))
			if err != nil {
				t.Errorf("error unmarshalling tests: %q", err)
			}

			if !reflect.DeepEqual(testcase.datapathActions, pt.DataPathActions) {
				t.Logf("expected: %v", testcase.datapathActions)
				t.Logf("actual: %v", pt.DataPathActions)
				t.Error("unexpected datapath actions")
			}

			if !reflect.DeepEqual(testcase.flowActions, pt.FlowActions) {
				t.Logf("expected: %v", testcase.flowActions)
				t.Logf("actual: %v", pt.FlowActions)
				t.Error("unexpected trace actions")
			}
		})
	}
}
