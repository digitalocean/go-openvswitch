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

//+build linux

package ovsnl

import (
	"fmt"
	"os"
	"testing"

	"github.com/danieldin95/go-openvswitch/ovsnl/internal/ovsh"
	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/genetlink/genltest"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

func TestClientNoFamiliesIsNotExist(t *testing.T) {
	conn := genltest.Dial(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Unrelated generic netlink families.
		return familyMessages([]string{
			"TASKSTATS",
			"nl80211",
		}), nil
	})

	_, err := newClient(conn)
	if !os.IsNotExist(err) {
		t.Fatalf("expected is not exist error, but got: %v", err)
	}

	t.Logf("OK error: %v", err)
}

func TestClientUnknownFamilies(t *testing.T) {
	conn := genltest.Dial(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		return familyMessages([]string{
			"ovs_foo",
		}), nil
	})

	_, err := newClient(conn)
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientNoFamilies(t *testing.T) {
	conn := genltest.Dial(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Too few OVS families.
		return nil, nil
	})

	_, err := newClient(conn)
	if err == nil {
		t.Fatalf("expected an error, but none occurred")
	}

	t.Logf("OK error: %v", err)
}

func TestClientKnownFamilies(t *testing.T) {
	conn := genltest.Dial(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		return familyMessages([]string{
			ovsh.DatapathFamily,
		}), nil
	})

	_, err := newClient(conn)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
}

func familyMessages(families []string) []genetlink.Message {
	msgs := make([]genetlink.Message, 0, len(families))

	var id uint16
	for _, f := range families {
		msgs = append(msgs, genetlink.Message{
			Data: mustMarshalAttributes([]netlink.Attribute{
				{
					Type: unix.CTRL_ATTR_FAMILY_ID,
					Data: nlenc.Uint16Bytes(id),
				},
				{
					Type: unix.CTRL_ATTR_FAMILY_NAME,
					Data: nlenc.Bytes(f),
				},
			}),
		})

		id++
	}

	return msgs
}

// ovsFamilies creates a genltest.Func which intercepts "list family" requests
// and returns all the OVS families.  Other requests are passed through to fn.
func ovsFamilies(fn genltest.Func) genltest.Func {
	return func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		if nreq.Header.Type == unix.GENL_ID_CTRL && greq.Header.Command == unix.CTRL_CMD_GETFAMILY {
			return familyMessages([]string{
				ovsh.DatapathFamily,
				ovsh.FlowFamily,
				ovsh.PacketFamily,
				ovsh.VportFamily,
				ovsh.MeterFamily,
			}), nil
		}

		return fn(greq, nreq)
	}
}

func mustMarshalAttributes(attrs []netlink.Attribute) []byte {
	b, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal attributes: %v", err))
	}

	return b
}
