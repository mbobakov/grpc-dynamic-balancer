## Client-side dynamic load balancer for the GRPC
[![Go Report Card](https://goreportcard.com/badge/github.com/mbobakov/grpc-dynamic-balancer)](https://goreportcard.com/report/github.com/mbobakov/grpc-dynamic-balancer) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Build Status](https://travis-ci.org/mbobakov/grpc-dynamic-balancer.svg?branch=master)](https://travis-ci.org/mbobakov/grpc-dynamic-balancer) [![codecov](https://codecov.io/gh/mbobakov/grpc-dynamic-balancer/branch/master/graph/badge.svg)](https://codecov.io/gh/mbobakov/grpc-dynamic-balancer)

This library support new [grpc-go/balancer](https://github.com/grpc/grpc-go/tree/master/balancer) interface.
You can pass self-implemented endpoint providers. For this you can realize [service interface](https://github.com/mbobakov/grpc-dynamic-balancer/blob/master/balancer.go#L10) and pass implementation to the `.NewBalancerBuilder` constructor. This library support consul by default.


### Example
```go
package api // which contains code generated from the proto-files

import (
	"github.com/mbobakov/grpc-dynamic-balancer"
	"github.com/mbobakov/grpc-dynamic-balancer/provider/consul"
	"google.golang.org/grpc"
	grpcbalancer "google.golang.org/grpc/balancer"
)

func NewLoadBalancedClient(consulAddr, serviceName string, opts ...grpc.DialOption) (<Client>, error) {
	bb := balancer.NewBalancerBuilder(serviceName, &consul.Consul{})
	grpcbalancer.Register(bb)
	opts = append(opts, grpc.WithBalancerName(serviceName))
	conn, err := grpc.Dial(consulAddr, opts...)
	if err != nil { <handle error> }
	return &grpcGeneratedClient{cc: conn}, nil
}
```
### Motivation
After some test load-balancing on the client-side is the most efficient way to HA.
Tests(envoy,nginx) TBD

### Contributing
Feel free to open issues, PRs and refer to the implementation of endpoints providers

