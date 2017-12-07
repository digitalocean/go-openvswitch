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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/digitalocean/go-openvswitch/ovsdb"
	"github.com/digitalocean/go-openvswitch/ovsdb/internal/jsonrpc"
	"github.com/google/go-cmp/cmp"
)

func TestClientJSONRPCError(t *testing.T) {
	const str = "some error"

	c, _, done := testClient(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID:    strPtr("1"),
			Error: str,
		}
	})
	defer done()

	_, err := c.ListDatabases(context.Background())
	if err == nil {
		t.Fatal("expected an error, but none occurred")
	}
}

func TestClientOVSDBError(t *testing.T) {
	const str = "some error"

	c, _, done := testClient(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID: strPtr("1"),
			Result: mustMarshalJSON(t, &ovsdb.Error{
				Err:     str,
				Details: "malformed",
				Syntax:  "{}",
			}),
		}
	})
	defer done()

	_, err := c.ListDatabases(context.Background())
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

func TestClientBadCallback(t *testing.T) {
	c, notifC, done := testClient(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID:     strPtr("1"),
			Result: mustMarshalJSON(t, []string{"foo"}),
		}
	})
	defer done()

	// Client doesn't have a callback for this ID.
	notifC <- &jsonrpc.Response{
		Method: "crash",
		ID:     strPtr("foo"),
	}

	if _, err := c.ListDatabases(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientContextCancelBeforeRPC(t *testing.T) {
	// Context canceled before RPC even begins.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c, _, done := testClient(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID:     strPtr("1"),
			Result: mustMarshalJSON(t, []string{"foo"}),
		}
	})
	defer done()

	_, err := c.ListDatabases(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context canceled error: %v", err)
	}
}

func TestClientContextCancelDuringRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping during short test run")
	}

	// Context canceled during long RPC.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _, done := testClient(t, func(_ jsonrpc.Request) jsonrpc.Response {
		// RPC canceled; RPC server still processing.
		// TODO(mdlayher): try to do something smarter than sleeping in a test.
		cancel()
		<-ctx.Done()

		time.Sleep(500 * time.Millisecond)

		return jsonrpc.Response{
			ID:     strPtr("1"),
			Result: mustMarshalJSON(t, []string{"foo"}),
		}
	})
	defer done()

	_, err := c.ListDatabases(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context canceled error: %v", err)
	}
}

func TestClientLeakCallbacks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping during short test run")
	}

	c, _, done := testClient(t, func(_ jsonrpc.Request) jsonrpc.Response {
		// Only respond with messages that don't match an incoming request.
		return jsonrpc.Response{
			ID:     strPtr("foo"),
			Result: mustMarshalJSON(t, []string{"foo"}),
		}
	})
	defer done()

	// Expect no callbacks registered before RPCs, and none after RPCs time out.
	var want ovsdb.ClientStats
	want.Callbacks.Current = 0

	if diff := cmp.Diff(want, c.Stats()); diff != "" {
		t.Fatalf("unexpected starting client stats (-want +got):\n%s", diff)
	}

	for i := 0; i < 5; i++ {
		// Give enough time for an RPC to be sent so we don't immediately time out.
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := c.ListDatabases(ctx)
		if err != context.DeadlineExceeded {
			t.Fatalf("expected context deadline exceeded error: %v", err)
		}
	}

	if diff := cmp.Diff(want, c.Stats()); diff != "" {
		t.Fatalf("unexpected ending client stats (-want +got):\n%s", diff)
	}
}

func TestClientEchoLoop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping during short test run")
	}

	// Count the number of requests sent to the server.
	echo := ovsdb.EchoInterval(50 * time.Millisecond)
	var reqID int64

	c, _, done := testClient(t, func(req jsonrpc.Request) jsonrpc.Response {
		if diff := cmp.Diff("echo", req.Method); diff != "" {
			panicf("unexpected RPC method (-want +got):\n%s", diff)
		}

		// Keep incrementing the request ID to match the client.
		id := strconv.Itoa(int(atomic.AddInt64(&reqID, 1)))
		return jsonrpc.Response{
			ID:     &id,
			Result: mustMarshalJSON(t, req.Params),
		}
	}, echo)
	defer done()

	// Fail the test if the RPCs don't fire.
	timer := time.AfterFunc(2*time.Second, func() {
		panicf("took too long to wait for echo RPCs")
	})
	defer timer.Stop()

	// Ensure that background echo RPCs are being sent.
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		// Just wait for a handful of RPCs to be sent before success.
		<-tick.C

		stats := c.Stats()

		if n := stats.EchoLoop.Failure; n > 0 {
			t.Fatalf("echo loop RPC failed %d times", n)
		}

		if n := stats.EchoLoop.Success; n > 5 {
			break
		}
	}
}

func TestClientEchoNotification(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping during short test run")
	}

	c, notifC, done := testClient(t, func(req jsonrpc.Request) jsonrpc.Response {
		if diff := cmp.Diff("echo", req.Method); diff != "" {
			panicf("unexpected RPC method (-want +got):\n%s", diff)
		}

		return jsonrpc.Response{
			ID:     &req.ID,
			Result: mustMarshalJSON(t, req.Params),
		}
	})
	defer done()

	// Prompt the client to send an echo in the same way ovsdb-server does.
	notifC <- &jsonrpc.Response{
		ID:     strPtr("echo"),
		Method: "echo",
	}

	// Fail the test if the RPCs don't fire.
	timer := time.AfterFunc(2*time.Second, func() {
		panicf("took too long to wait for echo RPCs")
	})
	defer timer.Stop()

	// Ensure that background echo RPCs are being sent.
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		// Just wait for a single echo RPC cycle before success.
		<-tick.C

		stats := c.Stats()

		if n := stats.EchoLoop.Failure; n > 0 {
			t.Fatalf("echo loop RPC failed %d times", n)
		}

		if n := stats.EchoLoop.Success; n > 0 {
			break
		}
	}
}

func testClient(t *testing.T, fn jsonrpc.TestFunc, options ...ovsdb.OptionFunc) (*ovsdb.Client, chan<- *jsonrpc.Response, func()) {
	t.Helper()

	// Prepend a verbose logger so the caller can override it easily.
	if testing.Verbose() {
		options = append([]ovsdb.OptionFunc{
			ovsdb.Debug(log.New(os.Stderr, "", 0)),
		}, options...)
	}

	conn, notifC, done := jsonrpc.TestNetConn(t, fn)

	c, err := ovsdb.New(conn, options...)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	return c, notifC, func() {
		_ = c.Close()
		done()

		// Make sure that the Client cleaned up appropriately.
		stats := c.Stats()

		if diff := cmp.Diff(0, stats.Callbacks.Current); diff != "" {
			t.Fatalf("unexpected final number of callbacks (-want +got):\n%s", diff)
		}
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
