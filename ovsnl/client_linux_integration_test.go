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

//+build linux

package ovsnl_test

import (
	"net"
	"os"
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsnl"
	"github.com/google/go-cmp/cmp"
)

func TestLinuxClientIntegration(t *testing.T) {
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
