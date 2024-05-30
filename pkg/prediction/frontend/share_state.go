package frontend

import (
	"k8s.io/klog/v2"
)

type ShareState struct {
	ProfilerList map[apis.ProfileKey]bool
}

func NewShareState() ShareState {
	return ShareState{
		ProfilerList: make(map[apis.ProfileKey]bool),
	}
}

func (s ShareState) Add(key apis.ProfileKey) error {
	if status, ok := s.ProfilerList[key]; ok {
		if status {
			klog.Infof("Already Profile %+v", key)
			return nil
		} else {
			s.ProfilerList[key] = true
			klog.Infof("Reopen Profiler %+v", key)
		}
	}
	s.ProfilerList[key] = true
	klog.Infof("Add Profiler %+v", key)
	return nil
}

func (s ShareState) Delete(key apis.ProfileKey) error {
	delete(s.ProfilerList, key)
	klog.Infof("Delete Profiler %+v", key)
	return nil
}
