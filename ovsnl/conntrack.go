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
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"

	"github.com/ti-mo/conntrack"
	"github.com/ti-mo/netfilter"
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
	// No persistent client - connections created as needed
}

// ZoneMarkAggregator keeps live counts (zone -> mark -> count).
type ZoneMarkAggregator struct {
	mu        sync.RWMutex
	counts    map[uint16]map[uint32]int
	listenCli *conntrack.Conn // Separate connection for listening to events
	stopCh    chan struct{}
	stoppedCh chan struct{}
}

// NewZoneMarkAggregator creates a new aggregator with its own listening connection.
func NewZoneMarkAggregator(s *ConntrackService) (*ZoneMarkAggregator, error) {
	// Create a separate connection for listening to events
	listenCli, err := conntrack.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create listening connection: %w", err)
	}

	return &ZoneMarkAggregator{
		counts:    make(map[uint16]map[uint32]int),
		listenCli: listenCli,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}, nil
}

func NewConntrackService() (*ConntrackService, error) {
	// Don't create a persistent connection - we'll create fresh connections as needed
	// This avoids any interference with the aggregator's multicast connection
	return &ConntrackService{}, nil
}

// Close closes the underlying Netlink connection for conntrack.
func (s *ConntrackService) Close() error {
	// No persistent connection to close
	return nil
}

// GetStats returns performance counters from the conntrack subsystem.
// DISABLED: Stats collection is disabled due to multicast connection issues.
// The exporter now uses nil for getStats to skip this functionality entirely.
/*
func (s *ConntrackService) GetStats() (*ConntrackPerformanceStats, error) {
	// Try the ti-mo/conntrack library first
	statsConn, err := conntrack.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial conntrack for stats: %w", err)
	}
	defer statsConn.Close()

	stats, err := statsConn.Stats()
	if err != nil {
		// If we get the multicast error, return a minimal stats object
		// This allows the exporter to continue functioning
		if strings.Contains(err.Error(), "Conn attached to multicast group") {
			log.Printf("Warning: Conntrack stats collection failed due to multicast issue, returning minimal stats")
			return &ConntrackPerformanceStats{
				CPUs: 1, // Default to 1 CPU
			}, nil
		}
		return nil, fmt.Errorf("failed to get conntrack stats: %w", err)
	}

	aggStats := &ConntrackPerformanceStats{
		CPUs: len(stats),
	}

	for _, stat := range stats {
		aggStats.TotalFound += stat.Found
		aggStats.TotalInvalid += stat.Invalid
		aggStats.TotalIgnore += stat.Ignore
		aggStats.TotalInsert += stat.Insert
		aggStats.TotalInsertFailed += stat.InsertFailed
		aggStats.TotalDrop += stat.Drop
		aggStats.TotalEarlyDrop += stat.EarlyDrop
		aggStats.TotalError += stat.Error
		aggStats.TotalSearchRestart += stat.SearchRestart
	}

	return aggStats, nil
}
*/

// Start subscribes to NEW and DESTROY events and maintains counts.
func (a *ZoneMarkAggregator) Start() error {
	events := make(chan conntrack.Event, 8192)

	// Subscribe to ALL groups (NEW, UPDATE, DESTROY).
	groups := []netfilter.NetlinkGroup{
		netfilter.GroupCTNew,
		netfilter.GroupCTDestroy,
		netfilter.GroupCTUpdate,
	}

	errCh, err := a.listenCli.Listen(events, 2, groups) // 2 workers; tune as needed
	if err != nil {
		return err
	}

	// Watch for errors from workers
	go func() {
		for {
			select {
			case <-a.stopCh:
				close(a.stoppedCh)
				return
			case e := <-errCh:
				if e != nil {
					log.Printf("conntrack listener error: %v", e)
				}
			case ev := <-events:
				a.applyEvent(ev)
			}
		}
	}()

	return nil
}

// Stop cancels listening and closes the connection.
func (a *ZoneMarkAggregator) Stop() {
	close(a.stopCh)
	<-a.stoppedCh
	if a.listenCli != nil {
		a.listenCli.Close()
	}
}

func (a *ZoneMarkAggregator) applyEvent(ev conntrack.Event) {
	f := ev.Flow
	zone := f.Zone
	mark := f.Mark

	// Debug: Log zone values to understand what the library is returning
	if zone != 0 {
		log.Printf("DEBUG: Event zone=%d, mark=%d, type=%d", zone, mark, ev.Type)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	zm, ok := a.counts[zone]
	if !ok {
		zm = make(map[uint32]int)
		a.counts[zone] = zm
	}

	switch {
	case ev.Type&conntrack.EventNew != 0:
		zm[mark]++
	case ev.Type&conntrack.EventDestroy != 0:
		if zm[mark] > 0 {
			zm[mark]--
		}
	case ev.Type&conntrack.EventUpdate != 0:
		// Optional: handle mark change; usually safe to ignore
	}
}

// Snapshot returns a safe copy of counts.
func (a *ZoneMarkAggregator) Snapshot() map[uint16]map[uint32]int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make(map[uint16]map[uint32]int, len(a.counts))
	for z, marks := range a.counts {
		cp := make(map[uint32]int, len(marks))
		for m, c := range marks {
			if c > 0 {
				cp[m] = c
			}
		}
		out[z] = cp
	}
	return out
}

// PrimeSnapshot: optional one-time seeding with Dump.
func (a *ZoneMarkAggregator) PrimeSnapshot(ctx context.Context, maxEntries int) error {
	// Create a fresh connection for dump to avoid multicast interference
	dumpConn, err := conntrack.Dial(nil)
	if err != nil {
		return fmt.Errorf("failed to dial conntrack for prime snapshot: %w", err)
	}
	defer dumpConn.Close()

	flows, err := dumpConn.Dump(nil)
	if err != nil {
		return err
	}
	defer func() {
		for i := range flows {
			flows[i] = conntrack.Flow{}
		}
		flows = nil
		runtime.GC()
	}()

	processed := 0
	// zoneCounts := make(map[uint16]int)
	a.mu.Lock()
	for _, f := range flows {
		z := f.Zone
		m := f.Mark

		// Debug: Track zone distribution
		// zoneCounts[z]++

		zm, ok := a.counts[z]
		if !ok {
			zm = make(map[uint32]int)
			a.counts[z] = zm
		}
		zm[m]++
		processed++
		if maxEntries > 0 && processed >= maxEntries {
			break
		}
	}
	a.mu.Unlock()

	// Debug: Log zone distribution
	// log.Printf("conntrack prime seeded %d entries", processed)
	// for zone, count := range zoneCounts {
	// 	if count > 10 { // Only log zones with significant entries
	// 		log.Printf("DEBUG: Zone %d has %d entries", zone, count)
	// 	}
	// }
	return nil
}
