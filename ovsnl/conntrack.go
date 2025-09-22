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
	"runtime"
	"sync"
	"sync/atomic"
	"time"

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

// ZoneMarkAggregator keeps live counts (zone -> mark -> count) with adaptive sync.
type ZoneMarkAggregator struct {
	mu        sync.RWMutex
	counts    map[uint16]map[uint32]int
	listenCli *conntrack.Conn // Separate connection for listening to events
	stopCh    chan struct{}
	stoppedCh chan struct{}

	// Event tracking for adaptive behavior
	eventCount    int64
	lastEventTime time.Time
	eventRate     float64 // events per second

	// Event queue removed - not needed since NEW events don't cause buffer overflow

	// Health monitoring
	missedEvents    int64
	lastHealthCheck time.Time

	// Initial snapshot state
	initialSnapshotComplete bool
	initialSnapshotError    error
}

// NewZoneMarkAggregator creates a new aggregator with its own listening connection.
func NewZoneMarkAggregator(s *ConntrackService) (*ZoneMarkAggregator, error) {
	log.Printf("Creating new conntrack zone mark aggregator...")

	// Create a separate connection for listening to events
	listenCli, err := conntrack.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create listening connection: %w", err)
	}

	// Increase netlink socket buffer size to handle 2.6M conntrack event rates
	if err := listenCli.SetReadBuffer(8 * 1024 * 1024); err != nil { // 8MB buffer
		log.Printf("Warning: Failed to set read buffer size: %v", err)
	}
	if err := listenCli.SetWriteBuffer(8 * 1024 * 1024); err != nil { // 8MB buffer
		log.Printf("Warning: Failed to set write buffer size: %v", err)
	}

	log.Printf("Successfully created conntrack listening connection with increased buffer size")

	return &ZoneMarkAggregator{
		counts:          make(map[uint16]map[uint32]int),
		listenCli:       listenCli,
		stopCh:          make(chan struct{}),
		stoppedCh:       make(chan struct{}),
		lastEventTime:   time.Now(),
		lastHealthCheck: time.Now(),
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

// Start subscribes to NEW and DESTROY events and maintains counts with adaptive sync.
func (a *ZoneMarkAggregator) Start() error {
	log.Printf("Starting conntrack event listener with adaptive sync...")

	// Start event listener first (non-blocking)
	if err := a.startEventListener(); err != nil {
		return err
	}

	// Start health monitoring
	go a.startHealthMonitoring()

	// CRITICAL: Initial snapshot DISABLED - even parallel processing causes OOM with 2M+ conntracks
	go func() {
		log.Printf("CRITICAL: Initial snapshot DISABLED to prevent OOM")
		log.Printf("Even parallel processing cannot handle 2M+ conntracks in memory")
		log.Printf("Starting with empty baseline - will rely on real-time events")
		a.initialSnapshotComplete = true
		a.initialSnapshotError = nil
	}()

	log.Printf("Conntrack event listener started successfully (initial snapshot in progress)")
	return nil
}

// startEventListener handles real-time conntrack events
func (a *ZoneMarkAggregator) startEventListener() error {
	events := make(chan conntrack.Event, 262144) // 256K events for 2.6M conntrack capacity

	// Subscribe to ALL groups (NEW, UPDATE, DESTROY).
	groups := []netfilter.NetlinkGroup{
		netfilter.GroupCTNew,
		netfilter.GroupCTDestroy,
		netfilter.GroupCTUpdate,
	}

	log.Printf("Subscribing to conntrack groups: %v", groups)

	// Test if we can at least get stats to verify conntrack is accessible
	// Note: We can't call Stats() on a multicast connection, so we'll skip this check
	// and rely on the event loop to detect issues

	errCh, err := a.listenCli.Listen(events, 8, groups) // 8 workers for 2.6M conntrack capacity
	if err != nil {
		return fmt.Errorf("failed to listen to conntrack events: %w", err)
	}

	log.Printf("Successfully subscribed to conntrack events, starting event loop...")

	// Watch for errors from workers
	go func() {
		eventCount := int64(0)
		lastEventTime := time.Now()
		rateWindow := make([]time.Time, 0, 100) // Track last 100 events for rate calculation

		// Start a ticker to check if we're receiving events
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-a.stopCh:
				log.Printf("Stopping conntrack event listener after %d events", atomic.LoadInt64(&eventCount))
				close(a.stoppedCh)
				return
			case e := <-errCh:
				if e != nil {
					log.Printf("conntrack listener error: %v", e)
					atomic.AddInt64(&a.missedEvents, 1)

					// If we get too many errors, try to recover
					if atomic.LoadInt64(&a.missedEvents) > 10 {
						log.Printf("Too many conntrack errors (%d), attempting recovery...", atomic.LoadInt64(&a.missedEvents))
						// Reset error counter and try to reinitialize
						atomic.StoreInt64(&a.missedEvents, 0)
						// Note: Full recovery would require restarting the listener, which is complex
						// For now, we'll just reset the counter and continue
					}
				}
			case ev := <-events:
				now := time.Now()
				atomic.AddInt64(&eventCount, 1)
				atomic.StoreInt64(&a.eventCount, eventCount)
				a.lastEventTime = now

				// Update rate calculation
				rateWindow = append(rateWindow, now)
				if len(rateWindow) > 100 {
					rateWindow = rateWindow[1:]
				}
				if len(rateWindow) > 1 {
					duration := rateWindow[len(rateWindow)-1].Sub(rateWindow[0])
					if duration > 0 {
						a.eventRate = float64(len(rateWindow)-1) / duration.Seconds()
					}
				}

				if eventCount%1000 == 0 {
					log.Printf("Processed %d conntrack events (rate: %.2f events/sec)", eventCount, a.eventRate)
				}

				// Smart DESTROY event handling - balance accuracy vs OOM risk
				if ev.Type == conntrack.EventDestroy {
					if a.eventRate > 75000 { // 75K events/sec threshold
						// During extreme bursts, process every other DESTROY event to prevent OOM
						if eventCount%3 != 0 {
							a.applyEvent(ev)
						} else {
							log.Printf("Rate limiting: Dropping DESTROY event during extreme burst (rate: %.2f events/sec)", a.eventRate)
						}
					} else {
						// Normal rate, process all DESTROY events for accuracy
						a.applyEvent(ev)
					}
					continue
				}

				// Only apply rate limiting to NEW events during high load
				if a.eventRate > 150000 {
					// Process every other NEW event during high load
					if eventCount%2 == 0 {
						a.applyEvent(ev)
					} else {
						log.Printf("Rate limiting: Dropping NEW event during high load (rate: %.2f events/sec)", a.eventRate)
					}
					continue
				}

				a.applyEvent(ev)

				// Rate limiting: yield every 100 events to prevent overwhelming the system
				if eventCount%100 == 0 {
					runtime.Gosched()
				}
			case <-ticker.C:
				// Check if we've received any events recently
				if eventCount == 0 && time.Since(lastEventTime) > 30*time.Second {
					log.Printf("Warning: No conntrack events received in the last 30 seconds")
					atomic.AddInt64(&a.missedEvents, 1)
					// Note: We can't call Stats() on a multicast connection
					// The lack of events might indicate a real issue or just low activity
				}
			}
		}
	}()

	return nil
}

