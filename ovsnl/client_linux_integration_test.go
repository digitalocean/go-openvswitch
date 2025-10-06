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

//go:build linux
// +build linux

package ovsnl_test

import (
	"net"
	"os"
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/google/go-cmp/cmp"
)

func TestLinuxClientIntegration(t *testing.T) {

	// Skip this test in CI or other automated environments where the OVS
	// kernel/netlink families are not present. GitHub Actions (and many CI
	// runners) set CI=true; additionally users can set SKIP_LIVE_TESTS=1 to
	// force skipping locally.
	// if os.Getenv("CI") == "true" || os.Getenv("SKIP_LIVE_TESTS") == "1" {
	// 	t.Skip("Skipping live OVS integration tests in CI / SKIP_LIVE_TESTS environment")
	// }

	c, err := ovsnl.New()
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("generic netlink OVS families not found: %v", err)
		}

		t.Fatalf("failed to create client %v", err)
	}
	defer c.Close()

	const (
		ovsSystem = "ovs-system"
		ovsBridge = "ovsbr0"
	)

	// Ensure required interfaces exist for remaining tests.
	for _, ifi := range []string{ovsSystem, ovsBridge} {
		if _, err := net.InterfaceByName(ifi); err != nil {
			t.Skipf("failed to check for OVS interface %q: %v", ifi, err)
		}
	}

	t.Run("datapath", func(t *testing.T) {
		testClientDatapath(t, c, ovsSystem)
	})
}

func testClientDatapath(t *testing.T, c *ovsnl.Client, datapath string) {
	dps, err := c.Datapath.List()
	if err != nil {
		t.Fatalf("failed to list datapaths: %v", err)
	}

	if diff := cmp.Diff(1, len(dps)); diff != "" {
		t.Fatalf("unexpected number of datapaths (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(datapath, dps[0].Name); diff != "" {
		t.Fatalf("unexpected datapath name (-want +got):\n%s", diff)
	}
}

// func TestClientDatapathListShortHeader(t *testing.T) {
// 	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
// 		// Not enough data for ovsh.Header - this should trigger an error
// 		return []genetlink.Message{
// 			{
// 				Data: []byte{0xff, 0xff}, // Only 2 bytes, but header needs more
// 			},
// 		}, nil
// 	}))

// 	c, err := ovsnl.New()
// 	if err != nil {
// 		t.Fatalf("failed to create client: %v", err)
// 	}
// 	defer c.Close()

// 	_, err = c.Datapath.List()
// 	if err == nil {
// 		t.Fatalf("expected an error due to short header, but none occurred")
// 	}

// 	t.Logf("OK error: %v", err)
// }

// ovsFamilies creates a test handler that returns OVS family messages
// func ovsFamilies(handler func(genetlink.Message, netlink.Message) ([]genetlink.Message, error)) func(genetlink.Message, netlink.Message) ([]genetlink.Message, error) {
// 	return func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
// 		// Handle family listing requests
// 		if greq.Header.Command == unix.CTRL_CMD_GETFAMILY {
// 			return familyMessages([]string{
// 				ovsh.DatapathFamily,
// 			}), nil
// 		}

// 		// Handle actual datapath requests
// 		return handler(greq, nreq)
// 	}
// }

// func familyMessages(families []string) []genetlink.Message {
// 	msgs := make([]genetlink.Message, 0, len(families))

// 	var id uint16
// 	for _, f := range families {
// 		msgs = append(msgs, genetlink.Message{
// 			Data: mustMarshalAttributes([]netlink.Attribute{
// 				{
// 					Type: unix.CTRL_ATTR_FAMILY_ID,
// 					Data: nlenc.Uint16Bytes(id),
// 				},
// 				{
// 					Type: unix.CTRL_ATTR_FAMILY_NAME,
// 					Data: nlenc.Bytes(f),
// 				},
// 			}),
// 		})

// 		id++
// 	}

// 	return msgs
// }

// func mustMarshalAttributes(attrs []netlink.Attribute) []byte {
// 	b, err := netlink.MarshalAttributes(attrs)
// 	if err != nil {
// 		panic(fmt.Sprintf("failed to marshal attributes: %v", err))
// 	}

// 	return b
// }
