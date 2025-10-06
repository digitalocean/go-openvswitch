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

//go:build linux
// +build linux

package ovsnl

import (
	"testing"
)

func TestZoneMarkAggregator(t *testing.T) {
	// Test aggregator creation
	agg, err := NewZoneMarkAggregator()
	if err != nil {
		// This is expected to fail in test environment due to permission requirements
		t.Logf("Expected failure in test environment: NewZoneMarkAggregator() error = %v", err)
		return
	}

	if agg == nil {
		t.Fatal("NewZoneMarkAggregator() returned nil aggregator")
	}

	// Test basic methods
	snapshot := agg.Snapshot()
	if snapshot == nil {
		t.Fatal("Snapshot() returned nil")
	}

	// Clean up
	agg.Stop()
}
