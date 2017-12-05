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
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
)

// A TestFunc is used to create RPC responses in TestConn.
type TestFunc func(req Request) Response

// TestConn creates a Conn backed by a server that calls a TestFunc.
// Notifications can be pushed to the client using the channel.
// Invoke the returned closure to clean up its resources.
func TestConn(t *testing.T, fn TestFunc) (*Conn, chan<- *Response, func()) {
	t.Helper()

	conn, notifC, done := TestNetConn(t, fn)

	c := NewConn(conn, log.New(os.Stderr, "", 0))

	return c, notifC, func() {
		_ = c.Close()
		done()
	}
}

// TestNetConn creates a net.Conn backed by a server that calls a TestFunc.
// Notifications can be pushed to the client using the channel.
// Invoke the returned closure to clean up its resources.
func TestNetConn(t *testing.T, fn TestFunc) (net.Conn, chan<- *Response, func()) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	notifC := make(chan *Response, 16)

	go func() {
		defer wg.Done()

		// Accept a single connection.
		c, err := l.Accept()
		if err != nil {
			if isNetworkCloseError(err) {
				return
			}

			panicf("failed to accept: %v", err)
		}
		defer c.Close()

		dec := json.NewDecoder(c)

		var encMu sync.RWMutex
		enc := json.NewEncoder(c)

		// Push RPC notifications to the client.
		var notifWG sync.WaitGroup
		notifWG.Add(1)
		defer notifWG.Wait()

		go func() {
			defer notifWG.Done()

			for n := range notifC {
				encMu.Lock()
				err := enc.Encode(n)
				encMu.Unlock()

				if err != nil {
					if isNetworkCloseError(err) {
						return
					}

					panicf("failed to encode notification: %v", err)
				}
			}
		}()

		// Handle RPC requests and responses to and from the client.
		for {
			var req Request
			if err := dec.Decode(&req); err != nil {
				if isNetworkCloseError(err) {
					return
				}

				panicf("failed to decode request: %#v", err)
			}

			res := fn(req)

			encMu.Lock()
			err := enc.Encode(res)
			encMu.Unlock()

			if err != nil {
				if isNetworkCloseError(err) {
					return
				}

				panicf("failed to encode response: %v", err)
			}
		}
	}()

	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	return c, notifC, func() {
		// Ensure types are cleaned up, and ensure goroutine stops.
		_ = l.Close()
		_ = c.Close()
		close(notifC)
		wg.Wait()
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

func isNetworkCloseError(err error) bool {
	return err == io.EOF ||
		strings.Contains(err.Error(), "use of closed network") ||
		strings.Contains(err.Error(), "connection reset by peer")
}
