package ovs

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
)

// parseMatch creates a Match function from the input string.
func parseMatch(key string, value string) (Match, error) {
	switch key {
	case arpSHA, arpTHA, ndSLL, ndTLL:
		return parseMACMatch(key, value)
	case icmpType, nwProto:
		return parseIntMatch(key, value, math.MaxUint8)
	case tpSRC, tpDST, ctZone:
		return parseIntMatch(key, value, math.MaxUint16)
	case arpSPA:
		return ARPSourceProtocolAddress(value), nil
	case arpTPA:
		return ARPTargetProtocolAddress(value), nil
	case ctState:
		return parseCTState(value)
	case tcpFlags:
		return parseTCPFlags(value)
	case dlSRC:
		return DataLinkSource(value), nil
	case dlDST:
		return DataLinkDestination(value), nil
	case dlType:
		etherType, err := parseHexUint16(value)
		if err != nil {
			return nil, err
		}

		return DataLinkType(etherType), nil
	case dlVLAN:
		return parseDataLinkVLAN(value)
	case ndTarget:
		return NeighborDiscoveryTarget(value), nil
	case ipv6SRC:
		return IPv6Source(value), nil
	case ipv6DST:
		return IPv6Destination(value), nil
	case nwSRC:
		return NetworkSource(value), nil
	case nwDST:
		return NetworkDestination(value), nil
	case vlanTCI:
		return parseVLANTCI(value)
	case ctMark:
		return parseCTMark(value)
	}

	return nil, fmt.Errorf("no action matched for %s=%s", key, value)
}

// parseClampInt calls strconv.Atoi on s, and then ensures that s is less than
// or equal to the integer specified by max.
func parseClampInt(s string, max int) (int, error) {
	t, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if t > max {
		return 0, fmt.Errorf("integer %d too large; %d > %d", t, t, max)
	}

	return t, nil
}

// parseIntMatch parses an integer Match value from the input key and value,
// with a maximum possible value of max.
func parseIntMatch(key string, value string, max int) (Match, error) {
	t, err := parseClampInt(value, max)
	if err != nil {
		return nil, err
	}

	switch key {
	case icmpType:
		return ICMPType(uint8(t)), nil
	case nwProto:
		return NetworkProtocol(uint8(t)), nil
	case tpSRC:
		return TransportSourcePort(uint16(t)), nil
	case tpDST:
		return TransportDestinationPort(uint16(t)), nil
	case ctZone:
		return ConnectionTrackingZone(uint16(t)), nil
	}

	return nil, fmt.Errorf("no action matched for %s=%s", key, value)
}

// parseMACMatch parses a MAC address Match value from the input key and value.
func parseMACMatch(key string, value string) (Match, error) {
	mac, err := net.ParseMAC(value)
	if err != nil {
		return nil, err
	}

	switch key {
	case arpSHA:
		return ARPSourceHardwareAddress(mac), nil
	case arpTHA:
		return ARPTargetHardwareAddress(mac), nil
	case ndSLL:
		return NeighborDiscoverySourceLinkLayer(mac), nil
	case ndTLL:
		return NeighborDiscoveryTargetLinkLayer(mac), nil
	}

	return nil, fmt.Errorf("no action matched for %s=%s", key, value)
}

// parseCTState parses a series of connection tracking values into a Match.
func parseCTState(value string) (Match, error) {
	if len(value)%4 != 0 {
		return nil, errors.New("ct_state length must be divisible by 4")
	}

	var buf bytes.Buffer
	var states []string

	for i, r := range value {
		if i != 0 && i%4 == 0 {
			states = append(states, buf.String())
			buf.Reset()
		}

		_, _ = buf.WriteRune(r)
	}
	states = append(states, buf.String())

	return ConnectionTrackingState(states...), nil
}

// parseTCPFlags parses a series of TCP flags into a Match.  Open vSwitch's representation
// of These TCP flags are outlined in the ovs-field(7) man page,
func parseTCPFlags(value string) (Match, error) {
	if len(value)%4 != 0 {
		return nil, errors.New("tcp_flags length must be divisible by 4")
	}

	var buf bytes.Buffer
	var flags []string

	for i, r := range value {
		if i != 0 && i%4 == 0 {
			flags = append(flags, buf.String())
			buf.Reset()
		}

		_, _ = buf.WriteRune(r)
	}
	flags = append(flags, buf.String())

	return TCPFlags(flags...), nil
}

// hexPrefix denotes that a string integer is in hex format.
const hexPrefix = "0x"

// parseDataLinkVLAN parses a DataLinkVLAN Match from value.
func parseDataLinkVLAN(value string) (Match, error) {
	if !strings.HasPrefix(value, hexPrefix) {
		vlan, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}

		return DataLinkVLAN(vlan), nil
	}

	vlan, err := parseHexUint16(value)
	if err != nil {
		return nil, err
	}

	return DataLinkVLAN(int(vlan)), nil
}

// parseVLANTCI parses a VLANTCI Match from value.
func parseVLANTCI(value string) (Match, error) {
	var values []uint16
	for _, s := range strings.Split(value, "/") {
		if !strings.HasPrefix(s, hexPrefix) {
			v, err := strconv.Atoi(s)
			if err != nil {
				return nil, err
			}

			values = append(values, uint16(v))
			continue
		}

		v, err := parseHexUint16(s)
		if err != nil {
			return nil, err
		}

		values = append(values, v)
	}

	switch len(values) {
	case 1:
		return VLANTCI(values[0], 0), nil
	case 2:
		return VLANTCI(values[0], values[1]), nil
	// Match had too many parts, e.g. "vlan_tci=10/10/10"
	default:
		return nil, fmt.Errorf("invalid vlan_tci match: %q", value)
	}
}

// parseCTMark parses a CTMark Match from value.
func parseCTMark(value string) (Match, error) {
	var values []uint32
	for _, s := range strings.Split(value, "/") {
		if !strings.HasPrefix(s, hexPrefix) {
			v, err := strconv.Atoi(s)
			if err != nil {
				return nil, err
			}

			values = append(values, uint32(v))
			continue
		}

		v, err := parseHexUint32(s)
		if err != nil {
			return nil, err
		}

		values = append(values, v)
	}

	switch len(values) {
	case 1:
		return ConnectionTrackingMark(values[0], 0), nil
	case 2:
		return ConnectionTrackingMark(values[0], values[1]), nil
	// Match had too many parts, e.g. "ct_mark=10/10/10"
	default:
		return nil, fmt.Errorf("invalid ct_mark match: %q", value)
	}
}

// pareHexUint16 parses a uint16 value from a hexadecimal string.
func parseHexUint16(value string) (uint16, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(value, hexPrefix))
	if err != nil {
		return 0, err
	}
	if len(b) != 2 {
		return 0, errors.New("hexadecimal value must be two bytes in length")
	}

	return binary.BigEndian.Uint16(b), nil
}

// pareHexUint32 parses a uint32 value from a hexadecimal string.
func parseHexUint32(value string) (uint32, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(value, hexPrefix))
	if err != nil {
		return 0, err
	}
	if len(b) != 4 {
		return 0, errors.New("hexadecimal value must be four bytes in length")
	}

	return binary.BigEndian.Uint32(b), nil
}
