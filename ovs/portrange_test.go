package ovs

import (
	"reflect"
	"testing"
)

func TestPortRangeBitwiseMatch(t *testing.T) {
	var tests = []struct {
		desc string
		p    *PortRange
		b    []BitRange
		err  error
	}{
		{
			desc: "empty",
			p:    &PortRange{},
			err:  ErrInvalidPortRange,
		},
		{
			desc: "no start",
			p:    &PortRange{End: 4000},
			err:  ErrInvalidPortRange,
		},
		{
			desc: "no end",
			p:    &PortRange{Start: 4000},
			err:  ErrInvalidPortRange,
		},
		{
			desc: "reversed range",
			p: &PortRange{
				Start: 5000,
				End:   4000,
			},
			err: ErrInvalidPortRange,
		},
		{
			desc: "ports 16-16",
			p: &PortRange{
				Start: 16,
				End:   16,
			},
			b: []BitRange{
				{Value: 0x10, Mask: 0xffff},
			},
			err: nil,
		},
		{
			desc: "ports 15-16 (cross boundary)",
			p: &PortRange{
				Start: 15,
				End:   16,
			},
			b: []BitRange{
				{Value: 0x0f, Mask: 0xffff},
				{Value: 0x10, Mask: 0xffff},
			},
			err: nil,
		},
		{
			desc: "ports 16-17 (binary boundary)",
			p: &PortRange{
				Start: 16,
				End:   17,
			},
			b: []BitRange{
				{Value: 0x10, Mask: 0xfffe},
			},
			err: nil,
		},
		{
			desc: "ports 16-31",
			p: &PortRange{
				Start: 16,
				End:   31,
			},
			b: []BitRange{
				{Value: 0x10, Mask: 0xfff0},
			},
			err: nil,
		},
		{
			desc: "ports 16-32",
			p: &PortRange{
				Start: 16,
				End:   32,
			},
			b: []BitRange{
				{Value: 0x10, Mask: 0xfff0},
				{Value: 0x20, Mask: 0xffff},
			},
			err: nil,
		},
		{
			desc: "ports 1000-1999",
			p: &PortRange{
				Start: 1000,
				End:   1999,
			},
			b: []BitRange{
				{Value: 0x03e8, Mask: 0xfff8},
				{Value: 0x03f0, Mask: 0xfff0},
				{Value: 0x0400, Mask: 0xfe00},
				{Value: 0x0600, Mask: 0xff00},
				{Value: 0x0700, Mask: 0xff80},
				{Value: 0x0780, Mask: 0xffc0},
				{Value: 0x07c0, Mask: 0xfff0},
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			b, err := tt.p.BitwiseMatch()
			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want %v\n- got %v",
					want, got)
			}

			if want, got := tt.b, b; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected bit range:\n- want %v\n- got: %v",
					want, got)
			}
		})
	}
}
