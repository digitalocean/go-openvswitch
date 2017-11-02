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
	"os"
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsnl"
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

	// TODO(mdlayher): fill in after Client has more methods.
	_ = c
}
