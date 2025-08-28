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
	"context" // Used in commented aggregator code
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/digitalocean/go-openvswitch/ovsnl/internal/ovsh"
	"github.com/mdlayher/genetlink"
)

var _ = context.Background // Used in commented aggregator code

// Sizes of various structures, used in unsafe casts.
const (
	sizeofHeader = int(unsafe.Sizeof(ovsh.Header{}))

	sizeofDPStats         = int(unsafe.Sizeof(ovsh.DPStats{}))
	sizeofDPMegaflowStats = int(unsafe.Sizeof(ovsh.DPMegaflowStats{}))
)

// A Client is a Linux Open vSwitch generic netlink client.
type Client struct {
	// Datapath provides access to DatapathService methods.
	Datapath *DatapathService

	c         *genetlink.Conn
	Conntrack *ConntrackService
	Agg       *ZoneMarkAggregator
}

// New creates a new Linux Open vSwitch generic netlink client.
//
// If no OvS generic netlink families are available on this system, an
// error will be returned which can be checked using os.IsNotExist.
func New() (*Client, error) {
	c := &Client{} // Create client instance first

	// Initialize the underlying genetlink connection.
	conn, err := genetlink.Dial(nil)
	if err != nil {
		return nil, err
	}
	c.c = conn

	// Initialize services.
	families, err := c.c.ListFamilies()
	if err != nil {
		_ = c.c.Close()
		return nil, err
	}

	if err := c.init(families); err != nil {
		_ = c.c.Close()
		return nil, err
	}

	// Initialize ConntrackService directly, as it manages its own internal conntrack.Conn
	conntrackService, err := NewConntrackService() // This will establish ti-mo/conntrack's connection
	if err != nil {
		_ = c.c.Close() // Ensure main client connection is closed
		return nil, fmt.Errorf("failed to create ConntrackService: %w", err)
	}
	c.Conntrack = conntrackService

	// Re-enable aggregator now that we've eliminated the stats collection issue
	agg, err := NewZoneMarkAggregator(conntrackService)
	if err != nil {
		// Log the error but continue without aggregator
		fmt.Printf("Warning: Failed to create conntrack aggregator: %v (continuing without event-driven aggregation)\n", err)
		c.Agg = nil
	} else {
		if err := agg.Start(); err != nil {
			// Log the error but continue without aggregator
			fmt.Printf("Warning: Failed to start conntrack aggregator: %v (continuing without event-driven aggregation)\n", err)
			agg.Stop() // Clean up the failed aggregator
			c.Agg = nil
		} else {
			c.Agg = agg
		}
	}

	// Only run prime snapshot if aggregator is available
	if c.Agg != nil {
		go func() {
			_ = c.Agg.PrimeSnapshot(context.Background(), 200000)
		}()
	}

	return c, nil
}

// newClient is the internal Client constructor, used in tests.
func newClient(c *genetlink.Conn) (*Client, error) {
	// Must ensure that the generic netlink connection is closed on any errors
	// that occur before it is returned to the caller.

	families, err := c.ListFamilies()
	if err != nil {
		_ = c.Close()
		return nil, err
	}

	client := &Client{c: c}
	if err := client.init(families); err != nil {
		_ = c.Close()
		return nil, err
	}

	return client, nil
}

// Close closes the Client's generic netlink connection.
func (c *Client) Close() error {
	var errs []error

	if c.Agg != nil {
		c.Agg.Stop()
	}

	if c.c != nil {
		if err := c.c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.Conntrack != nil {
		if err := c.Conntrack.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing client: %v", errs)
	}
	return nil
}

// init initializes the generic netlink family service of Client.
func (c *Client) init(families []genetlink.Family) error {
	var gotf int

	for _, f := range families {
		// Initialize OVS-specific families
		if strings.HasPrefix(f.Name, "ovs_") {
			if err := c.initFamily(f); err != nil {
				// Log but continue if an OVS family fails to init
				fmt.Printf("Warning: failed to initialize OVS family %q: %v\n", f.Name, err)
				continue
			}
		} else if f.Name == "nf_conntrack" { // Explicitly initialize for Netfilter conntrack family
			// The ConntrackService is initialized separately by NewConntrackService(),
			// so we just acknowledge this family exists.
			// No direct assignment to c.Conntrack here because it manages its own connection.
		} else {
			// Skip other non-OVS/non-conntrack families
			continue
		}
		gotf++
	}

	if gotf == 0 {
		return os.ErrNotExist
	}

	return nil
}

// initFamily initializes a single generic netlink family service.
func (c *Client) initFamily(f genetlink.Family) error {
	switch f.Name {
	case ovsh.DatapathFamily:
		c.Datapath = &DatapathService{
			f: f,
			c: c,
		}
		return nil
	default:
		// Unknown OVS netlink family, nothing we can do.
		return fmt.Errorf("unknown OVS generic netlink family: %q", f.Name)
	}
}

// headerBytes converts an ovsh.Header into a byte slice.
func headerBytes(h ovsh.Header) []byte {
	b := *(*[sizeofHeader]byte)(unsafe.Pointer(&h))
	return b[:]
}

// parseHeader converts a byte slice into ovsh.Header.
func parseHeader(b []byte) (ovsh.Header, error) {
	// Verify that the byte slice is long enough before doing unsafe casts.
	if l := len(b); l < sizeofHeader {
		return ovsh.Header{}, fmt.Errorf("not enough data for OVS message header: %d bytes", l)
	}

	h := *(*ovsh.Header)(unsafe.Pointer(&b[:sizeofHeader][0]))
	return h, nil
}
