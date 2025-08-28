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

// ConntrackService manages the connection to the kernel's conntrack via Netlink.
type ConntrackService struct {
	// No client on non-Linux platforms
}

// ZoneMarkAggregator keeps live counts (zone -> mark -> count).
type ZoneMarkAggregator struct {
	// No implementation on non-Linux platforms
}

// NewConntrackService creates a new ConntrackService.
// On non-Linux platforms, this returns an error indicating the feature is not supported.
func NewConntrackService() (*ConntrackService, error) {
	return nil, fmt.Errorf("conntrack service is only available on Linux systems")
}

// NewZoneMarkAggregator creates a new aggregator on top of an existing ConntrackService.
// On non-Linux platforms, this returns an error.
func NewZoneMarkAggregator(s *ConntrackService) (*ZoneMarkAggregator, error) {
	return nil, fmt.Errorf("conntrack aggregator is only available on Linux systems")
}

// Close closes the underlying Netlink connection for conntrack.
func (s *ConntrackService) Close() error {
	return nil
}

// GetStats returns performance counters from the conntrack subsystem.
// On non-Linux platforms, this returns an error.
func (s *ConntrackService) GetStats() (*ConntrackPerformanceStats, error) {
	return nil, fmt.Errorf("conntrack stats are only available on Linux systems")
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
func (a *ZoneMarkAggregator) Snapshot() map[uint16]map[uint32]int {
	return make(map[uint16]map[uint32]int)
}

// PrimeSnapshot tries a guarded one-shot dump to seed counts for long-lived flows.
// On non-Linux platforms, this returns an error.
func (a *ZoneMarkAggregator) PrimeSnapshot(ctx context.Context, maxEntries int) error {
	return fmt.Errorf("conntrack prime snapshot is only available on Linux systems")
}
