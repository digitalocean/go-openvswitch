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
	"bufio"
	"bytes"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestClientOpenFlowAddFlowInvalidFlow(t *testing.T) {
	flow := &Flow{
		// Actions must not be empty for valid flow
		Actions: []Action{},
	}

	c := testClient(nil, nil)

	want := &FlowError{
		Err: errNoActions,
	}

	if got := c.OpenFlow.AddFlow("foo", flow); !flowErrorEqual(want, got) {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientOpenFlowAddFlowNoArgs(t *testing.T) {
	bridge := "br0"
	flow := &Flow{
		Priority: 10,
		Protocol: ProtocolIPv4,
		Actions:  []Action{Drop()},
	}

	c := testClient(nil, func(cmd string, args ...string) ([]byte, error) {
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"add-flow",
			string(bridge),
			"priority=10,ip,table=0,idle_timeout=0,actions=drop",
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.OpenFlow.AddFlow(bridge, flow); err != nil {
		t.Fatalf("unexpected error for Client.OpenFlow.AddFlow: %v", err)
	}
}

func TestClientOpenFlowAddFlowOK(t *testing.T) {
	bridge := "br0"
	flow := &Flow{
		Priority: 10,
		Protocol: ProtocolIPv4,
		Actions:  []Action{Drop()},
	}

	options := []OptionFunc{
		Timeout(1),
		FlowFormat(FlowFormatNXMTableID),
	}

	c := testClient(options, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"add-flow",
			"--flow-format=NXM+table_id",
			string(bridge),
			"priority=10,ip,table=0,idle_timeout=0,actions=drop",
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.OpenFlow.AddFlow(bridge, flow); err != nil {
		t.Fatalf("unexpected error for Client.OpenFlow.AddFlow: %v", err)
	}
}

func TestClientOpenFlowAddFlowBundleOK(t *testing.T) {
	bridge := "br0"

	// Flows for addition
	flows := []*Flow{
		{
			Priority: 10,
			Protocol: ProtocolIPv4,
			Actions:  []Action{Drop()},
		},
		{
			Priority: 20,
			Protocol: ProtocolIPv6,
			Actions:  []Action{Drop()},
		},
		{
			Priority: 30,
			Protocol: ProtocolICMPv4,
			Actions:  []Action{Drop()},
		},
		{
			Priority: 40,
			Protocol: ProtocolICMPv6,
			Actions:  []Action{Drop()},
		},
	}

	// Flows for deletion
	matchFlows := []*MatchFlow{{
		Cookie: 0xdeadbeef,
	}}

	pipe := Pipe(func(stdin io.Reader, cmd string, args ...string) ([]byte, error) {
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"--bundle",
			"add-flow",
			"--flow-format=NXM+table_id",
			string(bridge),
			// Read from stdin.
			"-",
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		mustVerifyFlowBundle(t, stdin, flows, matchFlows)
		return nil, nil
	})

	options := []OptionFunc{
		Timeout(1),
		FlowFormat(FlowFormatNXMTableID),
		pipe,
	}

	c := testClient(options, nil)

	err := c.OpenFlow.AddFlowBundle(bridge, func(tx *FlowTransaction) error {
		for _, f := range flows {
			tx.Add(f)
		}
		for _, f := range matchFlows {
			tx.Delete(f)
		}

		return tx.Commit()
	})
	if err != nil {
		t.Fatalf("unexpected error for Client.OpenFlow.AddFlowBundle: %v", err)
	}
}

func TestClientOpenFlowAddFlowBundleNotCommitted(t *testing.T) {
	bridge := "br0"

	f := &Flow{
		Priority: 10,
		Protocol: ProtocolIPv4,
		Actions:  []Action{Drop()},
	}

	c := testClient(nil, nil)
	err := c.OpenFlow.AddFlowBundle(bridge, func(tx *FlowTransaction) error {
		tx.Add(f)
		// Did not call tx.Commit.
		return nil
	})
	if want, got := errNotCommitted, err; want != got {
		t.Fatalf("unexpected error for Client.OpenFlow.AddFlowBundle:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientOpenFlowAddFlowBundleCommitError(t *testing.T) {
	bridge := "br0"

	flows := []*Flow{
		// No actions, malformed flow.
		{
			Priority: 10,
			Protocol: ProtocolIPv4,
		},
		// Other flows will be ignored.
		{
			Priority: 20,
			Protocol: ProtocolIPv6,
			Actions:  []Action{Drop()},
		},
	}

	c := testClient(nil, nil)
	err := c.OpenFlow.AddFlowBundle(bridge, func(tx *FlowTransaction) error {
		// Explicitly adding one at a time, to trigger early return case
		// after first flow causes error.
		for _, f := range flows {
			tx.Add(f)
		}

		return tx.Commit()
	})

	want := &FlowError{
		Err: errNoActions,
	}
	if got := err; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected error for Client.OpenFlow.AddFlowBundle:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestClientOpenFlowAddFlowBundleDiscard(t *testing.T) {
	bridge := "br0"

	flow := &Flow{
		Priority: 10,
		Protocol: ProtocolIPv4,
		Actions:  []Action{Drop()},
	}

	errFoo := errors.New("some error which caused transaction discard")

	c := testClient(nil, nil)
	err := c.OpenFlow.AddFlowBundle(bridge, func(tx *FlowTransaction) error {
		tx.Add(flow)

		// An error occurred in the middle of the flow bundle. Discard.
		return tx.Discard(errFoo)
	})

	if s := err.Error(); !strings.Contains(s, errFoo.Error()) {
		t.Fatalf("error when discarding was not surfaced: %q", s)
	}
}

func TestClientOpenFlowDelFlowsOK(t *testing.T) {
	bridge := "br0"
	flow := &MatchFlow{
		Protocol: ProtocolIPv4,
		Table:    AnyTable,
	}

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"del-flows",
			string(bridge),
			"ip",
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.OpenFlow.DelFlows(bridge, flow); err != nil {
		t.Fatalf("unexpected error for Client.OpenFlow.DelFlows: %v", err)
	}
}

func TestClientOpenFlowDelFlowsFlushOK(t *testing.T) {
	bridge := "br0"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"del-flows",
			string(bridge),
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.OpenFlow.DelFlows(bridge, nil); err != nil {
		t.Fatalf("unexpected error for Client.OpenFlow.DelFlows: %v", err)
	}
}

func TestClientOpenSudoOK(t *testing.T) {
	const bridge = "br0"

	c := testClient([]OptionFunc{Sudo()}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "sudo", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"ovs-ofctl",
			"del-flows",
			string(bridge),
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.OpenFlow.DelFlows(bridge, nil); err != nil {
		t.Fatalf("unexpected error for Client.OpenFlow.DelFlows: %v", err)
	}
}

func TestClientOpenFlowModPort(t *testing.T) {
	tests := []struct {
		name   string
		bridge string
		port   string
		action PortAction
		err    error
	}{
		{
			name:   "test up action",
			bridge: "br0",
			port:   "bond0",
			action: PortAction("up"),
		},
		{
			name:   "test down action",
			bridge: "br0",
			port:   "bond0",
			action: PortAction("down"),
		},
		{
			name:   "test stp action",
			bridge: "br0",
			port:   "bond0",
			action: PortAction("stp"),
		},
		{
			name:   "test no-stp action",
			bridge: "br0",
			port:   "bond0",
			action: PortAction("no-stp"),
		},
		{
			name:   "test receive action",
			bridge: "br0",
			port:   "bond0",
			action: PortAction("receive"),
		},
		{
			name:   "test no-receive action",
			bridge: "br0",
			port:   "bond0",
			action: PortAction("no-receive"),
		},
		{
			name:   "test receive-stp action",
			bridge: "br0",
			port:   "bond0",
			action: PortAction("receive-stp"),
		},
		{
			name:   "test no-receive-stp action",
			bridge: "br0",
			port:   "bond0",
			action: PortAction("no-receive-stp"),
		},
		{
			name:   "test forward action",
			bridge: "br0",
			port:   "port0",
			action: PortAction("forward"),
		},
		{
			name:   "test no-forward action",
			bridge: "br0",
			port:   "port0",
			action: PortAction("no-forward"),
		},
		{
			name:   "test flood action",
			bridge: "br0",
			port:   "port0",
			action: PortAction("flood"),
		},
		{
			name:   "test no-flood action",
			bridge: "br0",
			port:   "port0",
			action: PortAction("no-flood"),
		},
		{
			name:   "test packet-in action",
			bridge: "br0",
			port:   "port0",
			action: PortAction("packet-in"),
		},
		{
			name:   "test no-packet-in action",
			bridge: "br0",
			port:   "port0",
			action: PortAction("no-packet-in"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
				if want, got := "ovs-ofctl", cmd; want != got {
					t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
						want, got)
				}
				wantArgs := []string{
					"--timeout=1",
					"mod-port",
					string(tt.bridge),
					string(tt.port),
					string(tt.action),
				}
				if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
					t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
						want, got)
				}
				return nil, tt.err
			}).OpenFlow.ModPort(tt.bridge, tt.port, tt.action)
		})
	}
}

func TestClientOpenFlowDumpPortMultipleValues(t *testing.T) {
	c := testClient(nil, func(cmd string, args ...string) ([]byte, error) {
		return []byte(`
		OFPST_PORT reply (xid=0x1): 2 ports
		port  1: rx pkts=1, bytes=1, drop=1, errs=1, frame=1, over=1, crc=1
		         tx pkts=1, bytes=1, drop=1, errs=1, coll=1
		port  2: rx pkts=2, bytes=2, drop=2, errs=2, frame=2, over=2, crc=2
		         tx pkts=2, bytes=2, drop=2, errs=2, coll=2
		`), nil
	})

	want := errMultipleValues
	if _, got := c.OpenFlow.DumpPort("foo", "1"); want != got {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientOpenFlowDumpPortOK(t *testing.T) {
	want := &PortStats{
		PortID: 1,
		Received: PortStatsReceive{
			Packets: 1,
			Bytes:   1,
			Dropped: 1,
			Errors:  1,
			Frame:   1,
			Over:    1,
			CRC:     1,
		},
		Transmitted: PortStatsTransmit{
			Packets:    1,
			Bytes:      1,
			Dropped:    1,
			Errors:     1,
			Collisions: 1,
		},
	}

	bridge := "foo"
	port := 1

	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "dump-ports", bridge, strconv.Itoa(port)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return []byte(`
		OFPST_PORT reply (xid=0x1): 1 port
		port  1: rx pkts=1, bytes=1, drop=1, errs=1, frame=1, over=1, crc=1
		         tx pkts=1, bytes=1, drop=1, errs=1, coll=1
		`), nil
	})

	got, err := c.OpenFlow.DumpPort("foo", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected stats:\n- want: %+v\n-  got: %+v",
			want, got)
	}
}

func TestClientOpenFlowDumpPortsInvalidPortStats(t *testing.T) {
	c := testClient(nil, func(cmd string, args ...string) ([]byte, error) {
		return []byte("OFPST_PORT reply\nport LOCAL: rx pkts=0\ntx"), nil
	})

	want := ErrInvalidPortStats
	if _, got := c.OpenFlow.DumpPorts("foo"); !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientOpenFlowDumpPortsOK(t *testing.T) {
	want := []*PortStats{
		{
			PortID: 1,
			Received: PortStatsReceive{
				Packets: 1,
				Bytes:   1,
				Dropped: 1,
				Errors:  1,
				Frame:   1,
				Over:    1,
				CRC:     1,
			},
			Transmitted: PortStatsTransmit{
				Packets:    1,
				Bytes:      1,
				Dropped:    1,
				Errors:     1,
				Collisions: 1,
			},
		},
		{
			PortID: 2,
			Received: PortStatsReceive{
				Packets: 2,
				Bytes:   2,
				Dropped: 2,
				Errors:  2,
				Frame:   2,
				Over:    2,
				CRC:     2,
			},
			Transmitted: PortStatsTransmit{
				Packets:    2,
				Bytes:      2,
				Dropped:    2,
				Errors:     2,
				Collisions: 2,
			},
		},
	}

	bridge := "br0"

	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "dump-ports", string(bridge)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return []byte(`
		OFPST_PORT reply (xid=0x1): 2 ports
		port  1: rx pkts=1, bytes=1, drop=1, errs=1, frame=1, over=1, crc=1
		         tx pkts=1, bytes=1, drop=1, errs=1, coll=1
		port  2: rx pkts=2, bytes=2, drop=2, errs=2, frame=2, over=2, crc=2
		         tx pkts=2, bytes=2, drop=2, errs=2, coll=2
		`), nil
	})

	got, err := c.OpenFlow.DumpPorts(bridge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected stats:\n- want: %+v\n-  got: %+v",
			want, got)
	}
}

func TestClientOpenFlowDumpPortsOpenFlow14OK(t *testing.T) {
	want := []*PortStats{
		{
			PortID: 1,
			Received: PortStatsReceive{
				Packets: 1,
				Bytes:   1,
				Dropped: 1,
				Errors:  1,
				Frame:   1,
				Over:    1,
				CRC:     1,
			},
			Transmitted: PortStatsTransmit{
				Packets:    1,
				Bytes:      1,
				Dropped:    1,
				Errors:     1,
				Collisions: 1,
			},
		},
		{
			PortID: 2,
			Received: PortStatsReceive{
				Packets: 2,
				Bytes:   2,
				Dropped: 2,
				Errors:  2,
				Frame:   2,
				Over:    2,
				CRC:     2,
			},
			Transmitted: PortStatsTransmit{
				Packets:    2,
				Bytes:      2,
				Dropped:    2,
				Errors:     2,
				Collisions: 2,
			},
		},
	}

	const bridge = "br0"

	options := []OptionFunc{
		Protocols([]string{ProtocolOpenFlow14}),
		FlowFormat(FlowFormatOXMOpenFlow14),
	}

	c := testClient(options, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--protocols=OpenFlow14", "--flow-format=OXM-OpenFlow14", "dump-ports", bridge}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return []byte(`
		OFPST_PORT reply (OF1.4) (xid=0x1): 1 ports
		port  1: rx pkts=1, bytes=1, drop=1, errs=1, frame=1, over=1, crc=1
		         tx pkts=1, bytes=1, drop=1, errs=1, coll=1
		         duration=1.001s
		port  2: rx pkts=2, bytes=2, drop=2, errs=2, frame=2, over=2, crc=2
		         tx pkts=2, bytes=2, drop=2, errs=2, coll=2
		         duration=2.002s
		`), nil
	})

	got, err := c.OpenFlow.DumpPorts(bridge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected stats:\n- want: %+v\n-  got: %+v",
			want, got)
	}
}

func TestClientOpenFlowDumpTablesInvalidTable(t *testing.T) {
	c := testClient(nil, func(cmd string, args ...string) ([]byte, error) {
		return []byte("OFPST_TABLE reply\n0: classifier\nfoo"), nil
	})

	want := ErrInvalidTable
	if _, got := c.OpenFlow.DumpTables("foo"); !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientOpenFlowDumpTablesOK(t *testing.T) {
	want := []*Table{
		{
			ID:      0,
			Name:    "classifier",
			Wild:    "0x3fffff",
			Max:     1000000,
			Active:  1,
			Lookup:  2,
			Matched: 3,
		},
		{
			ID:      1,
			Name:    "table1",
			Wild:    "0x3fffff",
			Max:     1000000,
			Active:  4,
			Lookup:  5,
			Matched: 6,
		},
	}

	bridge := "br0"

	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-ofctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "dump-tables", string(bridge)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		// Last table should be ignored
		return []byte(`
		OFPST_TABLE reply (xid=0x2): 3 tables
		  0: classifier: wild=0x3fffff, max=1000000, active=1
		                 lookup=2, matched=3
		  1: table1  :   wild=0x3fffff, max=1000000, active=4
		                 lookup=5, matched=6
		  2: table2  :   wild=0x3fffff, max=1000000, active=0
		                 lookup=0, matched=0
		`), nil
	})

	got, err := c.OpenFlow.DumpTables(bridge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected tables:\n- want: %+v\n-  got: %+v",
			want, got)
	}
}

func Test_parseEachUnexpectedEOFFirstLine(t *testing.T) {
	c := testClient(nil, func(cmd string, args ...string) ([]byte, error) {
		return nil, nil
	})

	want := io.ErrUnexpectedEOF
	if _, got := c.OpenFlow.DumpPorts("foo"); !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func Test_parseEachUnexpectedEOFIncorrectPrefix(t *testing.T) {
	c := testClient(nil, func(cmd string, args ...string) ([]byte, error) {
		return []byte("foo"), nil
	})

	want := io.ErrUnexpectedEOF
	if _, got := c.OpenFlow.DumpPorts("foo"); !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func Test_parseEachUnexpectedEOFOddLineCount(t *testing.T) {
	c := testClient(nil, func(cmd string, args ...string) ([]byte, error) {
		return []byte("OFPST_PORT reply\nfoo"), nil
	})

	want := io.ErrUnexpectedEOF
	if _, got := c.OpenFlow.DumpPorts("foo"); !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientOpenFlowDumpFlows(t *testing.T) {
	tests := []struct {
		name  string
		input string
		flows string
		want  []*Flow
		err   error
	}{
		{
			name:  "test single flow",
			input: "br0",
			flows: `NXST_FLOW reply (xid=0x4):
 cookie=0x0, duration=9215.748s, table=0, n_packets=6, n_bytes=480, idle_age=9206, priority=820,in_port=LOCAL actions=mod_vlan_vid:10,output:1
`,
			want: []*Flow{
				{
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
			err: nil,
		},
		{
			name:  "test multiple flows",
			input: "br0",
			flows: `NXST_FLOW reply (xid=0x4):
 cookie=0x0, duration=9215.748s, table=0, n_packets=6, n_bytes=480, idle_age=9206, priority=820,in_port=LOCAL actions=mod_vlan_vid:10,output:1
 cookie=0x0, duration=1121991.329s, table=50, n_packets=0, n_bytes=0, priority=110,ip,dl_src=f1:f2:f3:f4:f5:f6 actions=ct(table=51)
 cookie=0x0, duration=83229.846s, table=51, n_packets=3, n_bytes=234, priority=101,ct_state=+new+rel+trk,ip actions=ct(commit,table=65)
 cookie=0x0, duration=1381314.983s, table=65, n_packets=0, n_bytes=0, priority=4040,ip,dl_dst=f1:f2:f3:f4:f5:f6,nw_src=169.254.169.254,nw_dst=169.254.0.0/16 actions=output:19
  cookie=0x0, duration=13.265s, table=12, n_packets=0, n_bytes=0, idle_age=13, priority=4321,tcp,tcp_flags=+syn-psh+ack actions=resubmit(,13)
`,
			want: []*Flow{
				{
					Priority: 820,
					InPort:   PortLOCAL,
					Matches:  []Match{},
					Table:    0,
					Actions: []Action{
						ModVLANVID(10),
						Output(1),
					},
				},
				{
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
				{
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
				{
					Priority: 4040,
					Protocol: ProtocolIPv4,
					Matches: []Match{
						DataLinkDestination("f1:f2:f3:f4:f5:f6"),
						NetworkSource("169.254.169.254"),
						NetworkDestination("169.254.0.0/16"),
					},
					Table: 65,
					Actions: []Action{
						Output(19),
					},
				},
				{
					Priority: 4321,
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
			err: nil,
		},
		{
			name:  "test multiple flows mid dump NXST_FLOW",
			input: "br0",
			flows: `NXST_FLOW reply (xid=0x4): flags=[more]
 cookie=0x0, duration=9215.748s, table=0, n_packets=6, n_bytes=480, idle_age=9206, priority=820,in_port=LOCAL actions=mod_vlan_vid:10,output:1
 cookie=0x0, duration=1121991.329s, table=50, n_packets=0, n_bytes=0, priority=110,ip,dl_src=f1:f2:f3:f4:f5:f6 actions=ct(table=51)
NXST_FLOW reply (xid=0x4):
 cookie=0x0, duration=83229.846s, table=51, n_packets=3, n_bytes=234, priority=101,ct_state=+new+rel+trk,ip actions=ct(commit,table=65)
 cookie=0x0, duration=1381314.983s, table=65, n_packets=0, n_bytes=0, priority=4040,ip,dl_dst=f1:f2:f3:f4:f5:f6,nw_src=169.254.169.254,nw_dst=169.254.0.0/16 actions=output:19
  cookie=0x0, duration=13.265s, table=12, n_packets=0, n_bytes=0, idle_age=13, priority=4321,tcp,tcp_flags=+syn-psh+ack actions=resubmit(,13)
`,
			want: []*Flow{
				{
					Priority: 820,
					InPort:   PortLOCAL,
					Matches:  []Match{},
					Table:    0,
					Actions: []Action{
						ModVLANVID(10),
						Output(1),
					},
				},
				{
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
				{
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
				{
					Priority: 4040,
					Protocol: ProtocolIPv4,
					Matches: []Match{
						DataLinkDestination("f1:f2:f3:f4:f5:f6"),
						NetworkSource("169.254.169.254"),
						NetworkDestination("169.254.0.0/16"),
					},
					Table: 65,
					Actions: []Action{
						Output(19),
					},
				},
				{
					Priority: 4321,
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
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
				if want, got := "ovs-ofctl", cmd; want != got {
					t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
						want, got)
				}
				wantArgs := []string{
					"--timeout=1",
					"dump-flows",
					string(tt.input),
				}
				if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
					t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
						want, got)
				}
				return []byte(tt.flows), tt.err
			}).OpenFlow.DumpFlows(tt.input)
			if len(tt.want) != len(got) {
				t.Errorf("got  %d", len(got))
				t.Errorf("want %d", len(tt.want))
				t.Fatal("expected return value to be equal")
			}
			for i := range tt.want {
				if !flowsEqual(tt.want[i], got[i]) {
					t.Errorf("got  %v", got[i])
					t.Errorf("want %v", tt.want[i])
					t.Fatal("expected return value to be equal")
				}
			}
		})
	}
}

func mustVerifyFlowBundle(t *testing.T, stdin io.Reader, flows []*Flow, matchFlows []*MatchFlow) {
	s := bufio.NewScanner(stdin)
	var gotFlows []*Flow
	var gotMatchFlows []string

	for s.Scan() {
		bb := bytes.Fields(s.Bytes())
		if want, got := 2, len(bb); want != got {
			t.Fatalf("unexpected number of fields in flow bundle:\n- want: %d\n-  got: %d",
				want, got)
		}

		keyword := string(bb[0])
		switch keyword {
		case dirAdd:
			flow := &Flow{}
			if err := flow.UnmarshalText(bb[1]); err != nil {
				t.Fatalf("failed to unmarshal flow: %v", err)
			}

			gotFlows = append(gotFlows, flow)
		case dirDelete:
			gotMatchFlows = append(gotMatchFlows, string(bb[1]))
		default:
			t.Fatalf("unexpected directive in flow bundle: %q", keyword)
		}
	}

	if err := s.Err(); err != nil {
		t.Fatalf("failed to scan: %v", err)
	}

	if want, got := len(flows), len(gotFlows); want != got {
		t.Fatalf("unexpected number of flows:\n- want: %d\n-  got: %d", want, got)
	}

	for i := range flows {
		if want, got := flows[i], gotFlows[i]; !flowsEqual(want, got) {
			t.Fatalf("[%02d] unexpected flows:\n- want: %v\n-  got: %v",
				i, want, got)
		}
	}

	if want, got := len(matchFlows), len(gotMatchFlows); want != got {
		t.Fatalf("unexpected number of match flows:\n- want: %d\n-  got: %d", want, got)
	}

	for i := range matchFlows {
		// TODO(mdlayher): workaround for the fact that we don't have MatchFlow
		// unmarshaling code.
		mfb, err := matchFlows[i].MarshalText()
		if err != nil {
			t.Fatalf("failed to marshal MatchFlow: %v", err)
		}

		if want, got := string(mfb), gotMatchFlows[i]; want != got {
			t.Fatalf("[%02d] unexpected match flows:\n- want: %q\n-  got: %q",
				i, want, got)
		}
	}
}
