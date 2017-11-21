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

func TestFlowStatsUnmarshalText(t *testing.T) {
	var tests = []struct {
		desc string
		s    string
		p    *FlowStats
		err  error
	}{
		{
			desc: "empty string",
			err:  ErrInvalidFlowStats,
		},
		{
			desc: "incorrect number of fields",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=141379644 flow_count=2, flow_count=3",
			err:  ErrInvalidFlowStats,
		},
		{
			desc: "first field is not NXST_AGGREGATE",
			s:    "NXST_REPLY reply (xid=0x4): packet_count=642800 byte_count=141379644 flow_count=2",
			err:  ErrInvalidFlowStats,
		},
		{
			desc: "packet_count string is missing",
			s:    "NXST_AGGREGATE reply (xid=0x4): frame_count=642800 byte_count=141379644 flow_count=2",
			err:  ErrInvalidFlowStats,
		},
		{
			desc: "byte_count string is missing",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 bits*8_count=141379644 flow_count=2",
			err:  ErrInvalidFlowStats,
		},
		{
			desc: "broken packet_count=value pair",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=toosmall byte_count=141379644 flow_count=2",
			err:  ErrInvalidFlowStats,
		},
		{
			desc: "broken byte_count=value pair",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=toolarge flow_count=2",
			err:  ErrInvalidFlowStats,
		},
		{
			desc: "OK",
			s:    "NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=141379644 flow_count=2",
			p: &FlowStats{
				PacketCount: 642800,
				ByteCount:   141379644,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			p := new(FlowStats)
			err := p.UnmarshalText([]byte(tt.s))

			if want, got := errStr(tt.err), errStr(err); want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.p, p; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected PortStats:\n- want: %#v\n-  got: %#v",
					want, got)
			}
		})
	}
}
