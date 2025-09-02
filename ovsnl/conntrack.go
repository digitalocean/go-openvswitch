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
	"fmt"
	"log"
	"net"
	"time"

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
	log.Printf("Creating new conntrack zone mark aggregator...")

	// Create a separate connection for listening to events
	listenCli, err := conntrack.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create listening connection: %w", err)
	}

	log.Printf("Successfully created conntrack listening connection")
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

// Start subscribes to NEW and DESTROY events and maintains counts.
func (a *ZoneMarkAggregator) Start() error {
	log.Printf("Starting conntrack event listener...")
	events := make(chan conntrack.Event, 8192)

	// Subscribe to ALL groups (NEW, UPDATE, DESTROY).
	groups := []netfilter.NetlinkGroup{
		netfilter.GroupCTNew,
		netfilter.GroupCTDestroy,
		netfilter.GroupCTUpdate,
	}

	log.Printf("Subscribing to conntrack groups: %v", groups)

	// Test if we can at least get stats to verify conntrack is accessible
	if _, err := a.listenCli.Stats(); err != nil {
		log.Printf("Warning: Cannot get conntrack stats: %v - this might indicate permission issues", err)
	}

	errCh, err := a.listenCli.Listen(events, 2, groups) // 2 workers; tune as needed
	if err != nil {
		return fmt.Errorf("failed to listen to conntrack events: %w", err)
	}

	log.Printf("Successfully subscribed to conntrack events, starting event loop...")
	// Watch for errors from workers
	go func() {
		eventCount := 0
		lastEventTime := time.Now()

		// Start a ticker to check if we're receiving events
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-a.stopCh:
				log.Printf("Stopping conntrack event listener after %d events", eventCount)
				close(a.stoppedCh)
				return
			case e := <-errCh:
				if e != nil {
					log.Printf("conntrack listener error: %v", e)
				}
			case ev := <-events:
				eventCount++
				lastEventTime = time.Now()
				if eventCount%100 == 0 {
					log.Printf("Processed %d conntrack events", eventCount)
				}
				a.applyEvent(ev)
			case <-ticker.C:
				// Check if we've received any events recently
				if eventCount == 0 && time.Since(lastEventTime) > 30*time.Second {
					log.Printf("Warning: No conntrack events received in the last 30 seconds")
					// Try to get stats to see if conntrack is still accessible
					if stats, err := a.listenCli.Stats(); err != nil {
						log.Printf("Warning: Cannot get conntrack stats: %v", err)
					} else {
						log.Printf("Conntrack stats still accessible: %+v", stats)
					}
				}
			}
		}
	}()

	log.Printf("Conntrack event listener started successfully")
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

//TODO : To confirm if we absolutely need this, can omit if eventual consistency is ok

// PrimeSnapshot: optional one-time seeding with Dump.
// func (a *ZoneMarkAggregator) PrimeSnapshot(ctx context.Context, maxEntries int) error {
// 	log.Printf("Starting conntrack prime snapshot with max entries: %d", maxEntries)

// 	// Create a fresh connection for dump to avoid multicast interference
// 	dumpConn, err := conntrack.Dial(nil)
// 	if err != nil {
// 		return fmt.Errorf("failed to dial conntrack for prime snapshot: %w", err)
// 	}
// 	defer dumpConn.Close()

// 	log.Printf("Successfully connected to conntrack, starting dump...")

// 	// First try to get stats to verify we have access
// 	if stats, err := dumpConn.Stats(); err != nil {
// 		log.Printf("Warning: Cannot get conntrack stats: %v - this might indicate permission issues", err)
// 	} else {
// 		log.Printf("Conntrack stats accessible: %+v", stats)
// 	}

// 	flows, err := dumpConn.Dump(nil)
// 	if err != nil {
// 		// Check if it's a permission error
// 		if err.Error() == "operation not permitted" || err.Error() == "permission denied" {
// 			log.Printf("Permission denied when trying to dump conntrack - you may need to run with elevated privileges")
// 			return fmt.Errorf("permission denied when dumping conntrack: %w", err)
// 		}
// 		return fmt.Errorf("failed to dump conntrack flows: %w", err)
// 	}

// 	if len(flows) == 0 {
// 		log.Printf("Warning: conntrack dump returned 0 flows - this might indicate a permission issue or empty conntrack table")
// 		// Check if we can at least get stats
// 		if stats, err := dumpConn.Stats(); err != nil {
// 			log.Printf("Failed to get conntrack stats: %v", err)
// 		} else {
// 			log.Printf("Conntrack stats: %+v", stats)
// 		}
// 		return nil
// 	}

// 	log.Printf("Dumped %d conntrack flows", len(flows))
// 	defer func() {
// 		for i := range flows {
// 			flows[i] = conntrack.Flow{}
// 		}
// 		flows = nil
// 		runtime.GC()
// 	}()

// 	processed := 0
// 	zoneCounts := make(map[uint16]int)
// 	a.mu.Lock()
// 	for _, f := range flows {
// 		z := f.Zone
// 		m := f.Mark

// 		// Debug: Track zone distribution
// 		zoneCounts[z]++

// 		zm, ok := a.counts[z]
// 		if !ok {
// 			zm = make(map[uint32]int)
// 			a.counts[z] = zm
// 		}
// 		zm[m]++
// 		processed++
// 		if maxEntries > 0 && processed >= maxEntries {
// 			log.Printf("Reached max entries limit (%d), stopping processing", maxEntries)
// 			break
// 		}
// 	}
// 	a.mu.Unlock()

// 	// Debug: Log zone distribution
// 	log.Printf("conntrack prime seeded %d entries", processed)
// 	for zone, count := range zoneCounts {
// 		if count > 10 { // Only log zones with significant entries
// 			log.Printf("DEBUG: Zone %d has %d entries", zone, count)
// 		}
// 	}

// 	// Log final state
// 	a.mu.RLock()
// 	totalZones := len(a.counts)
// 	totalEntries := 0
// 	for _, marks := range a.counts {
// 		for _, cnt := range marks {
// 			totalEntries += cnt
// 		}
// 	}
// 	a.mu.RUnlock()

// 	log.Printf("Prime snapshot completed: %d zones, %d total entries", totalZones, totalEntries)
// 	return nil
// }
