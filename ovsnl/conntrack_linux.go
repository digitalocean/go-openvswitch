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
	"fmt"
	"log"
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

// ZoneMarkAggregator keeps live counts (zmKey -> count) with bounded ingestion
type ZoneMarkAggregator struct {
	// primary counts (zmKey -> count) - simplified flat mapping
	counts   map[ZmKey]int
	countsMu sync.RWMutex

	// conntrack listening connection
	listenCli *conntrack.Conn

	// lifecycle
	stopCh    chan struct{}
	stoppedCh chan struct{}
	wg        sync.WaitGroup

	// bounded event ingestion
	eventsCh chan conntrack.Event

	// aggregated DESTROY deltas (bounded by destroyDeltaCap)
	deltaMu       sync.Mutex
	destroyDeltas map[ZmKey]int

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
		counts:                  make(map[ZmKey]int),
		listenCli:               listenCli,
		stopCh:                  make(chan struct{}),
		stoppedCh:               make(chan struct{}),
		eventsCh:                make(chan conntrack.Event, eventChanSize),
		destroyDeltas:           make(map[ZmKey]int),
		lastEventTime:           time.Now(),
		lastHealthCheck:         time.Now(),
		initialSnapshotComplete: false,
		initialSnapshotError:    nil,
	}

	return a, nil
}

// Start subscribes to NEW/DESTROY/UPDATE events and maintains counts with bounded ingestion.
func (a *ZoneMarkAggregator) Start() error {

	if err := a.startEventListener(); err != nil {
		return err
	}

	for i := 0; i < eventWorkerCount; i++ {
		a.wg.Add(1)
		go a.eventWorker(i)
	}

	a.wg.Add(1)
	go a.destroyFlusher()

	a.wg.Add(1)
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

	errCh, err := a.listenCli.Listen(libEvents, 10, groups)
	if err != nil {
		return fmt.Errorf("failed to listen to conntrack events: %w", err)
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
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
	defer a.wg.Done()
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
	key := ZmKey{Zone: f.Zone, Mark: f.Mark}

	if ev.Type == conntrack.EventNew {
		a.countsMu.Lock()
		a.counts[key]++
		a.countsMu.Unlock()
		return
	}

	if ev.Type == conntrack.EventDestroy {
		a.deltaMu.Lock()
		if len(a.destroyDeltas) < destroyDeltaCap {
			a.destroyDeltas[key]++
			if len(a.destroyDeltas) > 50000 { // If we have >50K deltas, flush immediately
				deltas := a.destroyDeltas
				a.destroyDeltas = make(map[ZmKey]int)
				// Acquire countsMu while still holding deltaMu to maintain lock ordering
				a.countsMu.Lock()
				a.deltaMu.Unlock()
				// Apply deltas immediately to minimize lag during extreme load
				a.applyDeltasImmediatelyUnsafe(deltas)
				a.countsMu.Unlock()
				return
			}
			// Log every 1000 DESTROY events to verify they're being received
			if len(a.destroyDeltas)%1000 == 0 {
				log.Printf("DESTROY events: %d entries in destroyDeltas (zone=%d, mark=%d)", len(a.destroyDeltas), key.Zone, key.Mark)
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

// applyDeltasImmediatelyUnsafe applies deltas immediately to minimize lag during extreme load
// This method assumes countsMu is already held by the caller
func (a *ZoneMarkAggregator) applyDeltasImmediatelyUnsafe(deltas map[ZmKey]int) {
	totalDecrements := 0
	for k, cnt := range deltas {
		existing, ok := a.counts[k]
		if !ok {
			atomic.AddInt64(&a.missedEvents, int64(cnt))
			continue
		}
		if existing <= cnt {
			delete(a.counts, k)
			totalDecrements += existing
		} else {
			a.counts[k] = existing - cnt
			totalDecrements += cnt
		}
	}
}

// destroyFlusher periodically applies the aggregated DESTROY deltas into counts
// Uses adaptive flushing: more frequent during high event rates for minimal lag
func (a *ZoneMarkAggregator) destroyFlusher() {
	defer a.wg.Done()
	ticker := time.NewTicker(destroyFlushIntvl)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			log.Printf("Destroy flusher stopping, final flush...")
			a.flushDestroyDeltas()
			return
		case <-ticker.C:
			// Adaptive flushing: flush more frequently during high event rates
			a.countsMu.RLock()
			eventRate := a.eventRate
			a.countsMu.RUnlock()

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
	// First acquire deltaMu to check and swap deltas
	a.deltaMu.Lock()
	defer a.deltaMu.Unlock()
	if len(a.destroyDeltas) == 0 {
		return
	}
	deltas := a.destroyDeltas
	a.destroyDeltas = make(map[ZmKey]int)

	// Now acquire countsMu while still holding deltaMu to ensure atomicity
	a.countsMu.Lock()
	defer a.countsMu.Unlock()

	totalDecrements := 0
	for k, cnt := range deltas {
		existing, ok := a.counts[k]
		if !ok {
			atomic.AddInt64(&a.missedEvents, int64(cnt))
			continue
		}
		if existing <= cnt {
			delete(a.counts, k)
			totalDecrements += existing
		} else {
			a.counts[k] = existing - cnt
			totalDecrements += cnt
		}
	}
}

// Snapshot returns a safe copy of counts.
func (a *ZoneMarkAggregator) Snapshot() map[ZmKey]int {
	a.flushDestroyDeltas()
	a.countsMu.RLock()
	defer a.countsMu.RUnlock()

	out := make(map[ZmKey]int, len(a.counts))
	for k, c := range a.counts {
		if c > 0 {
			out[k] = c
		}
	}
	return out
}

// startHealthMonitoring periodically logs aggregator health
func (a *ZoneMarkAggregator) startHealthMonitoring() {
	defer a.wg.Done()
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

	if missed > dropsWarnThreshold {
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
	a.wg.Wait() // Wait for all goroutines to exit cleanly
	if a.listenCli != nil {
		if err := a.listenCli.Close(); err != nil {
			log.Printf("Error closing listenCli during cleanup: %v", err)
		}
	}
	a.flushDestroyDeltas()
}

// RestartListener attempts to restart the conntrack event listener
func (a *ZoneMarkAggregator) RestartListener() error {
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
