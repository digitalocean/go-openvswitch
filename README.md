go-openvswitch [![Build Status](https://travis-ci.org/digitalocean/go-openvswitch.svg?branch=master)](https://travis-ci.org/digitalocean/go-openvswitch) [![GoDoc](https://godoc.org/github.com/digitalocean/go-openvswitch?status.svg)](https://godoc.org/github.com/digitalocean/go-openvswitch) [![Go Report Card](https://goreportcard.com/badge/github.com/digitalocean/go-openvswitch)](https://goreportcard.com/report/github.com/digitalocean/go-openvswitch)
==============

Go packages which enable interacting with Open vSwitch. Apache 2.0 Licensed.

- `ovs`: Package ovs is a client library for Open vSwitch which enables programmatic control of the virtual switch.
- `ovsnl`: Package ovsnl enables interaction with the Linux Open vSwitch generic netlink interface.

ovs
---

Not yet open source; coming soon!

ovsnl
-----

Package `ovsnl` allows you to utilize OvS's Linux generic netlink interface to
pull data from the kernel.  Here's an example:

```go
// Dial a generic netlink connection and create a *ovsnl.Client.
c, err := ovsnl.New()
if err != nil {
    // If OVS generic netlink families aren't available, do nothing.
    if os.IsNotExist(err) {
        log.Printf("generic netlink OVS families not found: %v", err)
        return
    }

	log.Fatalf("failed to create client %v", err)
}
// Be sure to close the generic netlink connection!
defer c.Close()

// TODO(mdlayher): expand upon this example!
```