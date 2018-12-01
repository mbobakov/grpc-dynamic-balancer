package balancer

import (
	"time"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
)

//go:generate charlatan -package charlatan -output test/charlatan/service.go service
type service interface {
	Endpoints(string, uint64) ([]resolver.Address, uint64, error)
	Connect(addr string) error
}

//go:generate charlatan -package charlatan -output test/charlatan/backend.go backend
//go:generate charlatan -package charlatan -output test/charlatan/subconn.go balancer.SubConn
//go:generate charlatan -package charlatan -output test/charlatan/picker.go balancer.Picker
//go:generate charlatan -package charlatan -output test/charlatan/picker.go balancer.ClientConn
type backend interface {
	SetSubConnState(balancer.SubConn, connectivity.State)
	SubConnState(balancer.SubConn) connectivity.State
	Delete(balancer.SubConn)
	Picker() balancer.Picker
	AggregatedState() connectivity.State
}

type dynBalancer struct {
	serviceName string
	cc          balancer.ClientConn
	service     service
	backend     backend
	done        chan struct{}
}

// Initialize picker to a picker that always return
// ErrNoSubConnAvailable, because when state of a SubConn changes, we
// may call UpdateBalancerState with this picker.
// picker: base.NewErrPicker(balancer.ErrNoSubConnAvailable),

func (b *dynBalancer) HandleResolvedAddrs(addrs []resolver.Address, err error) {
	if err != nil {
		grpclog.Infof("dynBalancer[%s]: HandleResolvedAddrs called with error %v", b.serviceName, err)
		return
	}
	if len(addrs) == 0 {
		grpclog.Errorf("dynBalancer[%s]: HandleResolvedAddrs called with no addresses", b.serviceName)
		return
	}
	//start watching
	defer func() {
		select {
		case <-b.done:
		default:
			go b.Watch()
		}
	}()

	err = b.service.Connect(addrs[0].Addr)
	if err != nil {
		grpclog.Errorf("dynBalancer[%s]: HandleResolvedAddrs failed to connect to the service:  %v", b.serviceName, err)
		return
	}
	// Bootstrap endpoints
	ee, _, err := b.service.Endpoints(b.serviceName, 0)
	if err != nil {
		grpclog.Errorf("dynBalancer[%s]: HandleResolvedAddrs dyn provider error: %v", b.serviceName, err)
		return
	}
	if len(ee) == 0 {
		grpclog.Error("dynBalancer[%s]: HandleResolvedAddrs dyn provider has no endpoints. Waiting...", b.serviceName)
	}
	for _, e := range ee {
		sc, err := b.cc.NewSubConn([]resolver.Address{e}, balancer.NewSubConnOptions{})
		if err != nil {
			grpclog.Errorf("dynBalancer[%s]: HandleResolvedAddrs:  failed to open SubConn('%s'): %v", b.serviceName, e.Addr, err)
			continue
		}
		b.backend.SetSubConnState(sc, connectivity.Idle)
		sc.Connect()
	}
}

func (b *dynBalancer) HandleSubConnStateChange(sc balancer.SubConn, s connectivity.State) {
	grpclog.Infof("dynBalancer[%s]: HandleSubConnStateChange to %s", b.serviceName, s.String())
	prevState := b.backend.SubConnState(sc)
	b.backend.SetSubConnState(sc, s)
	if s == connectivity.TransientFailure && prevState != connectivity.Connecting ||
		s == connectivity.Shutdown {
		b.backend.Delete(sc)
		b.cc.RemoveSubConn(sc)
	}
	b.cc.UpdateBalancerState(b.backend.AggregatedState(), b.backend.Picker())
}

func (b *dynBalancer) Close() {
	close(b.done)
}

func (b *dynBalancer) Watch() {
	eeCh := make(chan []resolver.Address)
	go func() {
		var (
			snapshotIdx uint64
			ee          []resolver.Address
			err         error
		)
		for {
			ee, snapshotIdx, err = b.service.Endpoints(b.serviceName, snapshotIdx)
			if err != nil {
				grpclog.Errorf("dynBalancer[%s]: dyn provider Watch err: %v. Sleep and retry...", b.serviceName, err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if len(ee) == 0 {
				grpclog.Errorf("dynBalancer[%s]: dyn provider has no endpoints. Waiting...", b.serviceName)
				continue
			}
			eeCh <- ee
		}
	}()
	for {
		select {
		case ee := <-eeCh:
			for _, e := range ee {
				sc, err := b.cc.NewSubConn([]resolver.Address{e}, balancer.NewSubConnOptions{})
				if err != nil {
					grpclog.Errorf("dynBalancer[%s]: HandleResolvedAddrs:  failed to open SubConn('%s'): %v", b.serviceName, e.Addr, err)
					continue
				}
				b.backend.SetSubConnState(sc, connectivity.Idle)
				sc.Connect()
			}
		case <-b.done:
			return
		}
	}
}
