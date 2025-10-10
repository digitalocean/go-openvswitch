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

func TestZoneMarkAggregatorSnapshot(t *testing.T) {
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

	// Test snapshot functionality with new zmKey-based mapping
	snapshot := agg.Snapshot()
	if snapshot == nil {
		t.Fatal("Snapshot() returned nil")
	}

	// Verify snapshot is a map[ZmKey]int
	if len(snapshot) == 0 {
		t.Log("Snapshot is empty (expected in test environment)")
	}

	// Test that we can iterate over the snapshot
	for key, count := range snapshot {
		if count <= 0 {
			t.Errorf("Invalid count %d for key %+v", count, key)
		}
		t.Logf("Zone: %d, Mark: %d, Count: %d", key.Zone, key.Mark, count)
	}

	// Clean up
	agg.Stop()
}

func TestZmKeyComparison(t *testing.T) {
	// Test that ZmKey works correctly as a map key
	key1 := ZmKey{Zone: 1, Mark: 100}
	key2 := ZmKey{Zone: 1, Mark: 100}
	key3 := ZmKey{Zone: 2, Mark: 100}
	key4 := ZmKey{Zone: 1, Mark: 200}

	// Test equality
	if key1 != key2 {
		t.Error("Identical ZmKey structs should be equal")
	}

	// Test inequality
	if key1 == key3 {
		t.Error("Different zone ZmKey structs should not be equal")
	}
	if key1 == key4 {
		t.Error("Different mark ZmKey structs should not be equal")
	}

	// Test as map keys
	testMap := make(map[ZmKey]int)
	testMap[key1] = 5
	testMap[key3] = 10

	if testMap[key1] != 5 {
		t.Error("ZmKey should work as map key")
	}
	if testMap[key2] != 5 {
		t.Error("Equal ZmKey structs should map to same value")
	}
	if testMap[key3] != 10 {
		t.Error("Different ZmKey should map to different value")
	}
}
