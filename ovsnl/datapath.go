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
	"unsafe"

	"github.com/digitalocean/go-openvswitch/ovsnl/internal/ovsh"
	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
)

// A DatapathService provides access to methods which interact with the
// "ovs_datapath" generic netlink family.
type DatapathService struct {
	c *Client
	f genetlink.Family
}

// A Datapath is an Open vSwitch in-kernel datapath.
type Datapath struct {
	Index         int
	Name          string
	Features      DatapathFeatures
	Stats         DatapathStats
	MegaflowStats DatapathMegaflowStats
}

// DatapathFeatures is a set of bit flags that specify features for a datapath.
type DatapathFeatures uint32

// Possible DatapathFeatures flag values.
const (
	DatapathFeaturesUnaligned DatapathFeatures = ovsh.DpFUnaligned
	DatapathFeaturesVPortPIDs DatapathFeatures = ovsh.DpFVportPids
)

// String returns the string representation of a DatapathFeatures.
func (f DatapathFeatures) String() string {
	names := []string{
		"unaligned",
		"vportpids",
	}

	var s string
	for i, name := range names {
		if f&(1<<uint(i)) != 0 {
			if s != "" {
				s += "|"
			}

			s += name
		}
	}

	if s == "" {
		s = "0"
	}

	return s
}

// DatapathStats contains statistics about packets that have passed
// through a Datapath.
type DatapathStats struct {
	// Number of flow table matches.
	Hit uint64
	// Number of flow table misses.
	Missed uint64
	// Number of misses not sent to userspace.
	Lost uint64
	// Number of flows present.
	Flows uint64
}

// DatapathMegaflowStats contains statistics about mega flow mask
// usage for a Datapath.
type DatapathMegaflowStats struct {
	// Number of masks used for flow lookups.
	MaskHits uint64
	// Number of masks for the datapath.
	Masks uint32
}

// List lists all Datapaths in the kernel.
func (s *DatapathService) List() ([]Datapath, error) {
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: ovsh.DpCmdGet,
			Version: uint8(s.f.Version),
		},
		// Query all datapaths.
		Data: headerBytes(ovsh.Header{
			Ifindex: 0,
		}),
	}

	flags := netlink.HeaderFlagsRequest | netlink.HeaderFlagsDump
	msgs, err := s.c.c.Execute(req, s.f.ID, flags)
	if err != nil {
		return nil, err
	}

	return parseDatapaths(msgs)
}

// parseDatapaths parses a slice of Datapaths from a slice of generic netlink
// messages.
func parseDatapaths(msgs []genetlink.Message) ([]Datapath, error) {
	dps := make([]Datapath, 0, len(msgs))

	for _, m := range msgs {
		// Fetch the header at the beginning of the message.
		h, err := parseHeader(m.Data)
		if err != nil {
			return nil, err
		}

		dp := Datapath{
			Index: int(h.Ifindex),
		}

		// Skip the header to parse attributes.
		attrs, err := netlink.UnmarshalAttributes(m.Data[sizeofHeader:])
		if err != nil {
			return nil, err
		}

		for _, a := range attrs {
			switch a.Type {
			case ovsh.DpAttrName:
				dp.Name = nlenc.String(a.Data)
			case ovsh.DpAttrUserFeatures:
				dp.Features = DatapathFeatures(nlenc.Uint32(a.Data))
			case ovsh.DpAttrStats:
				dp.Stats, err = parseDPStats(a.Data)
				if err != nil {
					return nil, err
				}
			case ovsh.DpAttrMegaflowStats:
				dp.MegaflowStats, err = parseDPMegaflowStats(a.Data)
				if err != nil {
					return nil, err
				}
			}
		}

		dps = append(dps, dp)
	}

	return dps, nil
}

// parseDPStats converts a byte slice into DatapathStats.
func parseDPStats(b []byte) (DatapathStats, error) {
	// Verify that the byte slice is the correct length before doing
	// unsafe casts.
	if want, got := sizeofDPStats, len(b); want != got {
		return DatapathStats{}, fmt.Errorf("unexpected datapath stats structure size, want %d, got %d", want, got)
	}

	s := *(*ovsh.DPStats)(unsafe.Pointer(&b[0]))
	return DatapathStats{
		Hit:    s.Hit,
		Missed: s.Missed,
		Lost:   s.Lost,
		Flows:  s.Flows,
	}, nil
}

// parseDPMegaflowStats converts a byte slice into DatapathMegaflowStats.
func parseDPMegaflowStats(b []byte) (DatapathMegaflowStats, error) {
	// Verify that the byte slice is the correct length before doing
	// unsafe casts.
	if want, got := sizeofDPMegaflowStats, len(b); want != got {
		return DatapathMegaflowStats{}, fmt.Errorf("unexpected datapath megaflow stats structure size, want %d, got %d", want, got)
	}

	s := *(*ovsh.DPMegaflowStats)(unsafe.Pointer(&b[0]))

	return DatapathMegaflowStats{
		MaskHits: s.Mask_hit,
		Masks:    s.Masks,
	}, nil
}
