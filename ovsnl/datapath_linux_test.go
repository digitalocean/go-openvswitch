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
	"testing"
	"unsafe"

	"github.com/digitalocean/go-openvswitch/ovsnl/internal/ovsh"
	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/genetlink/genltest"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

func TestClientDatapathListShortHeader(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Not enough data for ovsh.Header.
		t.Logf("🔍 Mock returning short data: %v", []byte{0xff, 0xff})
		return []genetlink.Message{
			{
				Data: []byte{0xff, 0xff}, // Only 2 bytes, but sizeofHeader is 4
			},
		}, nil
	}))

	c, err := newTestClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	t.Logf("🔍 About to call c.Datapath.List()")
	dps, err := c.Datapath.List()
	if err == nil {
		t.Logf("🔍 DEBUG: No error occurred, got %d datapaths", len(dps))
		for i, dp := range dps {
			t.Logf("🔍 DEBUG: Datapath[%d]: Index=%d, Name=%q", i, dp.Index, dp.Name)
		}
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientDatapathListBadStats(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Valid header; not enough data for ovsh.DPStats.
		return []genetlink.Message{{
			Data: append(
				// ovsh.Header (4 bytes).
				[]byte{0xff, 0xff, 0xff, 0xff},
				// netlink attributes.
				mustMarshalAttributes([]netlink.Attribute{{
					Type: ovsh.DpAttrStats,
					Data: []byte{0xff}, // Only 1 byte, but sizeofDPStats is 32 bytes
				}})...,
			),
		}}, nil
	}))

	c, err := newTestClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	_, err = c.Datapath.List()
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientDatapathListBadMegaflowStats(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Valid header; not enough data for ovsh.DPMegaflowStats.
		return []genetlink.Message{{
			Data: append(
				// ovsh.Header (4 bytes).
				[]byte{0xff, 0xff, 0xff, 0xff},
				// netlink attributes.
				mustMarshalAttributes([]netlink.Attribute{{
					Type: ovsh.DpAttrMegaflowStats,
					Data: []byte{0xff}, // Only 1 byte, but sizeofDPMegaflowStats is 32 bytes
				}})...,
			),
		}}, nil
	}))

	c, err := newTestClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	_, err = c.Datapath.List()
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientDatapathListDebug(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Return some test data to see what we get
		return []genetlink.Message{
			{
				Data: append(
					// ovsh.Header (4 bytes).
					[]byte{0x01, 0x02, 0x03, 0x04},
					// netlink attributes.
					mustMarshalAttributes([]netlink.Attribute{
						{
							Type: ovsh.DpAttrName,
							Data: nlenc.Bytes("test-datapath"),
						},
						{
							Type: ovsh.DpAttrUserFeatures,
							Data: nlenc.Uint32Bytes(0x03), // Some features
						},
					})...,
				),
			},
		}, nil
	}))

	c, err := newTestClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	dps, err := c.Datapath.List()
	if err != nil {
		t.Fatalf("failed to list datapaths: %v", err)
	}

	t.Logf("🔍 DEBUG: Got %d datapaths", len(dps))
	for i, dp := range dps {
		t.Logf("🔍 DEBUG: Datapath[%d]: Index=%d, Name=%q, Features=%s", i, dp.Index, dp.Name, dp.Features)
		t.Logf("🔍 DEBUG: Stats: Hit=%d, Missed=%d, Lost=%d, Flows=%d", dp.Stats.Hit, dp.Stats.Missed, dp.Stats.Lost, dp.Stats.Flows)
		t.Logf("🔍 DEBUG: MegaflowStats: MaskHits=%d, Masks=%d", dp.MegaflowStats.MaskHits, dp.MegaflowStats.Masks)
	}
}

