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

package jsonrpc

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
)

// A TestFunc is used to create RPC responses in TestConn.
type TestFunc func(req Request) Response

// TestConn creates a Conn backed by a server that calls a TestFunc.
// Invoke the returned closure to clean up its resources.
func TestConn(t *testing.T, fn TestFunc) (*Conn, func()) {
	t.Helper()

	conn, done := TestNetConn(t, fn)

	c := NewConn(conn, nil)

	return c, func() {
		_ = c.Close()
		done()
	}
}

// TestNetConn creates a net.Conn backed by a server that calls a TestFunc.
// Invoke the returned closure to clean up its resources.
func TestNetConn(t *testing.T, fn TestFunc) (net.Conn, func()) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			c, err := l.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network") {
					return
				}

				panicf("failed to accept: %v", err)
			}

			var req Request
			if err := json.NewDecoder(c).Decode(&req); err != nil {
				panicf("failed to decode request: %v", err)
			}

			res := fn(req)
			if err := json.NewEncoder(c).Encode(res); err != nil {
				panicf("failed to encode response: %v", err)
			}
			_ = c.Close()
		}
	}()

	c, err := net.Dial("tcp", l.Addr().String())
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

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
