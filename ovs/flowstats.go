package ovs

import (
	"errors"
	"strconv"
	"strings"
)

var (
	// ErrInvalidFlowStats is returned when flow statistics from 'ovs-ofctl
	// dump-aggregate' do not match the expected output format.
	ErrInvalidFlowStats = errors.New("invalid flow statistics")
)

// FlowStats contains a variety of statistics about an Open vSwitch port,
// including its port ID and numbers about packet receive and transmit
// operations.
type FlowStats struct {
	PacketCount uint64
	ByteCount   uint64
}

// UnmarshalText unmarshals a FlowStats from textual form as output by
// 'ovs-ofctl dump-aggregate <bridge> table=<tablename>,cookie=<cookie>/-1':
//    NXST_AGGREGATE reply (xid=0x4): packet_count=642800 byte_count=141379644 flow_count=3
func (f *FlowStats) UnmarshalText(b []byte) error {
	// Make a copy per documentation for encoding.TextUnmarshaler.
	s := string(b)

	// Constants only needed within this method, to avoid polluting the
	// package namespace with generic names
	const (
		nxstAggregate = "NXST_AGGREGATE"
		packetCount   = "packet_count"
		byteCount     = "byte_count"
	)

	ss := strings.Fields(s)
	// Assumption here is that string s should have 6 fields of text separated by
	// (5) spaces
	if len(ss) != 6 {
		return ErrInvalidFlowStats
	}

	if ss[0] != nxstAggregate {
		return ErrInvalidFlowStats
	}

	packetCountPair := strings.Split(ss[3], "=")
	if len(packetCountPair) != 2 || packetCountPair[0] != packetCount {
		return ErrInvalidFlowStats
	}

	n, err := strconv.ParseUint(packetCountPair[1], 10, 64)
	if err != nil {
		return ErrInvalidFlowStats
	}

	f.PacketCount = n

	byteCountPair := strings.Split(ss[4], "=")
	if len(byteCountPair) != 2 || byteCountPair[0] != byteCount {
		return ErrInvalidFlowStats
	}

	n, err = strconv.ParseUint(byteCountPair[1], 10, 64)
	if err != nil {
		return ErrInvalidFlowStats
	}

	f.ByteCount = n

	return nil
}
