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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/digitalocean/go-openvswitch/ovsdb/internal/jsonrpc"
)

// A Client is an OVSDB client.
type Client struct {
	c  *jsonrpc.Conn
	ll *log.Logger

	// Incremented atomically when sending RPCs.
	rpcID *int64

	// Callbacks for RPC responses.
	cbMu      sync.RWMutex
	callbacks map[int]chan rpcResponse

	wg *sync.WaitGroup
}

type rpcResponse struct {
	Result json.RawMessage
	Error  error
}

// An OptionFunc is a function which can configure a Client.
type OptionFunc func(c *Client) error

// Debug enables debug logging for a Client.
func Debug(ll *log.Logger) OptionFunc {
	return func(c *Client) error {
		c.ll = ll
		return nil
	}
}

// Dial dials a connection to an OVSDB server and returns a Client.
func Dial(network, addr string, options ...OptionFunc) (*Client, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	return New(conn, options...)
}

// New wraps an existing connection to an OVSDB server and returns a Client.
func New(conn net.Conn, options ...OptionFunc) (*Client, error) {
	client := &Client{}
	for _, o := range options {
		if err := o(client); err != nil {
			return nil, err
		}
	}

	// Set up RPC request IDs.
	var rpcID int64
	client.rpcID = &rpcID

	// Set up the JSON-RPC connection.
	client.c = jsonrpc.NewConn(conn, client.ll)

	// Set up callbacks.
	client.callbacks = make(map[int]chan rpcResponse)

	// Start up any background routines.
	var wg sync.WaitGroup
	wg.Add(1)

	// Handle all incoming RPC responses and notifications.
	go func() {
		defer wg.Done()
		client.listen()
	}()

	client.wg = &wg

	return client, nil
}

// requestID returns the next available request ID for an RPC.
func (c *Client) requestID() int {
	return int(atomic.AddInt64(c.rpcID, 1))
}

// Close closes a Client's connection.
func (c *Client) Close() error {
	err := c.c.Close()
	c.wg.Wait()
	return err
}

// rpc performs a single RPC request, and checks the response for errors.
func (c *Client) rpc(method string, out interface{}, args ...interface{}) error {
	// Unmarshal results into empty struct if no out specified.
	if out == nil {
		out = &struct{}{}
	}

	// Captures any OVSDB errors.
	r := result{
		Reply: out,
	}

	req := jsonrpc.Request{
		Method: method,
		Params: args,
		ID:     c.requestID(),
	}

	// Add callback for this RPC ID to return results via channel.
	ch := make(chan rpcResponse, 0)
	c.addCallback(req.ID, ch)

	if err := c.c.Send(req); err != nil {
		return err
	}

	// Wait for callback to fire.
	res := <-ch
	if err := res.Error; err != nil {
		return err
	}

	if err := json.Unmarshal(res.Result, &r); err != nil {
		return err
	}

	// OVSDB server returned an error, return it.
	if r.Err != nil {
		return r.Err
	}

	return nil
}

// listen starts an RPC receive loop that can return RPC results to
// clients via a callback.
func (c *Client) listen() {
	for {
		res, err := c.c.Receive()
		if err != nil {
			// EOF or closed connection means time to stop serving.
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network") {
				return
			}

			// For any other connection errors, just keep trying.
			continue
		}

		// TODO(mdlayher): deal with RPC notifications.

		// Handle any JSON-RPC top-level errors.
		if err := res.Err(); err != nil {
			c.doCallback(*res.ID, rpcResponse{
				Error: err,
			})
			continue
		}

		// Return RPC results via callback.
		c.doCallback(*res.ID, rpcResponse{
			Result: res.Result,
		})
	}
}

// addCallback registers a callback for an RPC response for the specified ID,
// and accepts a channel to return the results on.
func (c *Client) addCallback(id int, ch chan rpcResponse) {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()

	if _, ok := c.callbacks[id]; ok {
		// This ID was already registered.
		panicf("OVSDB callback with ID %d already registered", id)
	}

	c.callbacks[id] = ch
}

// doCallback performs a callback for an RPC response and clears the
// callback on completion.
func (c *Client) doCallback(id int, res rpcResponse) {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()

	ch, ok := c.callbacks[id]
	if !ok {
		// Nobody is listening to this callback.
		return
	}

	// Return result, clean up channel, and remove this callback.
	ch <- res
	close(ch)
	delete(c.callbacks, id)
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
