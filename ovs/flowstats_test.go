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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFlowStatsUnmarshalText(t *testing.T) {
	var tests = []struct {
		desc  string
		s     string
		stats *FlowStats
		ok    bool
	}{
		{
			desc: "empty string",
		},
		{
			desc: "too few fields",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=141379644",
		},
		{
			desc: "too many fields",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=141379644 flow_count=2, flow_count=3",
		},
		{
			desc: "packet_count missing",
			s:    "NXST_AGGREGATE reply (xid=0x4): frame_count=642800 byte_count=141379644 flow_count=2",
		},
		{
			desc: "byte_count missing",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 bits*8_count=141379644 flow_count=2",
		},
		{
			desc: "bad key=value",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=1=foo byte_count=141379644 flow_count=2",
		},
		{
			desc: "bad packet count",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=toosmall byte_count=141379644 flow_count=2",
		},
		{
			desc: "bad byte count",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=toolarge flow_count=2",
		},
		{
			desc: "bad flow count",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=1 FLOW_count=2",
		},
		{
			desc: "OK",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=141379644 flow_count=2",
			stats: &FlowStats{
				PacketCount: 642800,
				ByteCount:   141379644,
			},
			ok: true,
		},
		{
			desc: "OK, OpenFlow 1.4",
			s:    "OFPST_AGGREGATE reply (OF1.4) (xid=0x2): packet_count=1207 byte_count=101673 flow_count=1",
			stats: &FlowStats{
				PacketCount: 1207,
				ByteCount:   101673,
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stats := new(FlowStats)
			err := stats.UnmarshalText([]byte(tt.s))

			if err != nil && tt.ok {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && !tt.ok {
				t.Fatal("expected an error, but none occurred")
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tt.stats, stats); diff != "" {
				t.Fatalf("unexpected FlowStats (-want +got):\n%s", diff)
			}
		})
	}
}
