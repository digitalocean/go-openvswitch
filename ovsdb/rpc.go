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

package ovsdb

import (
	"context"
	"fmt"
)

// ListDatabases returns the name of all databases known to the OVSDB server.
func (c *Client) ListDatabases(ctx context.Context) ([]string, error) {
	var dbs []string
	if err := c.rpc(ctx, "list_dbs", &dbs, nil); err != nil {
		return nil, err
	}

	return dbs, nil
}

// Echo verifies that the OVSDB connection is alive, and can be used to keep
// the connection alive.
func (c *Client) Echo(ctx context.Context) error {
	req := [1]string{"github.com/digitalocean/go-openvswitch/ovsdb"}

	var res [1]string
	if err := c.rpc(ctx, "echo", &res, req); err != nil {
		return err
	}

	if res[0] != req[0] {
		return fmt.Errorf("invalid echo response: %q", res[0])
	}

	return nil
}