func TestClientDatapathListOK(t *testing.T) {
	system := Datapath{
		Name:     "ovs-system",
		Index:    1,
		Features: DatapathFeaturesUnaligned | DatapathFeaturesVPortPIDs,
		Stats: DatapathStats{
			Hit:    10,
			Missed: 20,
			Lost:   1,
			Flows:  30,
		},
		MegaflowStats: DatapathMegaflowStats{
			MaskHits: 10,
			Masks:    20,
		},
	}

	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Ensure we are querying the "ovs_datapath" family with the
		// correct parameters.
		if diff := cmp.Diff(ovsh.DpCmdGet, int(greq.Header.Command)); diff != "" {
			t.Fatalf("unexpected generic netlink command (-want +got):\n%s", diff)
		}

		h, err := parseHeader(greq.Data)
		if err != nil {
			t.Fatalf("failed to parse OvS generic netlink header: %v", err)
		}

		if diff := cmp.Diff(0, int(h.Ifindex)); diff != "" {
			t.Fatalf("unexpected datapath ID (-want +got):\n%s", diff)
		}

		return []genetlink.Message{
			{
				Data: mustMarshalDatapath(system),
			},
		}, nil
	}))

	c, err := newTestClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	dps, err := c.Datapath.List()
	if err != nil {
		t.Fatalf("failed to list datapaths: %v", err)
	}

	// Debug: Print what we actually got
	t.Logf("🔍 DEBUG: Got %d datapaths", len(dps))
	for i, dp := range dps {
		t.Logf("🔍 DEBUG: Datapath[%d]: Index=%d, Name=%q, Features=%s", i, dp.Index, dp.Name, dp.Features)
		t.Logf("🔍 DEBUG: Stats: Hit=%d, Missed=%d, Lost=%d, Flows=%d", dp.Stats.Hit, dp.Stats.Missed, dp.Stats.Lost, dp.Stats.Flows)
		t.Logf("🔍 DEBUG: MegaflowStats: MaskHits=%d, Masks=%d", dp.MegaflowStats.MaskHits, dp.MegaflowStats.Masks)
	}

	if diff := cmp.Diff(1, len(dps)); diff != "" {
		t.Fatalf("unexpected number of datapaths (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(system, dps[0]); diff != "" {
		t.Fatalf("unexpected ovs-system datapath (-want +got):\n%s", diff)
	}
}

func mustMarshalDatapath(dp Datapath) []byte {
	h := ovsh.Header{
		Ifindex: int32(dp.Index),
	}

	hb := headerBytes(h)

	s := ovsh.DPStats{
		Hit:    dp.Stats.Hit,
		Missed: dp.Stats.Missed,
		Lost:   dp.Stats.Lost,
		Flows:  dp.Stats.Flows,
	}

	sb := *(*[sizeofDPStats]byte)(unsafe.Pointer(&s))

	ms := ovsh.DPMegaflowStats{
		Mask_hit: dp.MegaflowStats.MaskHits,
		Masks:    dp.MegaflowStats.Masks,
		// Pad already set to zero.
	}

	msb := *(*[sizeofDPMegaflowStats]byte)(unsafe.Pointer(&ms))

	ab := mustMarshalAttributes([]netlink.Attribute{
		{
			Type: ovsh.DpAttrName,
			Data: nlenc.Bytes(dp.Name),
		},
		{
			Type: ovsh.DpAttrUserFeatures,
			Data: nlenc.Uint32Bytes(uint32(dp.Features)),
		},
		{
			Type: ovsh.DpAttrStats,
			Data: sb[:],
		},
		{
			Type: ovsh.DpAttrMegaflowStats,
			Data: msb[:],
		},
	})

	return append(hb[:], ab...)
}

// ovsFamilies creates a test handler that returns OVS family messages
func ovsFamilies(handler func(genetlink.Message, netlink.Message) ([]genetlink.Message, error)) func(genetlink.Message, netlink.Message) ([]genetlink.Message, error) {
	return func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Handle family listing requests
		if greq.Header.Command == unix.CTRL_CMD_GETFAMILY {
			return familyMessages([]string{
				ovsh.DatapathFamily,
			}), nil
		}

		// Handle actual datapath requests
		fmt.Printf("🔍 ovsFamilies: calling handler for command %d\n", greq.Header.Command)
		return handler(greq, nreq)
	}
}
