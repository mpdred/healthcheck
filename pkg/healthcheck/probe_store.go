package healthcheck

import "sync"

type ProbeStore interface {
	Add(probes ...Probe)

	Get(name string) Probe
	GetAll() []Probe

	// GetByKind returns all probes that have a matching ProbeKind.
	GetByKind(kind ProbeKind) []Probe

	Delete(names ...string)
}

type inMemoryProbeStore struct {
	mu sync.RWMutex

	probes map[string]Probe
}

func (s *inMemoryProbeStore) Add(probes ...Probe) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range probes {
		s.probes[p.Name] = p
	}
}

func (s *inMemoryProbeStore) Get(name string) Probe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p := s.probes[name]

	return p
}

func (s *inMemoryProbeStore) GetAll() []Probe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	probeList := make([]Probe, 0, len(s.probes))
	for _, p := range s.probes {
		probeList = append(probeList, p)
	}

	return probeList
}

func (s *inMemoryProbeStore) GetByKind(kind ProbeKind) []Probe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	probeList := make([]Probe, 0)
	for _, p := range s.probes {
		if p.Kind != kind {
			continue
		}

		probeList = append(probeList, p)
	}

	return probeList
}

func (s *inMemoryProbeStore) Delete(names ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, name := range names {
		delete(s.probes, name)
	}
}

func NewInMemoryProbeStore() ProbeStore {
	s := &inMemoryProbeStore{
		mu:     sync.RWMutex{},
		probes: map[string]Probe{},
	}

	return s
}
