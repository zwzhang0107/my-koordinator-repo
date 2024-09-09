package sharestate

import (
	"sync"

	apis "github.com/koordinator-sh/koordinator/pkg/prediction/manager/apis"
)

type ShareState struct {
	ProfilerList sync.Map
	Count        int
	mu           sync.Mutex
}

func NewShareState() *ShareState {
	return &ShareState{
		ProfilerList: sync.Map{},
	}
}

func (s *ShareState) Add(key apis.ProfileKey, spec *apis.PredictionProfileSpec) error {
	_, loaded := s.ProfilerList.LoadOrStore(key, spec)
	s.mu.Lock()
	defer s.mu.Unlock()

	if !loaded {
		s.Count++
	}
	return nil
}

func (s *ShareState) Get(key apis.ProfileKey) *apis.PredictionProfileSpec {
	spec, found := s.ProfilerList.Load(key)
	if found {
		if profileSpec, ok := spec.(*apis.PredictionProfileSpec); ok {
			return profileSpec
		}
	}
	return nil
}

func (s *ShareState) Delete(key apis.ProfileKey) error {
	_, found := s.ProfilerList.Load(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	if found {
		s.ProfilerList.Delete(key)
		s.Count--
	}
	return nil
}
