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
	"sync"
	"testing"
	"time"

	"github.com/digitalocean/go-openvswitch/ovsdb"
	"github.com/google/go-cmp/cmp"
)

func TestClientIntegration(t *testing.T) {
	c := dialOVSDB(t)
	defer c.Close()

	// Cancel RPCs if they take too long.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("echo", func(t *testing.T) {
		testClientEcho(ctx, t, c)
	})

	t.Run("databases", func(t *testing.T) {
		testClientDatabases(ctx, t, c)
	})
}

func TestClientIntegrationConcurrent(t *testing.T) {
	c := dialOVSDB(t)
	defer c.Close()

	const n = 512

	// Wait for all goroutines to start before performing RPCs,
	// wait for them all to exit before ending the test.
	var startWG, doneWG sync.WaitGroup
	startWG.Add(n)
	doneWG.Add(n)

	// Block all goroutines until they're done spinning up.
	sigC := make(chan struct{}, 0)

	for i := 0; i < n; i++ {
		go func(c *ovsdb.Client) {
			// Block goroutines until all are spun up.
			startWG.Done()
			<-sigC

			for j := 0; j < 4; j++ {
				_, err := c.ListDatabases(context.Background())
				if err != nil {
					panic(fmt.Sprintf("failed to query concurrently: %v", err))
				}
			}

			doneWG.Done()
		}(c)
	}

	// Unblock all goroutines once they're all spun up, and wait
	// for them all to finish reading.
	startWG.Wait()
	close(sigC)
	doneWG.Wait()
}

func testClientDatabases(ctx context.Context, t *testing.T, c *ovsdb.Client) {
	dbs, err := c.ListDatabases(ctx)
	if err != nil {
		t.Fatalf("failed to list databases: %v", err)
	}

	want := []string{"Open_vSwitch"}

	if diff := cmp.Diff(want, dbs); diff != "" {
		t.Fatalf("unexpected databases (-want +got):\n%s", diff)
	}

	for _, d := range dbs {
		rows, err := c.Transact(ctx, d, []ovsdb.TransactOp{
			ovsdb.Select{
				Table: "Bridge",
			},
		})
		if err != nil {
			t.Fatalf("failed to perform transaction: %v", err)
		}

		for i, r := range rows {
			t.Logf("[%02d] %v", i, r)
		}
	}
}

func testClientEcho(ctx context.Context, t *testing.T, c *ovsdb.Client) {
	if err := c.Echo(ctx); err != nil {
		t.Fatalf("failed to echo: %v", err)
	}
}

func dialOVSDB(t *testing.T) *ovsdb.Client {
	t.Helper()

	// Assume the standard Linux location for the socket.
	const sock = "/var/run/openvswitch/db.sock"
	c, err := ovsdb.Dial("unix", sock)
	if err != nil {
		t.Skipf("could not access %q: %v", sock, err)
	}

	return c
}
