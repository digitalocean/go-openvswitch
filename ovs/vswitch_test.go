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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestClientVSwitchAddBridgeOK(t *testing.T) {
	bridge := "br0"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "--may-exist", "add-br", string(bridge)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.VSwitch.AddBridge(bridge); err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.AddBridge: %v", err)
	}
}

func TestClientVSwitchAddPortOK(t *testing.T) {
	bridge := "br0"
	port := "bond0"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "--may-exist", "add-port", string(bridge), string(port)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.VSwitch.AddPort(bridge, port); err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.AddPort: %v", err)
	}
}

func TestClientVSwitchDeleteBridgeOK(t *testing.T) {
	bridge := "br0"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "--if-exists", "del-br", string(bridge)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.VSwitch.DeleteBridge(bridge); err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.DeleteBridge: %v", err)
	}
}

func TestClientVSwitchDeletePortError(t *testing.T) {
	want := &Error{
		Out: []byte("foo"),
		Err: errors.New("bar"),
	}

	c := testClient(nil, func(cmd string, args ...string) ([]byte, error) {
		return want.Out, want.Err
	})

	if got := c.VSwitch.DeletePort("foo", "bar"); !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientVSwitchDeletePortOK(t *testing.T) {
	bridge := "br0"
	port := "bond0"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "--if-exists", "del-port", string(bridge), string(port)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.VSwitch.DeletePort(bridge, port); err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.DeletePort: %v", err)
	}
}

func TestClientVSwitchSetControllerOK(t *testing.T) {
	bridge := "br0"
	address := "pssl:6653:127.0.0.1"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "set-controller", string(bridge), address}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.VSwitch.SetController(bridge, address); err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.SetController: %v", err)
	}
}

func TestClientVSwitchGetControllerOK(t *testing.T) {
	bridge := "br0"
	address := "pssl:6653:127.0.0.1"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "get-controller", string(bridge)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return []byte(address), nil
	})

	gotAddress, err := c.VSwitch.GetController(bridge)
	if err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.GetController: %v", err)
	}
	if gotAddress != address {
		t.Fatalf("Controller address missmatch\n- got: %v\n- want: %v", gotAddress, address)
	}
}

func TestClientVSwitchListPorts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
		err   error
		c     *Client
	}{
		{
			name:  "test br0 bonded",
			input: "br0",
			want:  []string{"bond0"},
			err:   nil,
			c: testClient(nil, func(cmd string, args ...string) ([]byte, error) {
				return []byte("bond0\n"), nil
			}),
		},
		{
			name:  "test br0 bonded with other interfaces",
			input: "br0",
			want:  []string{"bond0", "eth0", "eth1"},
			err:   nil,
			c: testClient(nil, func(cmd string, args ...string) ([]byte, error) {
				return []byte("bond0\neth0\neth1"), nil
			}),
		},
		{
			name:  "test wrong bridge",
			input: "foo",
			want:  nil,
			err: &Error{
				Out: nil,
				Err: errors.New("foo"),
			},
			c: testClient(nil, func(cmd string, args ...string) ([]byte, error) {
				return nil, errors.New(args[1])
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.VSwitch.ListPorts(tt.input)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("got  %v", got)
				t.Errorf("want %v", tt.want)
				t.Fatal("expected return value to be equal")
			}
			if !reflect.DeepEqual(tt.err, err) {
				t.Errorf("got  %v", err)
				t.Errorf("want %v", tt.err)
				t.Fatal("expected error value to be equal")
			}
		})
	}
}

