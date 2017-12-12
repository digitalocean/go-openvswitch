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

import "encoding/json"

// A Cond is a conditional expression which is evaluated by the OVSDB server
// in a transaction.
type Cond struct {
	Column, Function, Value string
}

// TODO(mdlayher): more helper functions?  Cond as an interface?

// Equal creates a Cond that ensures a column's value equals the
// specified value.
func Equal(column, value string) Cond {
	return Cond{
		Column:   column,
		Function: "==",
		Value:    value,
	}
}

// MarshalJSON implements json.Marshaler.
func (c Cond) MarshalJSON() ([]byte, error) {
	// Conditionals are expected in three element arrays.
	return json.Marshal([3]string{
		c.Column,
		c.Function,
		c.Value,
	})
}

// A TransactOp is an operation that can be applied with Client.Transact.
type TransactOp interface {
	json.Marshaler
}

var _ TransactOp = Select{}

// Select is a TransactOp which fetches information from a database.
type Select struct {
	// The name of the table to select from.
	Table string

	// Zero or more Conds for conditional select.
	Where []Cond

	// TODO(mdlayher): specify columns.
}

// MarshalJSON implements json.Marshaler.
func (s Select) MarshalJSON() ([]byte, error) {
	// Send an empty array instead of nil if no where clause.
	where := s.Where
	if where == nil {
		where = []Cond{}
	}

	sel := struct {
		Op    string `json:"op"`
		Table string `json:"table"`
		Where []Cond `json:"where"`
	}{
		Op:    "select",
		Table: s.Table,
		Where: where,
	}

	return json.Marshal(sel)
}

// A transactArg is used to properly JSON marshal the arguments for a
// transact RPC.
type transactArg struct {
	Database string
	Ops      []TransactOp
}

// MarshalJSON implements json.Marshaler.
func (t transactArg) MarshalJSON() ([]byte, error) {
	out := []interface{}{
		t.Database,
	}

	for _, op := range t.Ops {
		out = append(out, op)
	}

	return json.Marshal(out)
}
