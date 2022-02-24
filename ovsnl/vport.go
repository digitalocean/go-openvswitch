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

package ovsnl

import (
	"fmt"
	"unsafe"

	"github.com/digitalocean/go-openvswitch/ovsnl/internal/ovsh"
	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
)

// VportService provides access to methods which interact with the
// "ovs_vport" generic netlink family.
type VportService struct {
	c *Client
	f genetlink.Family
}

// VportID numbers are scoped to a particular datapath
type VportID uint32

// VportSpec vport spec
type VportSpec interface {
	TypeName() string
	Name() string
	typeID() uint32
	optionNlAttrs()
}

// VportSpecBase vport spec basic structure
type VportSpecBase struct {
	name string
}

// Name vport spec name
func (v VportSpecBase) Name() string {
	return v.name
}

// SimpleVportSpec simple vport spec
type SimpleVportSpec struct {
	VportSpecBase
	typ      uint32
	typeName string
}

// TypeName vport spec type name
func (s SimpleVportSpec) TypeName() string {
	return s.typeName
}

func (s SimpleVportSpec) typeID() uint32 {
	return s.typ
}

func (s SimpleVportSpec) optionNlAttrs() {

}

// NewNetDevVportSpec creates netdev vport spec
func NewNetDevVportSpec(name string) VportSpec {
	return SimpleVportSpec{
		VportSpecBase: VportSpecBase{name: name},
		typ:           ovsh.VportTypeNetdev,
		typeName:      "netdev",
	}
}

// NewInternalVportSepc creates internal vport spec
func NewInternalVportSepc(name string) VportSpec {
	return SimpleVportSpec{
		VportSpecBase: VportSpecBase{name: name},
		typ:           ovsh.VportTypeInternal,
		typeName:      "internal",
	}
}

// GreVportSpec GRE vport spec
type GreVportSpec struct {
	VportSpecBase
}

// TypeName returns type name
func (s GreVportSpec) TypeName() string {
	return "gre"
}

func (s GreVportSpec) typeID() uint32 {
	return ovsh.VportTypeGre
}

func (s GreVportSpec) optionNlAttrs() {

}

// NewGreVportSpec creates GRE vport spec
func NewGreVportSpec(name string) VportSpec {
	return GreVportSpec{VportSpecBase{name: name}}
}

type udpVportSpec struct {
	VportSpecBase
	Port uint16
}

func (s udpVportSpec) optionNlAttrs() {

}

// VxLanVportSpec vxlan port spec
type VxLanVportSpec struct {
	udpVportSpec
}

// TypeName returns type name
func (s VxLanVportSpec) TypeName() string {
	return "vxlan"
}

func (s VxLanVportSpec) typeID() uint32 {
	return ovsh.VportTypeVxlan
}

// NewVxLanVportSpec creates vxlan vport spec
func NewVxLanVportSpec(name string, port uint16) VportSpec {
	return VxLanVportSpec{udpVportSpec{VportSpecBase{name: name}, port}}
}

// GeneveVportSpec geneve port spec
type GeneveVportSpec struct {
	udpVportSpec
}

// TypeName returns type name
func (s GeneveVportSpec) TypeName() string {
	return "geneve"
}

func (s GeneveVportSpec) typeID() uint32 {
	return ovsh.VportTypeGeneve
}

// NewGeneveVportSpec creates geneve vport spec
func NewGeneveVportSpec(name string, port uint16) VportSpec {
	return GeneveVportSpec{udpVportSpec{VportSpecBase{name: name}, port}}
}

// Vport is an Open vSwitch in-kernel vport.
type Vport struct {
	DatapathID int // datapath id
	ID         VportID
	Spec       VportSpec
	Stats      VportStats
	IfIndex    uint32
	NetNsID    uint32
}

func (p *Vport) String() string {
	return fmt.Sprintf("port %d: %s (%s) IfIndex:%d netnsid:%d", p.ID, p.Spec.Name(), p.Spec.TypeName(),
		p.IfIndex, p.NetNsID)
}

// VportStats contatins statistics about packets that have passed through a vport.
type VportStats struct {
	// total packets received
	RxPackets uint64
	// total packets transmitted
	TxPackets uint64
	// total bytes received
	RxBytes uint64
	// total bytes transmitted
	TxBytes uint64
	// total bad packets received
	RxErrors uint64
	// total bad packets transmitted
	TxErrors uint64
	// total dropped packets, no space in linux buffers
	RxDropped uint64
	// total dropped packets, no space available in linux
	TxDropped uint64
}

func (s *VportStats) String() string {
	return fmt.Sprintf("RX packets:%d errors:%d dropped:%d\nTX packets:%d errors:%d dropped:%d\n"+
		"RX bytes:%d TX bytes:%d", s.RxPackets, s.RxErrors, s.RxDropped, s.TxPackets, s.TxErrors, s.TxDropped,
		s.RxBytes, s.TxBytes)
}