func TestClientVSwitchListBridges(t *testing.T) {
	tests := []struct {
		name string
		want []string
		err  error
		c    *Client
	}{
		{
			name: "test single bridge",
			want: []string{"br0"},
			err:  nil,
			c: testClient(nil, func(cmd string, args ...string) ([]byte, error) {
				return []byte("br0\n"), nil
			}),
		},
		{
			name: "test multi bridge",
			want: []string{"br0", "br1"},
			err:  nil,
			c: testClient(nil, func(cmd string, args ...string) ([]byte, error) {
				return []byte("br0\nbr1"), nil
			}),
		},
		{
			name: "test wrong bridge",
			want: nil,
			err: &Error{
				Out: nil,
				Err: errors.New(""),
			},
			c: testClient(nil, func(cmd string, args ...string) ([]byte, error) {
				return nil, errors.New("")
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.VSwitch.ListBridges()
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("got  %v", got)
				t.Errorf("want %v", tt.want)
				t.Fatal("expected return value to be equal")
			}
			if !reflect.DeepEqual(tt.err, err) {
				t.Errorf("got  %v", err)
				t.Errorf("want %v", tt.err)
				t.Fatal("expected error value to be equal")
			}
		})
	}
}

func TestClientVSwitchPortToBridgeOK(t *testing.T) {
	port := "bond0"
	wantBr := "br0"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "port-to-br", string(port)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		// Verify whitespace trimmed appropriately
		return []byte("\n\n  " + wantBr + "\t\n "), nil
	})

	br, err := c.VSwitch.PortToBridge(port)
	if err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.PortToBridge: %v", err)
	}

	if want, got := wantBr, br; want != got {
		t.Fatalf("unexpected bridge:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientVSwitchGetFailModeOK(t *testing.T) {
	bridge := "br0"
	mode := FailModeSecure

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "get-fail-mode", string(bridge)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		// Make the return value with newline to simulate
		// the ovs-vsctl output.
		return []byte(fmt.Sprintln(mode)), nil
	})

	got, err := c.VSwitch.GetFailMode(bridge)
	if err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.GetFailMode: %v", err)
	}
	if FailMode(got) != mode {
		t.Fatalf("unexpected mode for Client.VSwitch.GetFailMode: %v", got)
	}
}

func TestClientVSwitchSetFailModeOK(t *testing.T) {
	bridge := "br0"
	mode := FailModeSecure

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"--timeout=1", "set-fail-mode", string(bridge), string(mode)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.VSwitch.SetFailMode(bridge, mode); err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.SetFailMode: %v", err)
	}
}

func TestClientVSwitchGetBridgeProtocolsOK(t *testing.T) {
	const bridge = "br0"
	protocols := []string{
		ProtocolOpenFlow10,
		ProtocolOpenFlow11,
		ProtocolOpenFlow12,
		ProtocolOpenFlow13,
		ProtocolOpenFlow14,
		ProtocolOpenFlow15,
	}

	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"--format=json",
			"get",
			"bridge",
			bridge,
			"protocols",
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		// Make the return value with newline to simulate
		// the ovs-vsctl output.
		data, err := json.Marshal(&protocols)
		if err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintln(string(data))), err
	})

	got, err := c.VSwitch.Get.Bridge(bridge)
	if err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.Get.Bridge: %v", err)
	}
	if !reflect.DeepEqual(got.Protocols, protocols) {
		t.Fatalf("unexpected protocols for Client.VSwitch.Get.Bridge: %v", got)
	}
}

func TestClientVSwitchSetBridgeProtocolsOK(t *testing.T) {
	const bridge = "br0"
	protocols := []string{
		ProtocolOpenFlow10,
		ProtocolOpenFlow11,
		ProtocolOpenFlow12,
		ProtocolOpenFlow13,
		ProtocolOpenFlow14,
		ProtocolOpenFlow15,
	}

	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"set",
			"bridge",
			bridge,
			fmt.Sprintf("protocols=%s", strings.Join(protocols, ",")),
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	err := c.VSwitch.Set.Bridge(bridge, BridgeOptions{
		Protocols: protocols,
	})
	if err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.Set.Bridge: %v", err)
	}
}

