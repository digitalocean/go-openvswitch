package ovs

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
)

// hwAddrGoString converts a net.HardwareAddr into its Go syntax representation.
func hwAddrGoString(addr net.HardwareAddr) string {
	buf := bytes.NewBufferString("net.HardwareAddr{")
	for i, b := range addr {
		_, _ = buf.WriteString(fmt.Sprintf("0x%02x", b))

		if i != len(addr)-1 {
			_, _ = buf.WriteString(", ")
		}
	}
	_, _ = buf.WriteString("}")

	return buf.String()
}

// ipv4GoString converts a net.IP (IPv4 only) into its Go syntax representation.
func ipv4GoString(ip net.IP) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return `panic("invalid IPv4 address")`
	}

	buf := bytes.NewBufferString("net.IPv4(")
	for i, b := range ip4 {
		_, _ = buf.WriteString(strconv.Itoa(int(b)))

		if i != len(ip4)-1 {
			_, _ = buf.WriteString(", ")
		}
	}
	_, _ = buf.WriteString(")")

	return buf.String()
}

// bprintf is fmt.Sprintf, but it returns a byte slice instead of a string.
func bprintf(format string, a ...interface{}) []byte {
	return []byte(fmt.Sprintf(format, a...))
}
