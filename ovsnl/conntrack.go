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
	eventChanSize      = 64 * 1024       // size of the bounded events channel
	eventWorkerCount   = 4               // number of goroutines consuming events
	destroyFlushIntvl  = 1 * time.Second // flush aggregated DESTROYs every second
	destroyDeltaCap    = 200000          // maximum distinct (zone,mark) entries in destroyDeltas
	dropsWarnThreshold = 100             // threshold of missedEvents to log a stronger warning
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

// zmKey is a compact key for (zone,mark)
type zmKey struct {
	zone uint16
	mark uint32
}

// ZoneMarkAggregator keeps live counts (zone -> mark -> count) with bounded ingestion
type ZoneMarkAggregator struct {
	// primary counts (zone -> mark -> count)
	mu     sync.RWMutex
	counts map[uint16]map[uint32]int

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
func NewZoneMarkAggregator(s *ConntrackService) (*ZoneMarkAggregator, error) {
	log.Printf("Creating new conntrack zone mark aggregator...")

	// Create a separate connection for listening to events
	listenCli, err := conntrack.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create listening connection: %w", err)
	}

	// Try to increase socket buffers (best-effort)
	if err := listenCli.SetReadBuffer(8 * 1024 * 1024); err != nil {
		log.Printf("Warning: Failed to set read buffer size: %v", err)
	}
	if err := listenCli.SetWriteBuffer(8 * 1024 * 1024); err != nil {
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

func NewConntrackService() (*ConntrackService, error) {
	return &ConntrackService{}, nil
}

func (s *ConntrackService) Close() error {
	return nil
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
		log.Printf("Initial snapshot DISABLED (to avoid OOM on large tables). Starting from empty baseline and relying on events.")
		a.initialSnapshotComplete = true
		a.initialSnapshotError = nil
	}()

	log.Printf("Conntrack aggregator started (workers=%d, eventChan=%d)", eventWorkerCount, eventChanSize)
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
				if eventCount%1000 == 0 {
					log.Printf("Received event from netlink: type=%d, zone=%d, mark=%d", ev.Type, ev.Flow.Zone, ev.Flow.Mark)
				}

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
	log.Printf("Event worker %d started", id)
	processedCount := 0

	for {
		select {
		case <-a.stopCh:
			log.Printf("Event worker %d stopping (processed %d events)", id, processedCount)
			return
		case ev := <-a.eventsCh:
			a.handleEvent(ev)
			processedCount++
			if processedCount%1000 == 0 {
				log.Printf("Event worker %d: processed %d events", id, processedCount)
			}
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
	eventCount := atomic.LoadInt64(&a.eventCount)
	if eventCount%1000 == 0 {
		log.Printf("handleEvent: processed %d events, current event type=%d", eventCount, ev.Type)
	}

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

// destroyFlusher periodically applies the aggregated DESTROY deltas into counts
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
			a.flushDestroyDeltas()
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

// GetTotalCount returns the total counted entries (best-effort)
func (a *ZoneMarkAggregator) GetTotalCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	total := 0
	for _, marks := range a.counts {
		for _, c := range marks {
			total += c
		}
	}
	return total
}

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
		log.Printf("Health check: missed_events=%d, event_count=%d, event_rate=%.2f, total_count=%d",
			missed, eventCount, a.eventRate, a.GetTotalCount())
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

// IsHealthy checks if the aggregator is in a healthy state
func (a *ZoneMarkAggregator) IsHealthy() bool {
	if a.initialSnapshotComplete && a.initialSnapshotError != nil {
		return false
	}
	if time.Since(a.lastEventTime) > 10*time.Minute {
		return false
	}
	if atomic.LoadInt64(&a.missedEvents) > 100000 {
		return false
	}
	return true
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

// ForceSync performs a manual sync (disabled for large tables)
func (a *ZoneMarkAggregator) ForceSync() error {
	log.Printf("ForceSync: disabled to avoid OOM with large conntrack tables")
	return fmt.Errorf("ForceSync disabled")
}
