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

//go:build !linux
// +build !linux

package ovsnl

import (
	"context"
	"fmt"
	"net"
)

// ConntrackEntry represents a single connection tracking entry from the kernel.
type ConntrackEntry struct {
	Protocol   string // "tcp", "udp", "icmp" etc.
	OrigSrc    net.IP
	OrigDst    net.IP
	OrigSPort  uint16
	OrigDPort  uint16
	ReplySrc   net.IP
	ReplyDst   net.IP
	ReplySPort uint16
	ReplyDPort uint16
	Zone       uint16
	Mark       uint32
	State      string
}

// ZoneStats holds statistics for a zone
type ZoneStats struct {
	TotalCount int
	Entries    []ConntrackEntry // Only populated if TotalCount > threshold
}

// ConntrackPerformanceStats represents aggregated performance counters from all CPUs
type ConntrackPerformanceStats struct {
	TotalFound         uint32
	TotalInvalid       uint32
	TotalIgnore        uint32
	TotalInsert        uint32
	TotalInsertFailed  uint32
	TotalDrop          uint32
	TotalEarlyDrop     uint32
	TotalError         uint32
	TotalSearchRestart uint32
	CPUs               int
}

// zmKey is a compact key for (zone,mark)
type zmKey struct {
	Zone uint16
	Mark uint32
}

// ZoneMarkAggregator keeps live counts (zone -> mark -> count).
type ZoneMarkAggregator struct {
	// No implementation on non-Linux platforms
}

// NewZoneMarkAggregator creates a new aggregator.
// On non-Linux platforms, this returns an error.
func NewZoneMarkAggregator() (*ZoneMarkAggregator, error) {
	return nil, fmt.Errorf("conntrack aggregator is only available on Linux systems")
}

// Start subscribes to conntrack events and maintains counts.
// On non-Linux platforms, this returns an error.
func (a *ZoneMarkAggregator) Start() error {
	return fmt.Errorf("conntrack aggregator is only available on Linux systems")
}

// Stop cancels listening.
func (a *ZoneMarkAggregator) Stop() {
	// No-op on non-Linux platforms
}

// Snapshot returns a safe copy of counts.
// On non-Linux platforms, this returns an empty map.
func (a *ZoneMarkAggregator) Snapshot() map[zmKey]int {
	return make(map[zmKey]int)
}

// PrimeSnapshot tries a guarded one-shot dump to seed counts for long-lived flows.
// On non-Linux platforms, this returns an error.
func (a *ZoneMarkAggregator) PrimeSnapshot(ctx context.Context, maxEntries int) error {
	return fmt.Errorf("conntrack prime snapshot is only available on Linux systems")
}
