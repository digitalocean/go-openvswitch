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

//
// Conntrack aggregator with bounded ingestion + DESTROY aggregation
// to handle massive bursts of conntrack DESTROY events without OOMing.
//

// Tunables - adjust for your environment
const (
	eventChanSize      = 512 * 1024
	eventWorkerCount   = 100
	destroyFlushIntvl  = 100 * time.Millisecond // flush aggregated DESTROYs every 100ms for minimal lag
	destroyDeltaCap    = 200000                 // maximum distinct (zone,mark) entries in destroyDeltas
	dropsWarnThreshold = 100                    // threshold of missedEvents to log a stronger warning
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
	zone uint16
	mark uint32
}

// ZoneMarkAggregator keeps live counts (zone -> mark -> count) with bounded ingestion
type ZoneMarkAggregator struct {
	// primary counts (zone -> mark -> count)
	counts map[uint16]map[uint32]int
	mu     sync.RWMutex

	// conntrack listening connection
	listenCli *conntrack.Conn

	// lifecycle
	stopCh    chan struct{}
	stoppedCh chan struct{}

	// bounded event ingestion
	eventsCh chan conntrack.Event

	// aggregated DESTROY deltas (bounded by destroyDeltaCap)
	deltaMu       sync.Mutex
	destroyDeltas map[zmKey]int

	// metrics / health
	eventCount      int64
	lastEventTime   time.Time
	eventRate       float64
	missedEvents    int64
	lastHealthCheck time.Time

	// initial snapshot state (we keep disabled for huge tables)
	initialSnapshotComplete bool
	initialSnapshotError    error
}

// NewZoneMarkAggregator creates a new aggregator with its own listening connection.
func NewZoneMarkAggregator() (*ZoneMarkAggregator, error) {
	log.Printf("Creating new conntrack zone mark aggregator...")

	// Create a separate connection for listening to events
	listenCli, err := conntrack.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create listening connection: %w", err)
	}

	if err := listenCli.SetReadBuffer(64 * 1024 * 1024); err != nil { // 64MB buffer for 1.4M events/sec
		log.Printf("Warning: Failed to set read buffer size: %v", err)
	}
	if err := listenCli.SetWriteBuffer(64 * 1024 * 1024); err != nil { // 64MB buffer for 1.4M events/sec
		log.Printf("Warning: Failed to set write buffer size: %v", err)
	}

	a := &ZoneMarkAggregator{
		counts:                  make(map[uint16]map[uint32]int),
		listenCli:               listenCli,
		stopCh:                  make(chan struct{}),
		stoppedCh:               make(chan struct{}),
		eventsCh:                make(chan conntrack.Event, eventChanSize),
		destroyDeltas:           make(map[zmKey]int),
		lastEventTime:           time.Now(),
		lastHealthCheck:         time.Now(),
		initialSnapshotComplete: false,
		initialSnapshotError:    nil,
	}

	log.Printf("Successfully created conntrack listening connection with event channel size %d", eventChanSize)
	return a, nil
}

// Start subscribes to NEW/DESTROY/UPDATE events and maintains counts with bounded ingestion.
func (a *ZoneMarkAggregator) Start() error {
	log.Printf("Starting conntrack event listener with bounded ingestion + DESTROY aggregation...")

	if err := a.startEventListener(); err != nil {
		return err
	}

	for i := 0; i < eventWorkerCount; i++ {
		go a.eventWorker(i)
	}

	go a.destroyFlusher()
	go a.startHealthMonitoring()

	go func() {
		a.initialSnapshotComplete = true
		a.initialSnapshotError = nil
	}()

	return nil
}

// startEventListener handles real-time conntrack events, pushing into bounded eventsCh.
func (a *ZoneMarkAggregator) startEventListener() error {
	libEvents := make(chan conntrack.Event, 8192)
	groups := []netfilter.NetlinkGroup{
		netfilter.GroupCTNew,
		netfilter.GroupCTDestroy,
		netfilter.GroupCTUpdate,
	}

	log.Printf("Subscribing to conntrack groups: %v", groups)

	errCh, err := a.listenCli.Listen(libEvents, 10, groups)
	if err != nil {
		return fmt.Errorf("failed to listen to conntrack events: %w", err)
	}

	go func() {
		eventCount := int64(0)
		rateWindow := make([]time.Time, 0, 100)

		for {
			select {
			case <-a.stopCh:
				log.Printf("Stopping lib->bounded relay after %d lib events", atomic.LoadInt64(&eventCount))
				return
			case e := <-errCh:
				if e != nil {
					log.Printf("conntrack listener error: %v", e)
					atomic.AddInt64(&a.missedEvents, 1)
				}
			case ev := <-libEvents:
				// Log every 1000 events to verify events are being received from netlink
				// if eventCount%1000 == 0 {
				// 	log.Printf("Received event from netlink: type=%d, zone=%d, mark=%d", ev.Type, ev.Flow.Zone, ev.Flow.Mark)
				// }

				select {
				case a.eventsCh <- ev:
					atomic.AddInt64(&eventCount, 1)
					atomic.StoreInt64(&a.eventCount, eventCount)
					a.lastEventTime = time.Now()

					rateWindow = append(rateWindow, a.lastEventTime)
					if len(rateWindow) > 100 {
						rateWindow = rateWindow[1:]
					}
					if len(rateWindow) > 1 {
						duration := rateWindow[len(rateWindow)-1].Sub(rateWindow[0])
						if duration > 0 {
							a.eventRate = float64(len(rateWindow)-1) / duration.Seconds()
						}
					}
				default:
					atomic.AddInt64(&a.missedEvents, 1)
					if atomic.LoadInt64(&a.missedEvents)%100 == 0 {
						log.Printf("Warning: eventsCh full, missedEvents=%d", atomic.LoadInt64(&a.missedEvents))
					}
				}
			}
		}
	}()

	return nil
}

