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

package ovsdb_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/digitalocean/go-openvswitch/ovsdb"
)

// This example demonstrates basic usage of a Client.  The Client connects to
// ovsdb-server and requests a list of all databases known to the server.
func ExampleClient_listDatabases() {
	// Dial an OVSDB connection and create a *ovsdb.Client.
	c, err := ovsdb.Dial("unix", "/var/run/openvswitch/db.sock")
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	// Be sure to close the connection!
	defer c.Close()

	// Ask ovsdb-server for all of its databases, but only allow the RPC
	// a limited amount of time to complete before timing out.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dbs, err := c.ListDatabases(ctx)
	if err != nil {
		log.Fatalf("failed to list databases: %v", err)
	}

	for _, d := range dbs {
		fmt.Println(d)
	}
}