func TestClientVSwitchSetInterfaceTypeOK(t *testing.T) {
	ifi := "bond0"
	ifiType := InterfaceTypePatch

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"set",
			"interface",
			string(ifi),
			fmt.Sprintf("type=%s", ifiType),
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	err := c.VSwitch.Set.Interface(ifi, InterfaceOptions{
		Type: ifiType,
	})
	if err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.Set.Interface: %v", err)
	}
}

func TestClientVSwitchSetInterfacePeerOK(t *testing.T) {
	ifi := "bond0"
	peer := "eth0"

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"set",
			"interface",
			string(ifi),
			fmt.Sprintf("options:peer=%s", peer),
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	err := c.VSwitch.Set.Interface(ifi, InterfaceOptions{
		Peer: peer,
	})
	if err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.Set.Interface: %v", err)
	}
}

func TestClientVSwitchSetInterfacePolicingOK(t *testing.T) {
	tests := []struct {
		name  string
		rate  int64
		burst int64
	}{
		{
			name: "no rate nor burst policing",
		},
		{
			name:  "default rate and burst policing",
			rate:  DefaultIngressRatePolicing,
			burst: DefaultIngressBurstPolicing,
		},
		{
			name: "only rate policing (2Gbps)",
			rate: 2000000,
		},
		{
			name:  "only burst policing (100Mb)",
			burst: 100000,
		},
		{
			name:  "rate (5Gbps) and burst (500Mb) policing",
			rate:  5000000,
			burst: 500000,
		},
	}

	opts := []OptionFunc{Timeout(1)}
	ifi := "tap0"

	for _, tt := range tests {
		c := testClient(opts, func(cmd string, args ...string) ([]byte, error) {
			// Verify correct command and arguments passed,
			// including option flags.
			if want, got := "ovs-vsctl", cmd; want != got {
				t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
					want, got)
			}

			wantArgs := []string{
				"--timeout=1",
				"set",
				"interface",
				string(ifi),
			}
			if tt.rate == DefaultIngressRatePolicing {
				wantArgs = append(wantArgs, "ingress_policing_rate=0")
			} else if tt.rate > 0 {
				wantArgs = append(wantArgs,
					fmt.Sprintf("ingress_policing_rate=%d", tt.rate))
			}
			if tt.burst == DefaultIngressBurstPolicing {
				wantArgs = append(wantArgs, "ingress_policing_burst=0")
			} else if tt.burst > 0 {
				wantArgs = append(wantArgs,
					fmt.Sprintf("ingress_policing_burst=%d", tt.burst))
			}

			if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
				t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
					want, got)
			}

			return nil, nil
		})
		t.Run(tt.name, func(t *testing.T) {
			err := c.VSwitch.Set.Interface(ifi, InterfaceOptions{
				IngressRatePolicing:  tt.rate,
				IngressBurstPolicing: tt.burst,
			})
			if err != nil {
				t.Fatalf("unexpected error for Client.VSwitch.Set.Interface: %v", err)
			}
		})
	}
}

func TestClientVSwitchSetInterfaceOK(t *testing.T) {
	ifi := "bond0"
	ifiType := InterfaceTypePatch
	peer := "eth0"
	ratePolicing := DefaultIngressRatePolicing
	burstPolicing := DefaultIngressBurstPolicing

	// Apply Timeout option to verify arguments
	c := testClient([]OptionFunc{Timeout(1)}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "ovs-vsctl", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{
			"--timeout=1",
			"set",
			"interface",
			string(ifi),
			fmt.Sprintf("type=%s", ifiType),
			fmt.Sprintf("options:peer=%s", peer),
			"ingress_policing_rate=0",
			"ingress_policing_burst=0",
		}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	err := c.VSwitch.Set.Interface(ifi, InterfaceOptions{
		Type:                 ifiType,
		Peer:                 peer,
		IngressRatePolicing:  ratePolicing,
		IngressBurstPolicing: burstPolicing,
	})
	if err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.Set.Interface: %v", err)
	}
}

