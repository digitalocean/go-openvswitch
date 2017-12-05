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

package jsonrpc_test

import (
	"fmt"
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsdb/internal/jsonrpc"
	"github.com/google/go-cmp/cmp"
)

func TestConnExecuteBadSequence(t *testing.T) {
	req := jsonrpc.Request{
		ID:     10,
		Method: "test",
	}

	c, done := jsonrpc.TestConn(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			// Bad sequence.
			ID: 1,
		}
	})
	defer done()

	if err := c.Execute(req, nil); err == nil {
		t.Fatal("expected an error, but none occurred")
	}
}

func TestConnExecuteError(t *testing.T) {
	// TODO(mdlayher): what does this actually look like?
	type rpcError struct {
		Details string
	}

	c, done := jsonrpc.TestConn(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID: 10,
			Error: rpcError{
				Details: "some error",
			},
		}
	})
	defer done()

	if err := c.Execute(jsonrpc.Request{ID: 10}, nil); err == nil {
		t.Fatal("expected an error, but none occurred")
	}
}

func TestConnExecuteOK(t *testing.T) {
	req := jsonrpc.Request{
		Method: "hello",
		Params: []interface{}{"world"},
	}

	type message struct {
		Message string `json:"message"`
	}

	want := message{
		Message: "hello world",
	}

	c, done := jsonrpc.TestConn(t, func(got jsonrpc.Request) jsonrpc.Response {
		req.ID = 1

		if diff := cmp.Diff(req, got); diff != "" {
			panicf("unexpected request (-want +got):\n%s", diff)
		}

		return jsonrpc.Response{
			ID:     1,
			Result: want,
		}
	})
	defer done()

	var out message
	if err := c.Execute(req, &out); err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if diff := cmp.Diff(want, out); diff != "" {
		t.Fatalf("unexpected response (-want +got):\n%s", diff)
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
