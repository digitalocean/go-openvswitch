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
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/digitalocean/go-openvswitch/ovsdb/internal/jsonrpc"
)

// A Client is an OVSDB client.
type Client struct {
	c  *jsonrpc.Conn
	ll *log.Logger
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

	client.c = jsonrpc.NewConn(conn, client.ll)

	return client, nil
}

// Close closes a Client's connection.
func (c *Client) Close() error {
	return c.c.Close()
}

// ListDatabases returns the name of all databases known to the OVSDB server.
func (c *Client) ListDatabases() ([]string, error) {
	var dbs []string
	if err := c.rpc("list_dbs", &dbs); err != nil {
		return nil, err
	}

	return dbs, nil
}

// rpc performs a single RPC request, and checks the response for errors.
func (c *Client) rpc(method string, out interface{}, args ...interface{}) error {
	// Captures any OVSDB errors.
	r := result{
		Reply: out,
	}

	req := jsonrpc.Request{
		Method: method,
		Params: args,
		// Let the client handle the request ID.
	}

	if err := c.c.Execute(req, &r); err != nil {
		return err
	}

	// OVSDB server returned an error, return it.
	if r.Err != nil {
		return r.Err
	}

	return nil
}

// A result is used to unmarshal JSON-RPC results, and to check for any errors.
type result struct {
	Reply interface{}
	Err   *Error
}

// errPrefix is a prefix that occurs if an error is present in a JSON-RPC response.
var errPrefix = []byte(`{"error":`)

func (r *result) UnmarshalJSON(b []byte) error {
	// No error? Return the result.
	if !bytes.HasPrefix(b, errPrefix) {
		return json.Unmarshal(b, r.Reply)
	}

	// Found an error, unmarshal and return it later.
	var e Error
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}

	r.Err = &e
	return nil
}

var _ error = &Error{}

// An Error is an error returned by an OVSDB server.  Its fields can be
// used to determine the cause of an error.
type Error struct {
	Err     string `json:"error"`
	Details string `json:"details"`
	Syntax  string `json:"syntax"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.Err, e.Details, e.Syntax)
}
