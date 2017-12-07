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
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsdb/internal/jsonrpc"
	"github.com/google/go-cmp/cmp"
)

func TestConnSendNoRequestID(t *testing.T) {
	c, _, done := jsonrpc.TestConn(t, nil)
	defer done()

	if err := c.Send(jsonrpc.Request{}); err == nil {
		t.Fatal("expected an error, but none occurred")
	}
}

func TestConnReceiveEOF(t *testing.T) {
	c := jsonrpc.NewConn(&eofReadWriteCloser{}, nil)

	// Conn should not mask io.EOF.
	_, err := c.Receive()
	if err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConnSendReceiveError(t *testing.T) {
	// TODO(mdlayher): what does this actually look like?
	type rpcError struct {
		Details string
	}

	c, _, done := jsonrpc.TestConn(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID: strPtr("10"),
			Error: rpcError{
				Details: "some error",
			},
		}
	})
	defer done()

	if err := c.Send(jsonrpc.Request{ID: "10"}); err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	res, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to receive response: %v", err)
	}

	if err := res.Err(); err == nil {
		t.Fatal("expected an error, but none occurred")
	}
}

func TestConnSendReceiveOK(t *testing.T) {
	req := jsonrpc.Request{
		Method: "hello",
		Params: []interface{}{"world"},
		ID:     "1",
	}

	type message struct {
		Message string `json:"message"`
	}

	want := message{
		Message: "hello world",
	}

	c, _, done := jsonrpc.TestConn(t, func(got jsonrpc.Request) jsonrpc.Response {
		if diff := cmp.Diff(req, got); diff != "" {
			panicf("unexpected request (-want +got):\n%s", diff)
		}

		return jsonrpc.Response{
			ID:     strPtr("1"),
			Result: mustMarshalJSON(t, want),
		}
	})
	defer done()

	if err := c.Send(req); err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	res, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to receive response: %v", err)
	}

	if err := res.Err(); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var out message
	if err := json.Unmarshal(res.Result, &out); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if diff := cmp.Diff(want, out); diff != "" {
		t.Fatalf("unexpected response (-want +got):\n%s", diff)
	}
}

func TestConnSendReceiveNotificationsOK(t *testing.T) {
	const id = "10"

	req := jsonrpc.Request{
		ID:     id,
		Method: "monitor",
		Params: []interface{}{"Open_vSwitch"},
	}

	res := jsonrpc.Response{
		ID:     strPtr(id),
		Result: mustMarshalJSON(t, "some bytes"),
	}

	c, notifC, done := jsonrpc.TestConn(t, func(got jsonrpc.Request) jsonrpc.Response {
		if diff := cmp.Diff(req, got); diff != "" {
			panicf("unexpected request (-want +got):\n%s", diff)
		}

		return res
	})
	defer done()

	note := &jsonrpc.Response{
		Method: "notify",
	}
	notifC <- note
	notifC <- note

	if err := c.Send(req); err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	var responses, notes int
	for i := 0; i < 3; i++ {
		res, err := c.Receive()
		if err != nil {
			t.Fatalf("failed to receive response: %v", err)
		}

		if res.ID != nil {
			responses++
			if diff := cmp.Diff(req.ID, *res.ID); diff != "" {
				t.Fatalf("unexpected response request ID (-want +got):\n%s", diff)
			}

			continue
		}

		notes++
		if diff := cmp.Diff(note.Method, res.Method); diff != "" {
			t.Fatalf("unexpected notification method (-want +got):\n%s", diff)
		}
	}

	if diff := cmp.Diff(1, responses); diff != "" {
		t.Fatalf("unexpected number of responses (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(2, notes); diff != "" {
		t.Fatalf("unexpected number of notifications (-want +got):\n%s", diff)
	}
}

func mustMarshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	return b
}

func strPtr(s string) *string {
	return &s
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

type eofReadWriteCloser struct {
	io.ReadWriteCloser
}

func (rwc *eofReadWriteCloser) Read(b []byte) (int, error) {
	return 0, io.EOF
}
