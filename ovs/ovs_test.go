package ovs

import (
	"errors"
	"testing"
)

func TestIsPortNotExist(t *testing.T) {
	var tests = []struct {
		desc string
		err  error
		ok   bool
	}{
		{
			desc: "not type Error",
			err:  errors.New("foo"),
		},
		{
			desc: "type Error, wrong Error.Out",
			err: &Error{
				Out: []byte("bar"),
				Err: errors.New("exit status 1"),
			},
		},
		{
			desc: "type Error, wrong Error.Err",
			err: &Error{
				Out: []byte("ovs-vsctl: no port named foo"),
				Err: errors.New("exit status foo"),
			},
		},
		{
			desc: "ok",
			err: &Error{
				Out: []byte("ovs-vsctl: no port named foo"),
				Err: errors.New("exit status 1"),
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if want, got := tt.ok, IsPortNotExist(tt.err); want != got {
				t.Fatalf("unexpected IsPortNotExist(%v):\n- want: %v\n-  got: %v",
					tt.err, want, got)
			}
		})
	}
}

// errStr is a helper to return the string form of an error, even if the
// error is nil.
func errStr(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