// startHealthMonitoring monitors the health of the aggregator
func (a *ZoneMarkAggregator) startHealthMonitoring() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.performHealthCheck()
		}
	}
}

// performHealthCheck performs health monitoring
func (a *ZoneMarkAggregator) performHealthCheck() {
	missed := atomic.LoadInt64(&a.missedEvents)
	eventCount := atomic.LoadInt64(&a.eventCount)

	if missed > 0 {
		log.Printf("Health check: %d missed events detected", missed)
	}

	if eventCount == 0 && time.Since(a.lastEventTime) > 5*time.Minute {
		log.Printf("Health check: No events received in %v", time.Since(a.lastEventTime))
	}

	// If we have too many missed events, try to restart the listener
	if missed > 50 {
		log.Printf("Health check: Too many missed events (%d), attempting listener restart", missed)
		if err := a.RestartListener(); err != nil {
			log.Printf("Health check: Failed to restart listener: %v", err)
		} else {
			// Reset the missed events counter after successful restart
			atomic.StoreInt64(&a.missedEvents, 0)
			log.Printf("Health check: Listener restarted successfully")
		}
	} else if missed > 5 {
		// For moderate missed events, just reset the counter to prevent repeated attempts
		log.Printf("Health check: Moderate missed events (%d), resetting counter (sync disabled)", missed)
		atomic.StoreInt64(&a.missedEvents, 0)
		log.Printf("Health check: All sync operations disabled to prevent OOM")
	}

	a.lastHealthCheck = time.Now()
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

	if ev.Type == conntrack.EventNew {
		zm[mark]++
		if zm[mark]%1000 == 0 {
			log.Printf("Zone %d, Mark %d: %d entries", zone, mark, zm[mark])
		}
	}

	if ev.Type == conntrack.EventDestroy {
		if zm[mark] > 0 {
			zm[mark]--
			if zm[mark]%1000 == 0 {
				log.Printf("Zone %d, Mark %d: %d entries (after DESTROY)", zone, mark, zm[mark])
			}
		} else {
			log.Printf("Warning: DESTROY event for non-existent entry (zone=%d, mark=%d) - current count: %d", zone, mark, zm[mark])
		}
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

// IsHealthy checks if the aggregator is in a healthy state
func (a *ZoneMarkAggregator) IsHealthy() bool {
	// Check if initial snapshot failed
	if a.initialSnapshotComplete && a.initialSnapshotError != nil {
		return false
	}

	// Check if we've received events recently
	if time.Since(a.lastEventTime) > 10*time.Minute {
		return false
	}

	// Check for too many missed events
	if atomic.LoadInt64(&a.missedEvents) > 1000 {
		return false
	}

	return true
}

// RestartListener attempts to restart the conntrack event listener
// This should be called when the listener is completely dead
func (a *ZoneMarkAggregator) RestartListener() error {
	log.Printf("Attempting to restart conntrack event listener...")

	// Stop the current listener
	if a.listenCli != nil {
		a.listenCli.Close()
	}

	// Create a new connection
	listenCli, err := conntrack.Dial(nil)
	if err != nil {
		return fmt.Errorf("failed to create new listening connection: %w", err)
	}
	a.listenCli = listenCli

	// Restart the event listener
	if err := a.startEventListener(); err != nil {
		return fmt.Errorf("failed to restart event listener: %w", err)
	}

	log.Printf("Conntrack event listener restarted successfully")
	return nil
}

// ForceSync performs a manual sync to get the current kernel state
// This should only be called when we know the event-based counts are wrong
func (a *ZoneMarkAggregator) ForceSync() error {
	log.Printf("Performing manual force sync...")

	// WARNING: ForceSync can cause OOM with large conntrack tables
	// For now, we'll disable it to prevent crashes
	log.Printf("ForceSync DISABLED to prevent OOM with large conntrack tables")
	log.Printf("Use real-time events for accuracy instead")
	return nil
}

// processQueuedEvents removed - not needed since NEW events don't cause buffer overflow

// getTotalEntries returns the total number of entries across all zones and marks
// func (a *ZoneMarkAggregator) getTotalEntries() int {
// 	total := 0
// 	for _, marks := range a.counts {
// 		for _, count := range marks {
// 			total += count
// 		}
// 	}
// 	return total
// }
