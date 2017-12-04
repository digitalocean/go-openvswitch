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
	"encoding/binary"
	"unsafe"

	"github.com/digitalocean/go-openvswitch/ovsnl/internal/ovsh"
	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
)

// A FlowService provides access to methods which interact with the
// "ovs_flow" generic netlink family.
type FlowService struct {
	c *Client
	f genetlink.Family
}

// A Flow is an Open vSwitch in-kernel Flow.
type Flow struct {
	Keys  []FlowKey
	Stats ovsh.FlowStats
	//Masks []FlowKey
}

type FlowKey interface{}

type FlowKeyEncap []FlowKey

type FlowKeyEtherType uint16

// List lists all Flows in the kernel for the datapath specified by index.
func (s *FlowService) List(index int) ([]Flow, error) {
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: ovsh.FlowCmdGet,
			Version: uint8(s.f.Version),
		},
		// Query the specified datapath.
		Data: headerBytes(ovsh.Header{
			Ifindex: int32(index),
		}),
	}

	flags := netlink.HeaderFlagsRequest | netlink.HeaderFlagsDump
	msgs, err := s.c.c.Execute(req, s.f.ID, flags)
	if err != nil {
		return nil, err
	}

	return parseFlows(msgs)
}

// parseFlows parses a slice of Flows from a slice of generic netlink
// messages.
func parseFlows(msgs []genetlink.Message) ([]Flow, error) {
	flows := make([]Flow, 0, len(msgs))

	for _, m := range msgs {
		// Fetch the header at the beginning of the message.
		h, err := parseHeader(m.Data)
		if err != nil {
			return nil, err
		}

		_ = h

		// Skip the header to parse attributes.
		attrs, err := netlink.UnmarshalAttributes(m.Data[sizeofHeader:])
		if err != nil {
			return nil, err
		}

		var f Flow

		for _, a := range attrs {
			switch a.Type {
			case ovsh.FlowAttrKey:
				f.Keys, err = parseFlowKeys(a.Data)
				if err != nil {
					return nil, err
				}
			case ovsh.FlowAttrStats:
				s := *(*ovsh.FlowStats)(unsafe.Pointer(&a.Data[0]))
				f.Stats = s
			}
		}

		if len(f.Keys) == 0 {
			continue
		}
		if f.Stats.Bytes == 0 && f.Stats.Packets == 0 {
			continue
		}

		flows = append(flows, f)
	}

	return flows, nil
}

func parseFlowKeys(b []byte) ([]FlowKey, error) {
	attrs, err := netlink.UnmarshalAttributes(b)
	if err != nil {
		return nil, err
	}

	var keys []FlowKey

	for _, a := range attrs {
		switch a.Type {
		case ovsh.KeyAttrEthertype:
			keys = append(keys, FlowKeyEtherType(binary.BigEndian.Uint16(a.Data)))
			/*
				case ovsh.KeyAttrVlan:
					var v ethernet.VLAN
					if err := (&v).UnmarshalBinary(a.Data); err != nil {
						return nil, err
					}

					keys = append(keys, v)
				case ovsh.KeyAttrEthernet:
					eth := *(*ovsh.KeyEthernet)(unsafe.Pointer(&a.Data[0]))
					keys = append(keys, eth)
			*/
		case ovsh.KeyAttrEncap:
			encap, err := parseFlowKeys(a.Data)
			if err != nil {
				return nil, err
			}
			if len(encap) == 0 {
				continue
			}

			keys = append(keys, FlowKeyEncap(encap))
		case ovsh.KeyAttrIpv4:
			ip4 := *(*ovsh.KeyIPv4)(unsafe.Pointer(&a.Data[0]))

			if ip4.Proto == 0 {
				continue
			}

			/*
				src := *(*[4]byte)(unsafe.Pointer(&ip4.Src))
				dst := *(*[4]byte)(unsafe.Pointer(&ip4.Dst))

				log.Println(src, dst)
			*/

			keys = append(keys, IP{
				Family:   FamilyIPv4,
				Protocol: int(ip4.Proto),
			})
		case ovsh.KeyAttrIpv6:
			ip6 := *(*ovsh.KeyIPv6)(unsafe.Pointer(&a.Data[0]))

			if ip6.Proto == 0 {
				continue
			}

			/*
				src := make([]byte, 16)
				for i, part := range ip6.Src {
					start := i * 4
					binary.LittleEndian.PutUint32(src[start:start+4], part)
				}

				dst := make([]byte, 16)
				for i, part := range ip6.Dst {
					start := i * 4
					binary.LittleEndian.PutUint32(dst[start:start+4], part)
				}
			*/

			keys = append(keys, IP{
				Family:   FamilyIPv6,
				Protocol: int(ip6.Proto),
			})
		}
	}

	return keys, nil
}

type IP struct {
	Family Family
	//Source, Destination net.IP
	Protocol int
}

type Family string

const (
	FamilyIPv4 Family = "ipv4"
	FamilyIPv6 Family = "ipv6"
)
