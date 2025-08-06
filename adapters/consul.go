package adapters

import (
	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/interfaces"
	"strings"
)

// ConsulAdapter implements the ConsulClient interface
type ConsulAdapter struct {
	Client *consulapi.Client
}

// NewConsulAdapter creates a new consul adapter
func NewConsulAdapter() (interfaces.ConsulClient, error) {
	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		return nil, err
	}
	return &ConsulAdapter{Client: client}, nil
}

// ListKeys lists all keys from consul
func (c *ConsulAdapter) ListKeys() (consulapi.KVPairs, error) {
	listOpt := &consulapi.QueryOptions{
		AllowStale:        false,
		RequireConsistent: true,
	}
	keys, _, err := c.Client.KV().List("/", listOpt)
	return keys, err
}

// ListPQs lists all prepared queries from consul
func (c *ConsulAdapter) ListPQs() ([]*consulapi.PreparedQueryDefinition, error) {
	listOpt := &consulapi.QueryOptions{
		AllowStale:        false,
		RequireConsistent: true,
	}
	pqs, _, err := c.Client.PreparedQuery().List(listOpt)
	return pqs, err
}

// ListACLs lists all ACLs from consul
func (c *ConsulAdapter) ListACLs() ([]*consulapi.ACLEntry, error) {
	listOpt := &consulapi.QueryOptions{
		AllowStale:        false,
		RequireConsistent: true,
	}
	
	acls, _, err := c.Client.ACL().List(listOpt)
	if err != nil {
		// Handle ACL disabled case
		if strings.Contains(err.Error(), "401 (ACL support disabled)") {
			return []*consulapi.ACLEntry{}, nil
		}
		return nil, err
	}
	return acls, nil
}

// PutKV puts a key-value pair in consul
func (c *ConsulAdapter) PutKV(key string, value []byte) error {
	p := &consulapi.KVPair{Key: key, Value: value}
	_, err := c.Client.KV().Put(p, nil)
	return err
}

// CreatePQ creates a prepared query in consul
func (c *ConsulAdapter) CreatePQ(pq *consulapi.PreparedQueryDefinition) error {
	_, _, err := c.Client.PreparedQuery().Create(pq, nil)
	return err
}

// CreateACL creates an ACL in consul
func (c *ConsulAdapter) CreateACL(acl *consulapi.ACLEntry) error {
	_, _, err := c.Client.ACL().Create(acl, nil)
	return err
}