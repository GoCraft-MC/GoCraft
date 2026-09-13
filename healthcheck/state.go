package healthcheck

import (
	"sync/atomic"
)

type HealthState struct {
	Live    atomic.Bool
	Ready   atomic.Bool
	Healthy atomic.Bool
}

func (s *HealthState) IsReady() bool {
	return s.Ready.Load()
}

func (s *HealthState) SetReady(v bool) {
	s.Ready.Store(v)
}

func (s *HealthState) IsHealthy() bool {
	return s.Healthy.Load()
}

func (s *HealthState) SetHealthy(v bool) {
	s.Healthy.Store(v)
}

func (s *HealthState) IsLive() bool {
	return s.Live.Load()
}

func (s *HealthState) SetLive(v bool) {
	s.Live.Store(v)
}
