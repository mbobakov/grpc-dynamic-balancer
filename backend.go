package balancer

import (
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/connectivity"
)

type inMemoryBackend struct {
	value sync.Map
}

func (m *inMemoryBackend) SetSubConnState(sc balancer.SubConn, s connectivity.State) {
	m.value.Store(sc, s)
}

func (m *inMemoryBackend) SubConnState(sc balancer.SubConn) connectivity.State {
	st, _ := m.value.LoadOrStore(sc, connectivity.Idle)
	return st.(connectivity.State)
}

func (m *inMemoryBackend) Delete(sc balancer.SubConn) {
	m.value.Delete(sc)
}
func (m *inMemoryBackend) Picker() balancer.Picker {
	var ready []balancer.SubConn
	m.value.Range(func(sc, st interface{}) bool {
		s, ok := st.(connectivity.State)
		if !ok {
			return true
		}
		if s == connectivity.Ready {
			ready = append(ready, sc.(balancer.SubConn))
		}
		return true
	})
	if len(ready) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	// Now Only ROUND ROBIN method is implemented
	return &rrPicker{subConns: ready}
}

func (m *inMemoryBackend) AggregatedState() connectivity.State {
	var ready, connecting int
	m.value.Range(func(sc, st interface{}) bool {
		s, ok := st.(connectivity.State)
		if !ok {
			return true
		}
		switch s {
		case connectivity.Ready:
			ready++
		case connectivity.Connecting:
			connecting++
		}
		return true
	})
	switch {
	case ready > 0:
		return connectivity.Ready
	case connecting > 0:
		return connectivity.Connecting
	}
	return connectivity.TransientFailure
}
