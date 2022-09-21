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
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
)

// Sizes of various structures, used in unsafe casts.
const (
	sizeofHeader = int(unsafe.Sizeof(ovsh.Header{}))

	sizeofDPStats         = int(unsafe.Sizeof(ovsh.DPStats{}))
	sizeofDPMegaflowStats = int(unsafe.Sizeof(ovsh.DPMegaflowStats{}))
	sizeofVportStats      = int(unsafe.Sizeof(ovsh.VportStats{}))
)

// A Client is a Linux Open vSwitch generic netlink client.
type Client struct {
	// Datapath provides access to DatapathService methods.
	Datapath *DatapathService
	// Vport provides access to VportService methods.
	Vport *VportService

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
	var gotf int

	for _, f := range families {
		// Ignore any families without the OVS prefix.
		if !strings.HasPrefix(f.Name, "ovs_") {
			continue
		}
		// Ignore any families that might be unknown.
		if err := c.initFamily(f); err != nil {
			continue
		}
		gotf++
	}

	// No known families; return error for os.IsNotExist check.
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
	case ovsh.VportFamily:
		c.Vport = &VportService{
			c: c,
			f: f,
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

// NlMsgBuilder to build genetlink message
type NlMsgBuilder struct {
	msg *genetlink.Message
}

// NewNlMsgBuilder construct a netlink message builder with genetlink.Message
func NewNlMsgBuilder() *NlMsgBuilder {
	return &NlMsgBuilder{msg: &genetlink.Message{}}
}

// PutGenlMsgHdr set msg header with genetlink.Header
func (nlmsg *NlMsgBuilder) PutGenlMsgHdr(command, version uint8) {
	nlmsg.msg.Header = genetlink.Header{
		Command: command,
		Version: version,
	}
}

// PutOvsHeader set ovs header with ovsh.Header
func (nlmsg *NlMsgBuilder) PutOvsHeader(dpID int32) {
	nlmsg.msg.Data = headerBytes(ovsh.Header{Ifindex: dpID})
}

// PutStringAttr put attribute with string value
func (nlmsg *NlMsgBuilder) PutStringAttr(typ uint16, value string) error {
	attrs := []netlink.Attribute{
		{
			Type: typ,
			Data: nlenc.Bytes(value),
		},
	}
	attr, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		return fmt.Errorf("marshal string attributes failed:%s", err)
	}

	nlmsg.msg.Data = append(nlmsg.msg.Data[:], attr...)
	return nil
}

// PutUint8Attr put attribute with uint8 value
func (nlmsg *NlMsgBuilder) PutUint8Attr(typ uint16, value uint8) error {
	attrs := []netlink.Attribute{
		{
			Type: typ,
			Data: nlenc.Uint8Bytes(value),
		},
	}
	attr, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		return fmt.Errorf("marshal uint8 attributes failed:%s", err)
	}

	nlmsg.msg.Data = append(nlmsg.msg.Data[:], attr...)
	return nil
}

// PutUint16Attr put attribute with uint16 value
func (nlmsg *NlMsgBuilder) PutUint16Attr(typ uint16, value uint16) error {
	attrs := []netlink.Attribute{
		{
			Type: typ,
			Data: nlenc.Uint16Bytes(value),
		},
	}
	attr, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		return fmt.Errorf("marshal uint16 attributes failed:%s", err)
	}

	nlmsg.msg.Data = append(nlmsg.msg.Data[:], attr...)
	return nil
}

// PutUint32Attr put attribute with uint32 value
func (nlmsg *NlMsgBuilder) PutUint32Attr(typ uint16, value uint32) error {
	attrs := []netlink.Attribute{
		{
			Type: typ,
			Data: nlenc.Uint32Bytes(value),
		},
	}
	attr, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		return fmt.Errorf("marshal uint32 attributes failed:%s", err)
	}

	nlmsg.msg.Data = append(nlmsg.msg.Data[:], attr...)
	return nil
}

// PutSliceAttr put attribute with slice byte value
func (nlmsg *NlMsgBuilder) PutSliceAttr(typ uint16, value []byte) error {
	attrs := []netlink.Attribute{
		{
			Type: typ,
			Data: value,
		},
	}
	attr, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		return fmt.Errorf("marshal slice byte attributes failed:%s", err)
	}

	nlmsg.msg.Data = append(nlmsg.msg.Data[:], attr...)
	return nil
}

// PutEmptyAttr put attribute with empty value
func (nlmsg *NlMsgBuilder) PutEmptyAttr(typ uint16) error {
	attrs := []netlink.Attribute{
		{
			Type: typ,
		},
	}
	attr, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		return fmt.Errorf("marshal empty attributes failed:%s", err)
	}

	nlmsg.msg.Data = append(nlmsg.msg.Data[:], attr...)
	return nil
}

// Message generic netlink message
func (nlmsg *NlMsgBuilder) Message() genetlink.Message {
	return *nlmsg.msg
}
