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
)

// A result is used to unmarshal JSON-RPC results, and to check for any errors.
type result struct {
	Reply interface{}
	Err   *Error
}

// An rpcResponse is a response used in RPC callbacks.
type rpcResponse struct {
	Result json.RawMessage
	Error  error
}

// rpcResult handles any errors from an rpcResponse and unmarshals results into
// a result.
func rpcResult(res rpcResponse, r *result) error {
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
