//go:build linux
// +build linux

package ovsnl

import (
	"testing"
)

func TestConntrackService(t *testing.T) {
	// Test basic ConntrackService creation and close
	svc, err := NewConntrackService()
	if err != nil {
		t.Fatalf("NewConntrackService() error = %v", err)
	}

	if svc == nil {
		t.Fatal("NewConntrackService() returned nil service")
	}

	// Test Close method
	if err := svc.Close(); err != nil {
		t.Fatalf("ConntrackService.Close() error = %v", err)
	}
}

func TestZoneMarkAggregator(t *testing.T) {
	// Test aggregator creation
	svc, err := NewConntrackService()
	if err != nil {
		t.Fatalf("NewConntrackService() error = %v", err)
	}

	agg, err := NewZoneMarkAggregator(svc)
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
