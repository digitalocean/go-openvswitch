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
	"sync"
)

// A Request is a JSON-RPC request.
type Request struct {
	ID     int           `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// A Response is a JSON-RPC response.
type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
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
		enc:    json.NewEncoder(rwc),
		dec:    json.NewDecoder(rwc),
		closer: rwc,
	}
}

// A Conn is a JSON-RPC connection.
type Conn struct {
	mu     sync.RWMutex
	enc    *json.Encoder
	dec    *json.Decoder
	closer io.Closer
	seq    int
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.closer.Close()
}

// Execute executes a single request and unmarshals its results into out.
func (c *Conn) Execute(req Request, out interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++

	// Use auto-increment sequence, or user-defined if requested.
	seq := c.seq
	if req.ID != 0 {
		seq = req.ID
	} else {
		req.ID = seq
	}

	// Non-nil array required for ovsdb-server to reply.
	if req.Params == nil {
		req.Params = []interface{}{}
	}

	if err := c.enc.Encode(req); err != nil {
		return fmt.Errorf("failed to encode JSON-RPC request: %v", err)
	}

	res := Response{
		Result: out,
	}

	if err := c.dec.Decode(&res); err != nil {
		return fmt.Errorf("failed to decode JSON-RPC request: %v", err)
	}

	if res.ID != seq {
		return fmt.Errorf("bad JSON-RPC sequence: %d, want: %d", res.ID, seq)
	}

	// TODO(mdlayher): better errors.
	if res.Error != nil {
		return fmt.Errorf("received JSON-RPC error: %#v", res.Error)
	}

	return nil
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
