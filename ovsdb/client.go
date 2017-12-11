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
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/digitalocean/go-openvswitch/ovsdb/internal/jsonrpc"
)

// A Client is an OVSDB client.  Clients can be customized by using OptionFuncs
// in the Dial and New functions.
//
// All methods on the Client that accept a context.Context can use the context
// to cancel or time out requests.  Some methods may use the context for advanced
// use cases.  If this is the case, the documentation for the method will explain
// these use cases.
type Client struct {
	// NB: must 64-bit align these atomic integers, so they should appear first
	// in the Client structure.
	// See: https://golang.org/pkg/sync/atomic/#pkg-note-BUG

	// Incremented atomically when sending RPCs.
	rpcID int64

	// Statistics about the echo loop.
	echoOK, echoFail int64

	// All other types should occur after atomic integers.

	// The RPC connection, and its logger.
	c  *jsonrpc.Conn
	ll *log.Logger

	// Callbacks for RPC responses.
	cbMu      sync.RWMutex
	callbacks map[string]callback

	// Interval at which echo RPCs should occur in the background.
	echoInterval time.Duration

	// Track and clean up background goroutines.
	cancel func()
	wg     *sync.WaitGroup
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

// EchoInterval specifies an interval at which the Client will send
// echo RPCs to an OVSDB server to keep the connection alive.  Note that the
// OVSDB server may also send its own echo RPCs to the Client, and the Client
// will always reply to those on behalf of the user.
//
// If this option is not used, the Client will only send echo RPCs when the
// server sends an echo RPC to it.
//
// Specify a duration of 0 to disable sending background echo RPCs at a
// fixed interval.
func EchoInterval(d time.Duration) OptionFunc {
	return func(c *Client) error {
		c.echoInterval = d
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

	// Set up the JSON-RPC connection.
	client.c = jsonrpc.NewConn(conn, client.ll)

	// Set up callbacks.
	client.callbacks = make(map[string]callback)

	// Coordinates the sending of echo messages among multiple goroutines.
	echoC := make(chan struct{})

	// Start up any background routines, and enable canceling them via context.
	ctx, cancel := context.WithCancel(context.Background())
	client.cancel = cancel

	var wg sync.WaitGroup
	wg.Add(2)

	// If configured, trigger echo RPCs in the background at a fixed interval.
	if d := client.echoInterval; d != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.echoTicker(ctx, d, echoC)
		}()
	}

	// Send echo RPCs when triggered by channel.
	go func() {
		defer wg.Done()
		client.echoLoop(ctx, echoC)
	}()

	// Handle all incoming RPC responses and notifications.
	go func() {
		defer wg.Done()
		client.listen(ctx, echoC)
	}()

	client.wg = &wg

	return client, nil
}

// requestID returns the next available request ID for an RPC.
func (c *Client) requestID() string {
	// We use integer IDs by convention, but OVSDB happily accepts
	// any non-null JSON value.
	return strconv.FormatInt(atomic.AddInt64(&c.rpcID, 1), 10)
}

// Close closes a Client's connection and cleans up its resources.
func (c *Client) Close() error {
	c.cancel()
	err := c.c.Close()
	c.wg.Wait()
	return err
}

// Stats returns a ClientStats with current statistics for the Client.
func (c *Client) Stats() ClientStats {
	var s ClientStats

	c.cbMu.RLock()
	s.Callbacks.Current = len(c.callbacks)
	c.cbMu.RUnlock()

	s.EchoLoop.Success = int(atomic.LoadInt64(&c.echoOK))
	s.EchoLoop.Failure = int(atomic.LoadInt64(&c.echoFail))

	return s
}

// ClientStats contains statistics about a Client.
type ClientStats struct {
	// Statistics about the Client's internal callbacks.
	Callbacks struct {
		// The number of callback hooks currently registered and waiting
		// for RPC responses.
		Current int
	}

	// Statistics about the Client's internal echo RPC loop.
	EchoLoop struct {
		// The number of successful and failed echo RPCs in the loop.
		Success, Failure int
	}
}

