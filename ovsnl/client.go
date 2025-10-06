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
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/digitalocean/go-openvswitch/ovsnl/internal/ovsh"
	"github.com/mdlayher/genetlink"
)

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

	c   *genetlink.Conn
	Agg *ZoneMarkAggregator
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

	// Initialize aggregator as nil - will be created when needed
	c.Agg = nil

	return c, nil
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
			// Acknowledge that conntrack family exists - aggregator will handle conntrack operations
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
		fmt.Printf("🔍 parseHeader: not enough data - got %d bytes, need %d\n", l, sizeofHeader)
		return ovsh.Header{}, fmt.Errorf("not enough data for OVS message header: %d bytes", l)
	}

	h := *(*ovsh.Header)(unsafe.Pointer(&b[:sizeofHeader][0]))
	return h, nil
}
