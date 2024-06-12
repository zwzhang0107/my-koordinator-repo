/*
Copyright 2022 The Koordinator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package checkpoint

import (
	"sync"

	"k8s.io/klog/v2"
)

type Checkpoint interface {
	Run()
	SaveToCheckpoint(checkpointKey string, snapshotData *SnapshotData) error
	LoadFromCheckpoint(checkpointKey string) (*SnapshotData, error)
}
type Snapshot interface {
	SaveToSnapshot() (interface{}, error)
	LoadFromSnapshot(interface{}) error
}

// like: workloadNamespace/workloadName-modeName
type CheckpointKey string

// key is "modelData"
type ModelData map[string]interface{}

// key is "modelArgs"
type ModelArgs map[string]interface{}
type SnapshotData struct {
	ModelData ModelData
	ModelArgs ModelArgs
}

var _ Checkpoint = &checkpointImpl{}

type checkpointImpl struct {
	// key is CheckpointKey, value is Checkpoint CR
	checkpointMap   sync.Map
	SnapshotDataMap sync.Map
	// ckClient
}

func (cp *checkpointImpl) Run() {
	// 1. load all checkpoints to memory cache
	if err := cp.loadAllCheckpoint(); err != nil {
		klog.Errorf("failed to load all checkpoint ")
	}

	// 2. start checkpoint gc routine
}
func (cp *checkpointImpl) checkpointGcRoutine() {
	// if the checkopint CR updateTime in cache is expired, delete checkopint CR forever.
}

func (cp *checkpointImpl) loadAllCheckpoint() error {
	// do load all checkpoints
	return nil
}

func (cp *checkpointImpl) SaveToCheckpoint(checkpointKey string, snapshotData *SnapshotData) error {
	// 1. update memory cache
	// 2. create the checkoutpoint CR if not exist
	// 3. update the checkoutpoint CR
	return nil
}

func (cp *checkpointImpl) LoadFromCheckpoint(checkpointKey string) (*SnapshotData, error) {
	// 1. get snapshotData from memory cache
	// 2. if not exist in memory cache, get checkpoint CR from apiServer and update cache;
	// 3. if not found in apiServer, return nil and not found

	return nil, nil
}
