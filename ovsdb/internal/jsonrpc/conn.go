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
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
)

// A Request is a JSON-RPC request.
type Request struct {
	ID     string      `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// A Response is either a JSON-RPC response, or a JSON-RPC request notification.
type Response struct {
	// Non-null for response; null for request notification.
	ID *string `json:"id"`

	// Response fields.
	Result json.RawMessage `json:"result,omitempty"`
	Error  interface{}     `json:"error"`

	// Request notification fields.
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Err returns an error, if one occurred in a Response.
func (r *Response) Err() error {
	// TODO(mdlayher): better errors.
	if r.Error == nil {
		return nil
	}

	return fmt.Errorf("received JSON-RPC error: %#v", r.Error)
}

// NewConn creates a new Conn with the input io.ReadWriteCloser.
// If a logger is specified, it is used for debug logs.
func NewConn(rwc io.ReadWriteCloser, ll *log.Logger) *Conn {
	if ll != nil {
		rwc = &debugReadWriteCloser{
			rwc: rwc,
			ll:  ll,
		}
	}

	return &Conn{
		c:   rwc,
		enc: json.NewEncoder(rwc),
		dec: json.NewDecoder(rwc),
	}
}

// A Conn is a JSON-RPC connection.
type Conn struct {
	c io.Closer

	encMu sync.Mutex
	enc   *json.Encoder

	decMu sync.Mutex
	dec   *json.Decoder
}

// Close closes the connection.
func (c *Conn) Close() error {
	// TODO(mdlayher): acquiring mutex will block forever if receive loop
	// is happening elsewhere. Any way to avoid this?
	return c.c.Close()
}

// Send sends a single JSON-RPC request.
func (c *Conn) Send(req Request) error {
	if req.ID == "" {
		return errors.New("JSON-RPC request ID must not be empty")
	}

	// Non-nil array required for ovsdb-server to reply.
	if req.Params == nil {
		req.Params = []interface{}{}
	}

	c.encMu.Lock()
	defer c.encMu.Unlock()

	if err := c.enc.Encode(req); err != nil {
		return fmt.Errorf("failed to encode JSON-RPC request: %v", err)
	}

	return nil
}

// Receive receives a single JSON-RPC response.
func (c *Conn) Receive() (*Response, error) {
	c.decMu.Lock()
	defer c.decMu.Unlock()

	var res Response
	if err := c.dec.Decode(&res); err != nil {
		// Don't mask EOF errors with added detail.
		if err == io.EOF {
			return nil, err
		}

		return nil, fmt.Errorf("failed to decode JSON-RPC response: %v", err)
	}

	return &res, nil
}

type debugReadWriteCloser struct {
	rwc io.ReadWriteCloser
	ll  *log.Logger
}

func (rwc *debugReadWriteCloser) Read(b []byte) (int, error) {
	n, err := rwc.rwc.Read(b)
	if err != nil {
		return n, err
	}

	rwc.ll.Printf(" read: %s", string(b[:n]))
	return n, nil
}

func (rwc *debugReadWriteCloser) Write(b []byte) (int, error) {
	n, err := rwc.rwc.Write(b)
	if err != nil {
		return n, err
	}

	rwc.ll.Printf("write: %s", string(b[:n]))
	return n, nil
}

func (rwc *debugReadWriteCloser) Close() error {
	err := rwc.rwc.Close()
	rwc.ll.Println("close:", err)
	return err
}