func TestClientVSwitchSudoOK(t *testing.T) {
	const bridge = "br0"

	c := testClient([]OptionFunc{Sudo()}, func(cmd string, args ...string) ([]byte, error) {
		// Verify correct command and arguments passed, including option flags
		if want, got := "sudo", cmd; want != got {
			t.Fatalf("incorrect command:\n- want: %v\n-  got: %v",
				want, got)
		}

		wantArgs := []string{"ovs-vsctl", "--may-exist", "add-br", string(bridge)}
		if want, got := wantArgs, args; !reflect.DeepEqual(want, got) {
			t.Fatalf("incorrect arguments\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})

	if err := c.VSwitch.AddBridge(bridge); err != nil {
		t.Fatalf("unexpected error for Client.VSwitch.AddBridge: %v", err)
	}
}

func TestBridgeOptions_slice(t *testing.T) {
	var tests = []struct {
		desc string
		o    BridgeOptions
		out  []string
	}{
		{
			desc: "no options",
		},
		{
			desc: "one protocol",
			o: BridgeOptions{
				Protocols: []string{ProtocolOpenFlow14},
			},
			out: []string{"protocols=OpenFlow14"},
		},
		{
			desc: "many protocols",
			o: BridgeOptions{
				Protocols: []string{
					ProtocolOpenFlow10,
					ProtocolOpenFlow11,
					ProtocolOpenFlow12,
					ProtocolOpenFlow13,
					ProtocolOpenFlow14,
					ProtocolOpenFlow15,
				},
			},
			out: []string{"protocols=OpenFlow10,OpenFlow11,OpenFlow12,OpenFlow13,OpenFlow14,OpenFlow15"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if want, got := tt.out, tt.o.slice(); !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected slices:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func TestInterfaceOptions_slice(t *testing.T) {
	var tests = []struct {
		desc string
		i    InterfaceOptions
		out  []string
	}{
		{
			desc: "no options",
		},
		{
			desc: "only Type",
			i: InterfaceOptions{
				Type: InterfaceTypePatch,
			},
			out: []string{
				"type=patch",
			},
		},
		{
			desc: "only Peer",
			i: InterfaceOptions{
				Peer: "bond0",
			},
			out: []string{
				"options:peer=bond0",
			},
		},
		{
			desc: "only ingress policing rate",
			i: InterfaceOptions{
				IngressRatePolicing: 2000000,
			},
			out: []string{
				"ingress_policing_rate=2000000",
			},
		},
		{
			desc: "default ingress policing rate",
			i: InterfaceOptions{
				IngressRatePolicing: DefaultIngressRatePolicing,
			},
			out: []string{
				"ingress_policing_rate=0",
			},
		},
		{
			desc: "only ingress policing burst",
			i: InterfaceOptions{
				IngressBurstPolicing: 200000,
			},
			out: []string{
				"ingress_policing_burst=200000",
			},
		},
		{
			desc: "default ingress policing rate",
			i: InterfaceOptions{
				IngressRatePolicing: DefaultIngressRatePolicing,
			},
			out: []string{
				"ingress_policing_rate=0",
			},
		},
		{
			desc: "default ingress policing burst",
			i: InterfaceOptions{
				IngressBurstPolicing: DefaultIngressBurstPolicing,
			},
			out: []string{
				"ingress_policing_burst=0",
			},
		},
		{
			desc: "flow based STT tunnel",
			i: InterfaceOptions{
				Type:     InterfaceTypeSTT,
				RemoteIP: "flow",
				Key:      "flow",
			},
			out: []string{
				"type=stt",
				"options:remote_ip=flow",
				"options:key=flow",
			},
		},
		{
			desc: "all options",
			i: InterfaceOptions{
				Type:                 InterfaceTypePatch,
				Peer:                 "bond0",
				IngressRatePolicing:  2000000,
				IngressBurstPolicing: 200000,
			},
			out: []string{
				"type=patch",
				"options:peer=bond0",
				"ingress_policing_rate=2000000",
				"ingress_policing_burst=200000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if want, got := tt.out, tt.i.slice(); !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected slices:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}
