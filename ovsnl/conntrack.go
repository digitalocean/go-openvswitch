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
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/digitalocean/go-openvswitch/ovsnl/internal/ovsh"
	"github.com/ti-mo/conntrack"
	"golang.org/x/sys/unix"
)

// ConntrackEntry represents a single connection tracking entry from the kernel.
// This struct remains consistent with what your exporter expects.
type ConntrackEntry struct {
	Protocol   string // "tcp", "udp", "icmp" etc.
	OrigSrc    net.IP
	OrigDst    net.IP
	OrigSPort  uint16
	OrigDPort  uint16
	ReplySrc   net.IP
	ReplyDst   net.IP
	ReplySPort uint16
	ReplyDPort uint16
	Zone       uint16
	Mark       uint32
	State      string
}

// ConntrackService manages the connection to the kernel's conntrack via Netlink.
type ConntrackService struct {
	client *conntrack.Conn // The client from github.com/ti-mo/conntrack
}

// NewConntrackService creates a new ConntrackService.
// This function establishes the Netlink connection to the kernel's conntrack subsystem.
func NewConntrackService() (*ConntrackService, error) {
	// Try to establish netlink connection with retries
	nfct, err := conntrack.Dial(nil)
	if err != nil {
		switch {
		case os.IsPermission(err):
			return nil, fmt.Errorf("permission denied accessing netlink socket (are you root?): %w", err)
		case errors.Is(err, syscall.ENOENT):
			return nil, fmt.Errorf("conntrack module not loaded in kernel: %w", err)
		case errors.Is(err, syscall.EPROTONOSUPPORT):
			return nil, fmt.Errorf("netlink protocol not supported: %w", err)
		case errors.Is(err, syscall.ENOMEM):
			return nil, fmt.Errorf("kernel failed to allocate memory for netlink: %w", err)
		default:
			return nil, fmt.Errorf("failed to dial conntrack netlink: %w", err)
		}
	}

	// Verify connection is working by attempting a statistics query
	_, err = nfct.Stats()
	if err != nil {
		nfct.Close()
		return nil, fmt.Errorf("failed to verify netlink connection: %w", err)
	}

	return &ConntrackService{
		client: nfct,
	}, nil
}

// Close closes the underlying Netlink connection for conntrack.
func (s *ConntrackService) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// List lists all conntrack entries from the kernel.
// datapathName is not used in this direct Netlink query, as it's a global dump.
// List lists all conntrack entries from the kernel.
func (s *ConntrackService) List(ctx context.Context) ([]ConntrackEntry, error) {
	// Add context support
	if ctx == nil {
		ctx = context.Background()
	}

	// Create a channel for the dump operation
	flowChan := make(chan conntrack.Flow)
	errChan := make(chan error, 1)

	// Start dump in goroutine
	go func() {
		defer close(flowChan)
		flows, err := s.client.Dump(nil)
		if err != nil {
			errChan <- fmt.Errorf("failed to dump conntrack entries: %w", err)
			return
		}
		for _, f := range flows {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case flowChan <- f:
			}
		}
	}()

	var entries []ConntrackEntry
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		case f, ok := <-flowChan:
			if !ok {
				return entries, nil
			}
			entry, err := s.convertFlow(f)
			if err != nil {
				return nil, fmt.Errorf("failed to convert flow: %w", err)
			}
			entries = append(entries, entry)
		}
	}
}

// Helper function to map protocol numbers to names
func getProtocolString(protoNum uint8) string {
	switch protoNum {
	case unix.IPPROTO_TCP:
		return "tcp"
	case unix.IPPROTO_UDP:
		return "udp"
	case unix.IPPROTO_ICMP:
		return "icmp"
	case unix.IPPROTO_ICMPV6:
		return "icmpv6"
	default:
		return fmt.Sprintf("proto_%d", protoNum)
	}
}

// parseConntrackStateFlags converts kernel conntrack state bitmask to readable string.
// (Uses ovsh/const.go flags like CsFNew, CsFEstablished etc.)
func parseConntrackStateFlags(flags uint32) string {
	states := []string{}
	// Mapping from ovsh/const.go (CsFNew, CsFEstablished, etc.)
	// Ensure ovsh is correctly imported and these flags are available.
	if flags&ovsh.CsFNew != 0 {
		states = append(states, "NEW")
	}
	if flags&ovsh.CsFEstablished != 0 {
		states = append(states, "ESTABLISHED")
	}
	if flags&ovsh.CsFRelated != 0 {
		states = append(states, "RELATED")
	}
	if flags&ovsh.CsFReplyDir != 0 {
		states = append(states, "REPLY")
	}
	if flags&ovsh.CsFInvalid != 0 {
		states = append(states, "INVALID")
	}
	if flags&ovsh.CsFTracked != 0 {
		states = append(states, "TRACKED")
	}
	if flags&ovsh.CsFSrcNat != 0 {
		states = append(states, "SNAT")
	}
	if flags&ovsh.CsFDstNat != 0 {
		states = append(states, "DNAT")
	}
	if len(states) == 0 {
		return "UNKNOWN"
	}
	return strings.Join(states, "|")
}

func (s *ConntrackService) convertFlow(f conntrack.Flow) (ConntrackEntry, error) {
	entry := ConntrackEntry{
		Protocol:   getProtocolString(f.TupleOrig.Proto.Protocol),
		OrigSrc:    net.IP(f.TupleOrig.IP.SourceAddress.AsSlice()),
		OrigDst:    net.IP(f.TupleOrig.IP.DestinationAddress.AsSlice()),
		ReplySrc:   net.IP(f.TupleReply.IP.SourceAddress.AsSlice()),
		ReplyDst:   net.IP(f.TupleReply.IP.DestinationAddress.AsSlice()),
		OrigSPort:  f.TupleOrig.Proto.SourcePort,
		OrigDPort:  f.TupleOrig.Proto.DestinationPort,
		ReplySPort: f.TupleReply.Proto.SourcePort,
		ReplyDPort: f.TupleReply.Proto.DestinationPort,
		Zone:       f.Zone,
		Mark:       f.Mark,
	}

	// Handle TCP state specifically
	if f.TupleOrig.Proto.Protocol == unix.IPPROTO_TCP {
		entry.State = parseConntrackStateFlags(uint32(f.ProtoInfo.TCP.State))
	}

	return entry, nil
}