// eventWorker consumes events from eventsCh and handles them
func (a *ZoneMarkAggregator) eventWorker(id int) {
	processedCount := 0

	for {
		select {
		case <-a.stopCh:
			log.Printf("Event worker %d stopping (processed %d events)", id, processedCount)
			return
		case ev := <-a.eventsCh:
			a.handleEvent(ev)
			processedCount++
			if atomic.LoadInt64(&a.eventCount)%100 == 0 {
				runtime.Gosched()
			}
		}
	}
}

// handleEvent processes a single event.
func (a *ZoneMarkAggregator) handleEvent(ev conntrack.Event) {
	f := ev.Flow
	key := zmKey{zone: f.Zone, mark: f.Mark}

	// Log every 1000 events to verify events are being processed
	// eventCount := atomic.LoadInt64(&a.eventCount)
	// if eventCount%1000 == 0 {
	// 	log.Printf("handleEvent: processed %d events, current event type=%d", eventCount, ev.Type)
	// }

	if ev.Type == conntrack.EventNew {
		a.mu.Lock()
		zm, ok := a.counts[f.Zone]
		if !ok {
			zm = make(map[uint32]int)
			a.counts[f.Zone] = zm
		}
		zm[f.Mark]++
		a.mu.Unlock()
		return
	}

	if ev.Type == conntrack.EventDestroy {
		a.deltaMu.Lock()
		if len(a.destroyDeltas) < destroyDeltaCap {
			a.destroyDeltas[key]++
			if len(a.destroyDeltas) > 50000 { // If we have >50K deltas, flush immediately
				deltas := a.destroyDeltas
				a.destroyDeltas = make(map[zmKey]int)
				a.deltaMu.Unlock()
				// Apply deltas immediately to minimize lag during extreme load
				a.applyDeltasImmediately(deltas)
				return
			}
			// Log every 1000 DESTROY events to verify they're being received
			if len(a.destroyDeltas)%1000 == 0 {
				log.Printf("DESTROY events: %d entries in destroyDeltas (zone=%d, mark=%d)", len(a.destroyDeltas), key.zone, key.mark)
			}
		} else {
			atomic.AddInt64(&a.missedEvents, 1)
			if atomic.LoadInt64(&a.missedEvents)%dropsWarnThreshold == 0 {
				log.Printf("Warning: destroyDeltas saturated (size=%d). missedEvents=%d", len(a.destroyDeltas), atomic.LoadInt64(&a.missedEvents))
			}
		}
		a.deltaMu.Unlock()
		return
	}
}

// applyDeltasImmediately applies deltas immediately to minimize lag during extreme load
func (a *ZoneMarkAggregator) applyDeltasImmediately(deltas map[zmKey]int) {
	log.Printf("applyDeltasImmediately: processing %d delta entries", len(deltas))

	a.mu.Lock()
	defer a.mu.Unlock()

	totalDecrements := 0
	for k, cnt := range deltas {
		zm, ok := a.counts[k.zone]
		if !ok {
			atomic.AddInt64(&a.missedEvents, int64(cnt))
			continue
		}
		existing := zm[k.mark]
		if existing <= cnt {
			delete(zm, k.mark)
			if len(zm) == 0 {
				delete(a.counts, k.zone)
			}
			totalDecrements += existing
		} else {
			zm[k.mark] = existing - cnt
			totalDecrements += cnt
		}
	}

	if len(deltas) > 0 {
		log.Printf("applyDeltasImmediately: applied %d deltas, decremented %d total entries, missedEvents=%d",
			len(deltas), totalDecrements, atomic.LoadInt64(&a.missedEvents))
	}
}

