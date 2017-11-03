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

	c *genetlink.Conn
}

// New creates a new Linux Open vSwitch generic netlink client.
//
// If no OvS generic netlink families are available on this system, an
// error will be returned which can be checked using os.IsNotExist.
func New() (*Client, error) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		return nil, err
	}

	return newClient(c)
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
	return c.c.Close()
}

// init initializes the generic netlink family service of Client.
func (c *Client) init(families []genetlink.Family) error {
	// Assume 4 families present.
	var gotf int
	const wantf = 4

	for _, f := range families {
		// Ignore any families without the OVS prefix.
		if !strings.HasPrefix(f.Name, "ovs_") {
			continue
		}

		gotf++
		if err := c.initFamily(f); err != nil {
			return err
		}
	}

	// No families; return error for os.IsNotExist check.
	if gotf == 0 {
		return os.ErrNotExist
	}

	if gotf != wantf {
		return fmt.Errorf("expected %d OVS generic netlink families, but found %d",
			wantf, gotf)
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
	case ovsh.FlowFamily, ovsh.PacketFamily, ovsh.VportFamily:
		// TODO(mdlayher): populate.
		return nil
	}

	return fmt.Errorf("unrecognized OVS generic netlink family: %q", f.Name)
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
