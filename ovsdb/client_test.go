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
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsdb"
	"github.com/google/go-cmp/cmp"
)

func TestClientError(t *testing.T) {
	const str = "some error"

	c, done := testClient(t, func(_ string, _ []interface{}) interface{} {
		return &ovsdb.Error{
			Err:     str,
			Details: "malformed",
			Syntax:  "{}",
		}
	})
	defer done()

	_, err := c.ListDatabases()
	if err == nil {
		t.Fatal("expected an error, but none occurred")
	}

	oerr, ok := err.(*ovsdb.Error)
	if !ok {
		t.Fatalf("error of wrong type: %#v", err)
	}

	if diff := cmp.Diff(str, oerr.Err); diff != "" {
		t.Fatalf("unexpected error (-want +got):\n%s", diff)
	}
}
func TestClientListDatabases(t *testing.T) {
	want := []string{"Open_vSwitch", "test"}

	c, done := testClient(t, func(method string, params []interface{}) interface{} {
		if diff := cmp.Diff("list_dbs", method); diff != "" {
			t.Fatalf("unexpected RPC method (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(1, len(params)); diff != "" {
			t.Fatalf("unexpected number of RPC parameters (-want +got):\n%s", diff)
		}

		return want
	})
	defer done()

	dbs, err := c.ListDatabases()
	if err != nil {
		t.Fatalf("failed to list databases: %v", err)
	}

	if diff := cmp.Diff(want, dbs); diff != "" {
		t.Fatalf("unexpected databases (-want +got):\n%s", diff)
	}
}

type rpcFunc func(method string, params []interface{}) interface{}

func testClient(t *testing.T, fn rpcFunc) (*ovsdb.Client, func()) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Accept a single connection.
		c, err := l.Accept()
		if err != nil {
			panicf("failed to accept: %v", err)
		}
		defer c.Close()
		_ = l.Close()

		if err := handleConn(c, fn); err != nil {
			panicf("failed to handle connection: %v", err)
		}
	}()

	c, err := ovsdb.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	return c, func() {
		// Ensure types are cleaned up, and ensure goroutine stops.
		_ = l.Close()
		_ = c.Close()
		wg.Wait()
	}
}

func handleConn(c net.Conn, fn rpcFunc) error {
	var req struct {
		Method string        `json:"method"`
		Params []interface{} `json:"params"`
		ID     int           `json:"id"`
	}

	var res struct {
		Result interface{} `json:"result"`
		ID     int         `json:"id"`
	}

	if err := json.NewDecoder(c).Decode(&req); err != nil {
		return err
	}

	result := fn(req.Method, req.Params)

	res.ID = req.ID
	res.Result = result

	return json.NewEncoder(c).Encode(res)
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