// destroyFlusher periodically applies the aggregated DESTROY deltas into counts
// Uses adaptive flushing: more frequent during high event rates for minimal lag
func (a *ZoneMarkAggregator) destroyFlusher() {
	ticker := time.NewTicker(destroyFlushIntvl)
	defer ticker.Stop()

	log.Printf("Destroy flusher started (interval: %v)", destroyFlushIntvl)

	for {
		select {
		case <-a.stopCh:
			log.Printf("Destroy flusher stopping, final flush...")
			a.flushDestroyDeltas()
			return
		case <-ticker.C:
			// Adaptive flushing: flush more frequently during high event rates
			a.mu.RLock()
			eventRate := a.eventRate
			a.mu.RUnlock()

			if eventRate > 500000 { // Very high event rate (>500K/sec)
				// Flush immediately and reset ticker for faster interval
				a.flushDestroyDeltas()
				ticker.Reset(50 * time.Millisecond) // 50ms during extreme load
			} else if eventRate > 100000 { // High event rate (>100K/sec)
				a.flushDestroyDeltas()
				ticker.Reset(100 * time.Millisecond) // 100ms during high load
			} else if eventRate > 10000 { // Medium event rate (>10K/sec)
				a.flushDestroyDeltas()
				ticker.Reset(200 * time.Millisecond) // 200ms during medium load
			} else {
				// Normal flush
				a.flushDestroyDeltas()
				ticker.Reset(destroyFlushIntvl) // Back to normal interval
			}
		}
	}
}

// flushDestroyDeltas atomically swaps the delta map and applies decrements
func (a *ZoneMarkAggregator) flushDestroyDeltas() {
	a.deltaMu.Lock()
	if len(a.destroyDeltas) == 0 {
		a.deltaMu.Unlock()
		return
	}
	deltas := a.destroyDeltas
	a.destroyDeltas = make(map[zmKey]int)
	a.deltaMu.Unlock()

	log.Printf("flushDestroyDeltas: processing %d delta entries", len(deltas))

	a.mu.Lock()
	defer a.mu.Unlock()

	totalDecrements := 0
	for k, cnt := range deltas {
		zm, ok := a.counts[k.zone]
		if !ok {
			atomic.AddInt64(&a.missedEvents, int64(cnt))
			continue
		}
		existing := zm[k.mark]
		if existing <= cnt {
			delete(zm, k.mark)
			if len(zm) == 0 {
				delete(a.counts, k.zone)
			}
			totalDecrements += existing
		} else {
			zm[k.mark] = existing - cnt
			totalDecrements += cnt
		}
	}

	if len(deltas) > 0 {
		log.Printf("flushDestroyDeltas: applied %d deltas, decremented %d total entries, missedEvents=%d",
			len(deltas), totalDecrements, atomic.LoadInt64(&a.missedEvents))
	}
}

// Snapshot returns a safe copy of counts.
func (a *ZoneMarkAggregator) Snapshot() map[uint16]map[uint32]int {
	a.flushDestroyDeltas()
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

// // GetTotalCount returns the total counted entries (best-effort)
// func (a *ZoneMarkAggregator) GetTotalCount() int {
// 	a.mu.RLock()
// 	defer a.mu.RUnlock()
// 	total := 0
// 	for _, marks := range a.counts {
// 		for _, c := range marks {
// 			total += c
// 		}
// 	}
// 	return total
// }

// startHealthMonitoring periodically logs aggregator health
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

func (a *ZoneMarkAggregator) performHealthCheck() {
	missed := atomic.LoadInt64(&a.missedEvents)
	eventCount := atomic.LoadInt64(&a.eventCount)

	if missed > 0 {
		log.Printf("Health check: missed_events=%d, event_count=%d, event_rate=%.2f",
			missed, eventCount, a.eventRate)
	}
	if missed > dropsWarnThreshold {
		log.Printf("Health check: missed events exceeded threshold (%d); attempting listener restart", missed)
		if err := a.RestartListener(); err != nil {
			log.Printf("Health check: RestartListener failed: %v", err)
		} else {
			atomic.StoreInt64(&a.missedEvents, 0)
			log.Printf("Health check: Listener restarted successfully")
		}
	}
	a.lastHealthCheck = time.Now()
}

// Stop cancels listening and closes the connection.
func (a *ZoneMarkAggregator) Stop() {
	close(a.stopCh)
	time.Sleep(20 * time.Millisecond)
	if a.listenCli != nil {
		a.listenCli.Close()
	}
	a.flushDestroyDeltas()
}

// RestartListener attempts to restart the conntrack event listener
func (a *ZoneMarkAggregator) RestartListener() error {
	log.Printf("Attempting to restart conntrack event listener...")
	if a.listenCli != nil {
		_ = a.listenCli.Close()
	}
	listenCli, err := conntrack.Dial(nil)
	if err != nil {
		return fmt.Errorf("failed to create new listening connection: %w", err)
	}
	a.listenCli = listenCli
	return a.startEventListener()
}
