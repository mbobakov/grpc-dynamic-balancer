package balancer

import (
	"google.golang.org/grpc/balancer"
)

type balancerBuilder struct {
	name    string
	service service
}

// NewBalancerBuilder returns a balancer builder. The balancers
// built by this builder will use the picker builder to build pickers.
func NewBalancerBuilder(name string, provider service) balancer.Builder {
	return &balancerBuilder{
		name:    name,
		service: provider,
	}
}

func (bb *balancerBuilder) Build(cc balancer.ClientConn, opt balancer.BuildOptions) balancer.Balancer {
	return &dynBalancer{
		serviceName: bb.name,
		cc:          cc,
		service:     bb.service,
		backend:     &inMemoryBackend{},
		done:        make(chan struct{}),
	}
}

func (bb *balancerBuilder) Name() string {
	return bb.name
}
