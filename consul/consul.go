package consul

import (
	"log"

	consulapi "github.com/hashicorp/consul/api"
)

// Consul struct is used for consul client such as the client
// and the actual key data.
type Consul struct {
	Client     consulapi.Client
	KeyData    consulapi.KVPairs
	KeyDataLen int
}

// Client creates a consul client from the consul api
func Client() *consulapi.Client {
	consul, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Fatalf("[ERR] Unable to create a consul client: %v", err)
	}
	return consul
}

// ListKeys lists all the keys from consul with no prefix.
func (c *Consul) ListKeys() {
	listOpt := &consulapi.QueryOptions{
		AllowStale:        false,
		RequireConsistent: true,
	}
	keys, _, err := c.Client.KV().List("/", listOpt)
	if err != nil {
		log.Fatalf("[ERR] Unable to list keys: %v", err)
	}
	c.KeyData = keys
	c.KeyDataLen = len(keys)
}
