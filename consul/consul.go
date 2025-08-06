package consul

import (
	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/interfaces"
)

// Consul struct is used for consul client such as the client
// and the actual key data.
type Consul struct {
	Client     interfaces.ConsulClient
	KeyData    consulapi.KVPairs
	KeyDataLen int
	PQData     []*consulapi.PreparedQueryDefinition
	PQDataLen  int
	ACLData    []*consulapi.ACLEntry
	ACLDataLen int
}

// NewConsul creates a consul instance with the given client
func NewConsul(client interfaces.ConsulClient) *Consul {
	return &Consul{
		Client: client,
	}
}

// Client creates a consul client from the consul api (legacy function)
func Client() *consulapi.Client {
	consul, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		panic(err) // Changed from log.Fatalf to panic for easier testing
	}
	return consul
}

// ListKeys lists all the keys from consul with no prefix.
func (c *Consul) ListKeys() error {
	keys, err := c.Client.ListKeys()
	if err != nil {
		return err
	}
	c.KeyData = keys
	c.KeyDataLen = len(keys)
	return nil
}

// ListPQs lists all the prepared queries from consul
func (c *Consul) ListPQs() error {
	pqs, err := c.Client.ListPQs()
	if err != nil {
		return err
	}
	c.PQData = pqs
	c.PQDataLen = len(pqs)
	return nil
}

// ListACLs lists all the ACLs from consul
func (c *Consul) ListACLs() error {
	acls, err := c.Client.ListACLs()
	if err != nil {
		return err
	}
	c.ACLData = acls
	c.ACLDataLen = len(acls)
	return nil
}

// RestoreKeys restores keys to consul
func (c *Consul) RestoreKeys(keys consulapi.KVPairs) error {
	for _, kv := range keys {
		if err := c.Client.PutKV(kv.Key, kv.Value); err != nil {
			return err
		}
	}
	return nil
}

// RestorePQs restores prepared queries to consul
func (c *Consul) RestorePQs(pqs []*consulapi.PreparedQueryDefinition) error {
	for _, pq := range pqs {
		if err := c.Client.CreatePQ(pq); err != nil {
			return err
		}
	}
	return nil
}

// RestoreACLs restores ACLs to consul
func (c *Consul) RestoreACLs(acls []*consulapi.ACLEntry) error {
	for _, acl := range acls {
		if err := c.Client.CreateACL(acl); err != nil {
			return err
		}
	}
	return nil
}
