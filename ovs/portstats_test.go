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
	"strconv"
	"testing"
)

func TestPortStatsUnmarshalText(t *testing.T) {
	var tests = []struct {
		desc string
		s    string
		p    *PortStats
		err  error
	}{
		{
			desc: "empty string",
			err:  ErrInvalidPortStats,
		},
		{
			desc: "incorrect 'port' location",
			s:    "a port c d e f g h i j k l m n o p",
			err:  ErrInvalidPortStats,
		},
		{
			desc: "incorrect 'rx' location",
			s:    "port 1: c rx e f g h i j k l m n o p",
			err:  ErrInvalidPortStats,
		},
		{
			desc: "incorrect 'tx' location",
			s:    "port 1: rx d e f g h i tx k l m n o p",
			err:  ErrInvalidPortStats,
		},
		{
			desc: "broken key=value pair",
			s: `
				  port  1: rx pkts=0, bytes=0, drop=0, errs=0, frame=0, over=0, crc=0
				             tx pkts=0, bytes=0, drop=0, errs=0, collfoo
				`,
			err: ErrInvalidPortStats,
		},
		{
			desc: "invalid integer port",
			s: `
				  port  foo: rx pkts=0, bytes=0, drop=0, errs=0, frame=0, over=0, crc=0
				             tx pkts=0, bytes=0, drop=0, errs=0, coll=0
				`,
			err: &strconv.NumError{
				Func: "ParseInt",
				Num:  "foo",
				Err:  strconv.ErrSyntax,
			},
		},
		{
			desc: "invalid integer value",
			s: `
				  port  1: rx pkts=0, bytes=0, drop=0, errs=0, frame=0, over=0, crc=0
				             tx pkts=0, bytes=0, drop=0, errs=0, coll=foo
				`,
			err: &strconv.NumError{
				Func: "ParseUint",
				Num:  "foo",
				Err:  strconv.ErrSyntax,
			},
		},
		{
			desc: "OK",
			s: `
				  port  LOCAL: rx pkts=159998521, bytes=3839413852, drop=15891659, errs=10, frame=20, over=30, crc=40
				             tx pkts=7315577, bytes=3699296923, drop=50, errs=60, coll=70
				`,
			p: &PortStats{
				PortID: PortLOCAL,
				Received: PortStatsReceive{
					Packets: 159998521,
					Bytes:   3839413852,
					Dropped: 15891659,
					Errors:  10,
					Frame:   20,
					Over:    30,
					CRC:     40,
				},
				Transmitted: PortStatsTransmit{
					Packets:    7315577,
					Bytes:      3699296923,
					Dropped:    50,
					Errors:     60,
					Collisions: 70,
				},
			},
		},
		{
			desc: "OK tun0",
			s: `
				port  8: rx pkts=10, bytes=20, drop=?, errs=?, frame=?, over=?, crc=?
				         tx pkts=10, bytes=20, drop=?, errs=?, coll=?
				`,
			p: &PortStats{
				PortID: 8,
				Received: PortStatsReceive{
					Packets: 10,
					Bytes:   20,
				},
				Transmitted: PortStatsTransmit{
					Packets: 10,
					Bytes:   20,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			p := new(PortStats)
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
