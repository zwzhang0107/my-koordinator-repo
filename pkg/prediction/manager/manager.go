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

package manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/apis"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/checkpoint"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/metricscollector"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/profiler"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/sharestate"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/workloadfetcher"
	"go.etcd.io/etcd/client/v2"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PredictionManager interface {
	Run() error
	Started() bool
	Register(apis.ProfileKey, apis.PredictionProfileSpec) error
	Unregister(apis.ProfileKey) error
	GetResult(apis.ProfileKey, apis.ProfileResult, *apis.GetResultOptions) error
}

// Options are the arguments for creating a new Manager.
type Options struct {
	client.Client
	source            string
	ctx               context.Context
	period            int
	prometheusPeriod  int
	updateCachePeriod int
	profileParallel   int
	featureParallel   int
	zone              string
	provider          string
	token             string
}

// create new option.
// client: client to get api object;
// ctx: context that manage the life cricle;
// period: fetch metric server time interval
func NewOptions(source string, client client.Client, ctx context.Context,
	period, prometheusPeriod, updateCachePeriod int,
	zone, provider string, token string,
	profileParallel, featureParallel int) Options {
	return Options{
		source:            source,
		Client:            client,
		ctx:               ctx,
		period:            period,
		prometheusPeriod:  prometheusPeriod,
		updateCachePeriod: updateCachePeriod,
		zone:              zone,
		provider:          provider,
		token:             token,

		profileParallel: profileParallel,
		featureParallel: featureParallel,
	}
}

func New(opt Options) PredictionManager {
	shareState := sharestate.NewShareState()
	predictionMgrImpl := predictionMgrImpl{
		ctx:               opt.ctx,
		source:            opt.source,
		ProfilePeriod:     opt.period,
		UpdateCachePeriod: opt.updateCachePeriod,
		factory:           profiler.NewFactory(opt.Client, opt.zone, opt.provider, opt.period, opt.token, opt.source, opt.ctx),
		profilers:         sync.Map{},
		ShareState:        shareState,
		profilersMutex:    sync.RWMutex{},
		zone:              opt.zone,
		provider:          opt.provider,
		token:             opt.token,
	}
	return &predictionMgrImpl
}

var _ PredictionManager = &predictionMgrImpl{}

type predictionMgrImpl struct {
	metricsServerRepo   metricscollector.MetricsServerRepository
	checkpoint          checkpoint.Checkpoint
	KubeWorkloadFetcher workloadfetcher.KubeWorkloadFetcher

	ctx               context.Context
	ProfilePeriod     int
	UpdateCachePeriod int

	factory profiler.Factory

	ShareState *sharestate.ShareState
	profilers  sync.Map
	Count      uint64

	profilersMutex sync.RWMutex
	zone           string
	provider       string
	token          string
	source         string
}

func (p *predictionMgrImpl) GetShareState() (shareState *sharestate.ShareState) {
	return p.ShareState
}

func (p *predictionMgrImpl) GetProfiler(key apis.ProfileKey) (profiler.Profiler, error) {
	spec, found := p.profilers.Load(key)
	if found {
		if profiler, ok := spec.(profiler.Profiler); ok {
			return profiler, nil
		}
	}
	return nil, fmt.Errorf("no Such Profiler, key: %+v", key)
}

func (p *predictionMgrImpl) SetProfiler(key apis.ProfileKey, profiler profiler.Profiler) {
	_, loaded := p.profilers.LoadOrStore(key, profiler)
	p.profilersMutex.Lock()
	defer p.profilersMutex.Unlock()

	if !loaded {
		p.Count++
	}
}

func (p *predictionMgrImpl) DeleteProfiler(key apis.ProfileKey) {
	_, found := p.profilers.Load(key)
	p.profilersMutex.Lock()
	defer p.profilersMutex.Unlock()

	if found {
		p.profilers.Delete(key)
		p.Count--
	}
}

func (p *predictionMgrImpl) RunProfiler(parallel int, name string) {
	var profilerWg sync.WaitGroup
	sem := make(chan struct{}, parallel)
	klog.Info("profilers number:", p.ShareState.Count)
	p.ShareState.ProfilerList.Range(func(key, value interface{}) bool {
		sem <- struct{}{}
		profilerWg.Add(1)
		go func(key interface{}) {
			defer func() { <-sem }()
			defer profilerWg.Done()
			k := key.(apis.ProfileKey)
			profiler, err := p.GetProfiler(k)
			if err != nil {
				klog.Errorf("error while profile %+v: %+v", key, err)
				return
			}
			err = profiler.Profile()
			if err != nil {
				klog.V(5).Infof(`profile %+v failed: %+v`, key, err)
			}
		}(key)
		return true
	})
}

func (p *predictionMgrImpl) Started() bool {
	// return true only if all components are started
	return p.factory.Started()
}

func (p *predictionMgrImpl) Register(key apis.ProfileKey, profile *apis.PredictionProfileSpec) error {
	// return error if not started
	if !p.Started() {
		klog.Errorf("Manager do not start!")
		return fmt.Errorf("prediction Manager do not start")
	}
	// Update Profilers
	_, err := p.GetProfiler(key)
	if err != nil {
		// Init feature profiler and pass to channel
		p.ShareState.Add(key, profile)
		// Init analysis profiler
		profiler, err := p.factory.New(profile, p.zone)
		if err != nil {
			return err
		}
		p.SetProfiler(key, profiler)
		klog.Infof("Register Profiler Success. %+v", key)
	} else {
		// update profile if already registered
		profiler, err := p.GetProfiler(key)
		if err != nil {
			return err
		}
		err = profiler.Update(profile)
		if err != nil {
			klog.Error("Error while Build Profiler: ", err)
			return err
		}
		klog.Infof("Update Profiler Success. %+v", key)
	}
	return nil
}

func (p *predictionMgrImpl) Unregister(key apis.ProfileKey) error {
	// return error if not started
	if !p.Started() {
		klog.Error("Prediction Manager do not start.")
		return nil
	}
	// remove profile from map
	if _, err := p.GetProfiler(key); err != nil {
		klog.Errorf("Unregister key failed, %v did not register.", key)
	} else {
		p.DeleteProfiler(key)
	}
	// remove from sharestate
	p.ShareState.Delete(key)
	return nil
}

func (p *predictionMgrImpl) GetResult(key apis.ProfileKey, result apis.ProfileResult, opt *apis.GetResultOptions) error {
	// return error if not started
	if !p.Started() {
		return fmt.Errorf("prediction Manager do not start")
	}
	// return error if not registered
	if profiler, err := p.GetProfiler(key); err == nil {
		err := profiler.GetResult(result)
		if err != nil {
			return fmt.Errorf("can not get result: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unregister key failed, %v did not register", key)
}
