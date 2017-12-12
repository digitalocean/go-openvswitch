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
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsdb"
	"github.com/digitalocean/go-openvswitch/ovsdb/internal/jsonrpc"
	"github.com/google/go-cmp/cmp"
)

func TestClientListDatabases(t *testing.T) {
	want := []string{"Open_vSwitch", "test"}

	c, _, done := testClient(t, func(req jsonrpc.Request) jsonrpc.Response {
		if diff := cmp.Diff("list_dbs", req.Method); diff != "" {
			panicf("unexpected RPC method (-want +got):\n%s", diff)
		}

		// Client should send an empty array parameter.
		ps := req.Params.([]interface{})

		if diff := cmp.Diff(0, len(ps)); diff != "" {
			panicf("unexpected number of RPC parameters (-want +got):\n%s", diff)
		}

		return jsonrpc.Response{
			ID:     strPtr("1"),
			Result: mustMarshalJSON(t, want),
		}
	})
	defer done()

	dbs, err := c.ListDatabases(context.Background())
	if err != nil {
		t.Fatalf("failed to list databases: %v", err)
	}

	if diff := cmp.Diff(want, dbs); diff != "" {
		t.Fatalf("unexpected databases (-want +got):\n%s", diff)
	}
}

func TestClientEchoError(t *testing.T) {
	c, _, done := testClient(t, func(req jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID:     strPtr("1"),
			Result: mustMarshalJSON(t, []string{"foo"}),
		}
	})
	defer done()

	if err := c.Echo(context.Background()); err == nil {
		t.Fatal("expected an error, but none occurred")
	}
}

func TestClientEchoOK(t *testing.T) {
	const echo = "github.com/digitalocean/go-openvswitch/ovsdb"

	c, _, done := testClient(t, func(req jsonrpc.Request) jsonrpc.Response {
		if diff := cmp.Diff("echo", req.Method); diff != "" {
			panicf("unexpected RPC method (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff([]interface{}{echo}, req.Params); diff != "" {
			panicf("unexpected RPC parameters (-want +got):\n%s", diff)
		}

		return jsonrpc.Response{
			ID:     strPtr("1"),
			Result: mustMarshalJSON(t, []string{echo}),
		}
	})
	defer done()

	if err := c.Echo(context.Background()); err != nil {
		t.Fatalf("failed to echo: %v", err)
	}
}

func TestClientTransactSelect(t *testing.T) {
	const db = "Open_vSwitch"

	c, _, done := testClient(t, func(req jsonrpc.Request) jsonrpc.Response {
		if diff := cmp.Diff("transact", req.Method); diff != "" {
			panicf("unexpected RPC method (-want +got):\n%s", diff)
		}

		// TODO(mdlayher): clean up with JSON unmarshaler implementations.
		params := []interface{}{
			db,
			map[string]interface{}{
				"op":    "select",
				"table": "Bridge",
				"where": []interface{}{
					[]interface{}{"name", "==", "ovsbr0"},
				},
			},
		}

		if diff := cmp.Diff(params, req.Params); diff != "" {
			panicf("unexpected RPC parameters (-want +got):\n%s", diff)
		}

		type result struct {
			Rows []ovsdb.Row `json:"rows"`
		}

		return jsonrpc.Response{
			ID: strPtr("1"),
			Result: mustMarshalJSON(t, []result{{
				Rows: []ovsdb.Row{{
					"name": "ovsbr0",
				}},
			}}),
		}
	})
	defer done()

	ops := []ovsdb.TransactOp{ovsdb.Select{
		Table: "Bridge",
		Where: []ovsdb.Cond{
			ovsdb.Equal("name", "ovsbr0"),
		},
	}}

	rows, err := c.Transact(context.Background(), db, ops)
	if err != nil {
		t.Fatalf("failed to perform transaction: %v", err)
	}

	want := []ovsdb.Row{{
		"name": "ovsbr0",
	}}

	if diff := cmp.Diff(want, rows); diff != "" {
		t.Fatalf("unexpected rows (-want +got):\n%s", diff)
	}
}
