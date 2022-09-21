// Copyright 2022 DigitalOcean.
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
)

var (
	optUnexportSpecBase    = cmp.AllowUnexported(VportSpecBase{})
	optUnexportSimpleVport = cmp.AllowUnexported(SimpleVportSpec{})
)

func TestClientVportListOK(t *testing.T) {
	tap0 := Vport{
		DatapathID: 1,
		ID:         1,
		Spec:       NewInternalVportSepc("tap0"),
		Stats: VportStats{
			RxPackets: 100,
			TxPackets: 200,
			RxBytes:   1000,
			TxBytes:   500,
			RxDropped: 0,
			TxDropped: 5,
			RxErrors:  0,
			TxErrors:  10,
		},
		IfIndex: 10,
		NetNsID: 9,
	}

	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Ensure we are querying the "ovs_vport" family with the
		// correct parameters.
		if diff := cmp.Diff(ovsh.VportCmdGet, int(greq.Header.Command)); diff != "" {
			t.Fatalf("unexpected generic netlink command (-want +got):\n%s", diff)
		}

		h, err := parseHeader(greq.Data)
		if err != nil {
			t.Fatalf("failed to parse OvS generic netlink header: %v", err)
		}

		if diff := cmp.Diff(1, int(h.Ifindex)); diff != "" {
			t.Fatalf("unexpected datapath ID (-want +got):\n%s", diff)
		}

		return []genetlink.Message{
			{
				Data: mustMarshalVport(tap0),
			},
		}, nil
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	vports, err := c.Vport.List(1)
	if err != nil {
		t.Fatalf("failed to list vports: %v", err)
	}

	if diff := cmp.Diff(1, len(vports)); diff != "" {
		t.Fatalf("unexpected number of vports (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(tap0, vports[0], optUnexportSpecBase, optUnexportSimpleVport); diff != "" {
		t.Fatalf("unexpected tap0 vport (-want +got):\n%s", diff)
	}
}

func TestClientVportListBadStats(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Valid header; not enough data for ovsh.VportAttrStats.
		return []genetlink.Message{{
			Data: append(
				// ovsh.Header.
				[]byte{0xff, 0xff, 0xff, 0xff},
				// netlink attributes.
				mustMarshalAttributes([]netlink.Attribute{{
					Type: ovsh.VportAttrStats,
					Data: []byte{0xff},
				}})...,
			),
		}}, nil
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	_, err = c.Vport.List(1)
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientGetVportByIDOK(t *testing.T) {
	tap0 := Vport{
		DatapathID: 1,
		ID:         1,
		Spec:       NewInternalVportSepc("tap0"),
		Stats: VportStats{
			RxPackets: 100,
			TxPackets: 200,
			RxBytes:   1000,
			TxBytes:   500,
			RxDropped: 0,
			TxDropped: 5,
			RxErrors:  0,
			TxErrors:  10,
		},
		IfIndex: 10,
		NetNsID: 9,
	}

	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		return []genetlink.Message{
			{
				Data: mustMarshalVport(tap0),
			},
		}, nil
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	vport, err := c.Vport.GetByID(1, 1)
	if err != nil {
		t.Fatalf("failed to get vport: %v", err)
	}

	if diff := cmp.Diff(tap0, vport, optUnexportSpecBase, optUnexportSimpleVport); diff != "" {
		t.Fatalf("unexpected tap0 vport (-want +got):\n%s", diff)
	}
}

func TestClientGetVportByIDBad(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Valid header;
		return []genetlink.Message{{
			Data: append(
				// ovsh.Header.
				[]byte{0xff, 0xff, 0xff, 0xff},
			),
		}}, fmt.Errorf("400 bad request")
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	_, err = c.Vport.GetByID(1, 1)
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientGetVportByIDNotFound(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Valid header; not enough data for ovsh.VportAttrs.
		return []genetlink.Message{{
			Data: append(
				// ovsh.Header.
				[]byte{0xff, 0xff, 0xff, 0xff},
			),
		}}, nil
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	_, err = c.Vport.GetByID(1, 1)
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientGetVportByIDEmpty(t *testing.T) {
	tap0 := Vport{}
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Empty messages
		return []genetlink.Message{}, nil
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	vport, err := c.Vport.GetByID(1, 1)
	if err != nil {
		t.Fatalf("failed to get vport: %v", err)
	}

	if diff := cmp.Diff(tap0, vport, optUnexportSpecBase, optUnexportSimpleVport); diff != "" {
		t.Fatalf("unexpected tap0 vport (-want +got):\n%s", diff)
	}
}

func TestClientGetVportByNameOK(t *testing.T) {
	tap0 := Vport{
		DatapathID: 1,
		ID:         1,
		Spec:       NewInternalVportSepc("tap0"),
		Stats: VportStats{
			RxPackets: 100,
			TxPackets: 200,
			RxBytes:   1000,
			TxBytes:   500,
			RxDropped: 0,
			TxDropped: 5,
			RxErrors:  0,
			TxErrors:  10,
		},
		IfIndex: 10,
		NetNsID: 9,
	}

	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		return []genetlink.Message{
			{
				Data: mustMarshalVport(tap0),
			},
		}, nil
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	vport, err := c.Vport.GetByName(1, "tap0")
	if err != nil {
		t.Fatalf("failed to get vport: %v", err)
	}

	if diff := cmp.Diff(tap0, vport, optUnexportSpecBase, optUnexportSimpleVport); diff != "" {
		t.Fatalf("unexpected tap0 vport (-want +got):\n%s", diff)
	}
}

func TestClientGetVportByNameBad(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Valid header;
		return []genetlink.Message{{
			Data: append(
				// ovsh.Header.
				[]byte{0xff, 0xff, 0xff, 0xff},
			),
		}}, fmt.Errorf("400 bad request")
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	_, err = c.Vport.GetByName(1, "")
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientGetVportByNameNotFound(t *testing.T) {
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Valid header; not enough data for ovsh.VportAttrs.
		return []genetlink.Message{{
			Data: append(
				// ovsh.Header.
				[]byte{0xff, 0xff, 0xff, 0xff},
			),
		}}, nil
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	_, err = c.Vport.GetByName(1, "")
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientGetVportByNameEmpty(t *testing.T) {
	tap0 := Vport{}
	conn := genltest.Dial(ovsFamilies(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Empty messages
		return []genetlink.Message{}, nil
	}))

	c, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	vport, err := c.Vport.GetByName(1, "")
	if err != nil {
		t.Fatalf("failed to get vport: %v", err)
	}

	if diff := cmp.Diff(tap0, vport, optUnexportSpecBase, optUnexportSimpleVport); diff != "" {
		t.Fatalf("unexpected tap0 vport (-want +got):\n%s", diff)
	}
}

func mustMarshalVport(p Vport) []byte {
	h := ovsh.Header{
		Ifindex: int32(p.DatapathID),
	}

	hb := headerBytes(h)

	s := ovsh.VportStats{
		Rx_packets: p.Stats.RxPackets,
		Tx_packets: p.Stats.TxPackets,
		Rx_bytes:   p.Stats.RxBytes,
		Tx_bytes:   p.Stats.TxBytes,
		Rx_errors:  p.Stats.RxErrors,
		Tx_errors:  p.Stats.TxErrors,
		Rx_dropped: p.Stats.RxDropped,
		Tx_dropped: p.Stats.TxDropped,
	}

	sb := *(*[sizeofVportStats]byte)(unsafe.Pointer(&s))

	ab := mustMarshalAttributes([]netlink.Attribute{
		{
			Type: ovsh.VportAttrPortNo,
			Data: nlenc.Uint32Bytes(uint32(p.ID)),
		},
		{
			Type: ovsh.VportAttrType,
			Data: nlenc.Uint32Bytes(p.Spec.typeID()),
		},
		{
			Type: ovsh.VportAttrName,
			Data: nlenc.Bytes(p.Spec.Name()),
		},
		{
			Type: ovsh.VportAttrStats,
			Data: sb[:],
		},
		{
			Type: ovsh.VportAttrIfindex,
			Data: nlenc.Uint32Bytes(p.IfIndex),
		},
		{
			Type: ovsh.VportAttrNetnsid,
			Data: nlenc.Uint32Bytes(p.NetNsID),
		},
	})

	return append(hb[:], ab...)
}
