package consul

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
)

// Consul is a provider for a dynamic endpoint configuration
type Consul struct {
	api *api.Client
}

// Endpoints for the service. It blocks until it is don't connected or event occurs in the catalog
func (c *Consul) Endpoints(sn string, idx uint64) ([]resolver.Address, uint64, error) {
	if c.api == nil {
		return nil, 0, errors.New("Service is not ready")
	}
	ss, meta, err := c.api.Catalog().Service(sn, "", &api.QueryOptions{WaitIndex: idx, WaitTime: time.Hour})
	if err != nil {
		var lastid uint64
		if meta != nil {
			lastid = meta.LastIndex
		}
		return nil, lastid, err
	}
	var ee []resolver.Address
	for _, s := range ss {
		e := fmt.Sprintf("%s:%d", s.ServiceAddress, s.ServicePort)
		ee = append(ee, resolver.Address{Addr: e, Metadata: s.ServiceTags})
	}
	return ee, meta.LastIndex, nil
}

// Connect to the consul agent
func (c *Consul) Connect(addr string) error {
	cli, err := api.NewClient(&api.Config{Address: addr})
	if err != nil {
		return err
	}
	c.api = cli
	return nil
}
