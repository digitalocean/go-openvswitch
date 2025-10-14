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

package ovsnl

import (
	"sync"
	"time"

	"github.com/ti-mo/conntrack"
)

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
	stopCh chan struct{}
	wg     sync.WaitGroup

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
}

// ZmKey is a compact key for (zone,mark)
type ZmKey struct {
	Zone uint16
	Mark uint32
}
