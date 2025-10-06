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

//go:build linux && integration
// +build linux,integration

package ovsnl

import (
	"testing"
)

// TestClientDatapathListIntegration tests datapath listing with real Open vSwitch
func TestClientDatapathListIntegration(t *testing.T) {
	// Skip if not running in integration test environment
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	c, err := NewClient()
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	defer c.Close()

	dps, err := c.Datapath.List()
	if err != nil {
		t.Fatalf("failed to list datapaths: %v", err)
	}

	if len(dps) == 0 {
		t.Log("no datapaths found (Open vSwitch may not be running)")
		return
	}

	// Verify we can list datapaths
	t.Logf("found %d datapaths", len(dps))
	for _, dp := range dps {
		t.Logf("datapath: %s (index: %d)", dp.Name, dp.Index)
	}
}
