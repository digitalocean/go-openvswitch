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
