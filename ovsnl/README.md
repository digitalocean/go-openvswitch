ovsnl
=====

Package `ovsnl` enables interaction with the Linux Open vSwitch generic
netlink interface.

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