// rpc performs a single RPC request, and checks the response for errors.
func (c *Client) rpc(ctx context.Context, method string, out, arg interface{}) error {
	// Was the context canceled before sending the RPC?
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Unmarshal results into empty struct if no out specified.
	if out == nil {
		out = &struct{}{}
	}

	// Captures any OVSDB errors.
	r := result{Reply: out}

	req := jsonrpc.Request{
		Method: method,
		Params: arg,
		ID:     c.requestID(),
	}

	// Add callback for this RPC ID to return results via channel.
	ch := make(chan rpcResponse, 1)
	c.addCallback(req.ID, callback{
		Ctx:      ctx,
		Response: ch,
	})

	// Ensure that the callback is always cleaned up on return from this function.
	// Note that this will result in the callback being deleted twice if the RPC
	// returns successfully, but that's okay; it's a no-op.
	//
	// TODO(mdlayher): a more robust solution around callback map modifications.
	defer func() {
		c.cbMu.Lock()
		defer c.cbMu.Unlock()

		delete(c.callbacks, req.ID)
	}()

	if err := c.c.Send(req); err != nil {
		return err
	}

	// Await RPC completion or cancelation.
	select {
	case <-ctx.Done():
		// RPC canceled.  The callback is cleaned up by deferred function in
		// case no message ever arrives with its request ID, so we don't leak
		// callbacks.
		return ctx.Err()
	case res, ok := <-ch:
		if !ok {
			// Channel was closed by producer after a context cancelation,
			// and woke up this consumer.  The select statement happened
			// to pick this case even though the context was canceled.
			return ctx.Err()
		}

		// RPC complete.
		return rpcResult(res, &r)
	}
}

// listen starts an RPC receive loop that can return RPC results to
// clients via a callback.
func (c *Client) listen(ctx context.Context, echoC chan<- struct{}) {
	for {
		res, err := c.c.Receive()
		if err != nil {
			// EOF or closed connection means time to stop serving.
			if err == io.EOF || isClosedNetwork(err) {
				return
			}

			// For any other connection errors, just keep trying.
			continue
		}

		// Handle any JSON-RPC notifications.
		// TODO(mdlayher): deal with other RPC notifications.
		switch res.Method {
		case "echo":
			// OVSDB server wants us to send an echo to it, but will also send
			// us a response to that echo.  Since this goroutine is the one that
			// needs to receive that response and issue the callback for it, we
			// ask the echo loop goroutine to send an echo on our behalf.
			select {
			case <-ctx.Done():
			case echoC <- struct{}{}:
			}
			continue
		}

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

// echoTicker starts a loop that triggers echo RPCs via channel at a fixed
// time interval.
func (c *Client) echoTicker(ctx context.Context, d time.Duration, echoC chan<- struct{}) {
	t := time.NewTicker(d)
	defer t.Stop()

	for {
		// If context is canceled, we should exit this loop.  If a request is fired
		// and the context was already canceled, we exit there as well.
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := ctx.Err(); err != nil {
				return
			}
		}

		// Allow producer to stop if context closed instead of blocking if
		// the consumer is stopped.
		select {
		case <-ctx.Done():
			return
		case echoC <- struct{}{}:
		}
	}
}

// echoLoop starts a loop that sends echo RPCs when requested via channel.
func (c *Client) echoLoop(ctx context.Context, echoC <-chan struct{}) {
	for {
		// If context is canceled, we should exit this loop.  If a request is fired
		// and the context was already canceled, we exit there as well.
		select {
		case <-ctx.Done():
			return
		case <-echoC:
			if err := ctx.Err(); err != nil {
				return
			}
		}

		// For the time being, we will track metrics about the number of successes
		// and failures while sending echo RPCs.
		// TODO(mdlayher): improve error handling for echo loop.
		if err := c.Echo(ctx); err != nil {
			if isClosedNetwork(err) {
				// Our socket was closed, which means the context should be canceled
				// and we should terminate on the next loop.  No need to increment
				// errors counter.
				continue
			}

			// Count other errors as failures.
			atomic.AddInt64(&c.echoFail, 1)
			continue
		}

		atomic.AddInt64(&c.echoOK, 1)
	}
}

// A callback can be used to send a message back to a caller, or
// allow the caller to cancel waiting for a message.
type callback struct {
	Ctx      context.Context
	Response chan rpcResponse
}

// addCallback registers a callback for an RPC response for the specified ID.
func (c *Client) addCallback(id string, cb callback) {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()

	if _, ok := c.callbacks[id]; ok {
		// This ID was already registered.
		panicf("OVSDB callback with ID %q already registered", id)
	}

	c.callbacks[id] = cb
}

// doCallback performs a callback for an RPC response and clears the
// callback on completion.
func (c *Client) doCallback(id string, res rpcResponse) {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()

	cb, ok := c.callbacks[id]
	if !ok {
		// Nobody is listening to this callback.
		return
	}

	// Producer can safely close channel on return.
	defer close(cb.Response)

	// Wait for send or cancelation.
	select {
	case <-cb.Ctx.Done():
		// Request's context was canceled.
	case cb.Response <- res:
		// Message was successfully sent.
	}

	delete(c.callbacks, id)
}

// isClosedNetwork checks for errors caused by a closed network connection.
func isClosedNetwork(err error) bool {
	if err == nil {
		return false
	}

	// Not an awesome solution, but see: https://github.com/golang/go/issues/4373.
	return strings.Contains(err.Error(), "use of closed network connection")
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
