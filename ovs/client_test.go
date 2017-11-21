package ovs

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	var tests = []struct {
		desc    string
		options []OptionFunc
		c       *Client
	}{
		{
			desc: "no options",
			c: &Client{
				flags: make([]string, 0),
				debug: false,
			},
		},
		{
			desc:    "Timeout(2)",
			options: []OptionFunc{Timeout(2)},
			c: &Client{
				flags: []string{"--timeout=2"},
				debug: false,
			},
		},
		{
			desc:    "Debug(true)",
			options: []OptionFunc{Debug(true)},
			c: &Client{
				flags: make([]string, 0),
				debug: true,
			},
		},
		{
			desc:    "FlowFormat(FlowFormatNXMTableID)",
			options: []OptionFunc{FlowFormat(FlowFormatNXMTableID)},
			c: &Client{
				flags:      make([]string, 0),
				ofctlFlags: []string{"--flow-format=NXM+table_id"},
			},
		},
		{
			desc:    "Protocols([]string{ProtocolOpenFlow14})",
			options: []OptionFunc{Protocols([]string{ProtocolOpenFlow14})},
			c: &Client{
				flags:      make([]string, 0),
				ofctlFlags: []string{"--protocols=OpenFlow14"},
			},
		},
		{
			desc: "Timeout(5), Debug(true)",
			options: []OptionFunc{
				Timeout(5),
				Debug(true),
			},
			c: &Client{
				flags: []string{"--timeout=5"},
				debug: true,
			},
		},
		{
			desc: "Sudo()",
			options: []OptionFunc{
				Sudo(),
			},
			c: &Client{
				flags: make([]string, 0),
				sudo:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			c := New(tt.options...)

			if want, got := tt.c.flags, c.flags; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Client.flags:\n- want: %v\n-  got: %v",
					want, got)
			}

			if want, got := tt.c.debug, c.debug; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Client.debug:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func Test_shellPipe(t *testing.T) {
	b := bytes.TrimSpace([]byte(`
foo
bar
baz
`))

	// stdin pipe must be consumed.  This test will hang if broken.
	buf := bytes.NewBuffer(b)
	out, err := shellPipe(buf, "cat", "-")
	if err != nil {
		t.Fatalf("failed to pipe to cat: %v", err)
	}

	if want, got := b, out; !bytes.Equal(want, got) {
		t.Fatalf("unexpected bytes:\n- want: %v\n-  got: %v",
			want, got)
	}
}

// testClient creates a new Client with the specified OptionFuncs applied and
// using the specified ExecFunc.
func testClient(options []OptionFunc, fn ExecFunc) *Client {
	options = append(options, Exec(fn))
	c := New(options...)
	return c
}