// parseVportStats parses a slice of byte into VportStats
func parseVportStats(b []byte) (VportStats, error) {
	// Verfiy that the byte slice is the correct length before doing
	// unsafe casts.
	if want, got := sizeofVportStats, len(b); want != got {
		return VportStats{}, fmt.Errorf("unexpected vport stats structure size, want %d, got %d", want, got)
	}

	s := *(*ovsh.VportStats)(unsafe.Pointer(&b[0]))
	return VportStats{
		RxPackets: s.Rx_packets,
		TxPackets: s.Tx_packets,
		RxBytes:   s.Rx_bytes,
		TxBytes:   s.Tx_bytes,
		RxErrors:  s.Rx_errors,
		TxErrors:  s.Tx_errors,
		RxDropped: s.Rx_dropped,
		TxDropped: s.Tx_dropped,
	}, nil
}

// parseVport parses a Vport from a generic netlink message
func parseVport(msg genetlink.Message) (Vport, error) {
	// Fetch the header at the beginning of the message
	h, err := parseHeader(msg.Data)
	if err != nil {
		return Vport{}, err
	}

	vport := Vport{
		DatapathID: int(h.Ifindex),
	}

	var (
		spec VportSpec
		typ  uint32
		name string
	)

	// Skip the header to parse attributes.
	attrs, err := netlink.UnmarshalAttributes(msg.Data[sizeofHeader:])
	if err != nil {
		return Vport{}, err
	}

	for _, a := range attrs {
		switch a.Type {
		case ovsh.VportAttrPortNo:
			vport.ID = VportID(nlenc.Uint32(a.Data))
		case ovsh.VportAttrType:
			typ = nlenc.Uint32(a.Data)
		case ovsh.VportAttrName:
			name = nlenc.String(a.Data)
		case ovsh.VportAttrIfindex:
			vport.IfIndex = nlenc.Uint32(a.Data)
		case ovsh.VportAttrNetnsid:
			vport.NetNsID = nlenc.Uint32(a.Data)
		case ovsh.VportAttrStats:
			vport.Stats, err = parseVportStats(a.Data)
			if err != nil {
				return Vport{}, err
			}
		}
	}

	switch typ {
	case ovsh.VportTypeNetdev:
		spec = NewNetDevVportSpec(name)
	case ovsh.VportTypeInternal:
		spec = NewInternalVportSepc(name)
	case ovsh.VportTypeGre:
		spec = NewGreVportSpec(name)
	case ovsh.VportTypeVxlan:
		spec = NewVxLanVportSpec(name, 0)
	case ovsh.VportTypeGeneve:
		spec = NewGeneveVportSpec(name, 0)
	default:
		err = fmt.Errorf("unsupported vport type %d", typ)
	}

	if err != nil {
		return Vport{}, err
	}

	vport.Spec = spec
	return vport, nil
}

// parseVports parses a slice of Vport from a slice of generic netlink messages.
func parseVports(msgs []genetlink.Message) ([]Vport, error) {
	vports := make([]Vport, 0, len(msgs))

	for _, m := range msgs {
		vport, err := parseVport(m)
		if err != nil {
			return nil, err
		}
		vports = append(vports, vport)
	}

	return vports, nil
}

// GetByID get a Vport with specified id in the kernel.
func (s *VportService) GetByID(dpID int, vportID VportID) (Vport, error) {
	req := NewNlMsgBuilder()
	req.PutGenlMsgHdr(ovsh.VportCmdGet, s.f.Version)
	req.PutOvsHeader(int32(dpID))
	if err := req.PutUint32Attr(ovsh.VportAttrPortNo, uint32(vportID)); err != nil {
		return Vport{}, err
	}

	flags := netlink.Request | netlink.Echo
	msgs, err := s.c.c.Execute(req.Message(), s.f.ID, flags)
	if err != nil {
		return Vport{}, err
	}

	if len(msgs) == 0 {
		return Vport{}, nil
	}

	return parseVport(msgs[0])
}

// GetByName get a Vport with specified name in the kernel.
func (s *VportService) GetByName(dpID int, name string) (Vport, error) {
	req := NewNlMsgBuilder()
	req.PutGenlMsgHdr(ovsh.VportCmdGet, s.f.Version)
	req.PutOvsHeader(int32(dpID))
	if err := req.PutStringAttr(ovsh.VportAttrPortNo, name); err != nil {
		return Vport{}, err
	}

	flags := netlink.Request | netlink.Echo
	msgs, err := s.c.c.Execute(req.Message(), s.f.ID, flags)
	if err != nil {
		return Vport{}, err
	}

	if len(msgs) == 0 {
		return Vport{}, nil
	}

	return parseVport(msgs[0])
}

// List lists all Vport in the kernel.
func (s *VportService) List(dpID int) ([]Vport, error) {
	req := NewNlMsgBuilder()
	req.PutGenlMsgHdr(ovsh.VportCmdGet, s.f.Version)
	req.PutOvsHeader(int32(dpID))

	flags := netlink.Request | netlink.Echo
	msgs, err := s.c.c.Execute(req.Message(), s.f.ID, flags)
	if err != nil {
		return nil, err
	}

	if len(msgs) == 0 {
		return nil, nil
	}

	return parseVports(msgs)
}
