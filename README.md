go-openvswitch [![Build Status](https://travis-ci.org/digitalocean/go-openvswitch.svg?branch=master)](https://travis-ci.org/digitalocean/go-openvswitch) [![GoDoc](https://godoc.org/github.com/digitalocean/go-openvswitch?status.svg)](https://godoc.org/github.com/digitalocean/go-openvswitch) [![Go Report Card](https://goreportcard.com/badge/github.com/digitalocean/go-openvswitch)](https://goreportcard.com/report/github.com/digitalocean/go-openvswitch)
==============

Go packages which enable interacting with Open vSwitch and related tools. Apache 2.0 Licensed.

- `ovs`: Package ovs is a client library for Open vSwitch which enables programmatic control of the virtual switch.
- `ovsdb`: Package ovsdb implements an OVSDB client, as described in RFC 7047.
- `ovsnl`: Package ovsnl enables interaction with the Linux Open vSwitch generic netlink interface.

ovs
---

Package `ovs` is a wrapper around the `ovs-vsctl` and `ovs-ofctl` utilities, but
in the future, it may speak OVSDB and OpenFlow directly with the same interface.

```go
// Create a *ovs.Client.  Specify ovs.OptionFuncs to customize it.
c := ovs.New(
    // Prepend "sudo" to all commands.
    ovs.Sudo(),
)

// $ sudo ovs-vsctl --may-exist add-br ovsbr0
if err := c.VSwitch.AddBridge("ovsbr0"); err != nil {
    log.Fatalf("failed to add bridge: %v", err)
}

// $ sudo ovs-ofctl add-flow ovsbr0 priority=100,ip,actions=drop
err := c.OpenFlow.AddFlow("ovsbr0", &ovs.Flow{
    Priority: 100,
    Protocol: ovs.ProtocolIPv4,
    Actions:  []ovs.Action{ovs.Drop()},
})
if err != nil {
    log.Fatalf("failed to add flow: %v", err)
}
```

ovsdb
-----

Package `ovsdb` allows you to communicate with an instance of `ovsdb-server` using
the OVSDB protocol, specified in [RFC 7047](https://tools.ietf.org/html/rfc7047).

```go
// Dial an OVSDB connection and create a *ovsdb.Client.
c, err := ovsdb.Dial("unix", "/var/run/openvswitch/db.sock")
if err != nil {
	log.Fatalf("failed to dial: %v", err)
}
// Be sure to close the connection!
defer c.Close()

// Ask ovsdb-server for all of its databases.
dbs, err := c.ListDatabases()
if err != nil {
	log.Fatalf("failed to list databases: %v", err)
}

for _, d := range dbs {
	log.Println(d)
}
```

ovsnl
-----

Package `ovsnl` allows you to utilize OvS's Linux generic netlink interface to
pull data from the kernel.

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

// List available OVS datapaths.
dps, err := c.Datapath.List()
if err != nil {
	log.Fatalf("failed to list datapaths: %v", err)
}

for _, d := range dps {
	log.Printf("datapath: %q, flows: %d", d.Name, d.Stats.Flows)
}
```