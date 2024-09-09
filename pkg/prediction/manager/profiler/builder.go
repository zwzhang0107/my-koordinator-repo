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

package profiler

import (
	"context"
	"fmt"

	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/apis"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/checkpoint"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/metricscollector"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/workloadfetcher"
	"go.etcd.io/etcd/client/v2"
)

type Factory struct {
	metricsServerRepo   metricscollector.MetricsServerRepository
	KubeWorkloadFetcher workloadfetcher.KubeWorkloadFetcher
	checkpoint          checkpoint.Checkpoint
	source              string
}

func NewFactory(client client.Client, region, provider string, period int, source string, context context.Context) Factory {
	factory := Factory{
		source:              source,
		metricsServerRepo:   metricscollector.NewMetricServerRepo(client),
		checkpoint:          checkpoint.NewCheckPoint(),
		KubeWorkloadFetcher: workloadfetcher.NewWorkloadFetcher(context),
	}
	factory.metricsServerRepo.Start()
	return factory
}

func (f *Factory) New(profile *apis.PredictionProfileSpec) (Profiler, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}
	switch profile.Profiler.Model {
	case apis.ProfilerTypeDistribution:
		return NewKubeMetricDistribution(profile, f.metricsServerRepo, f.KubeWorkloadFetcher, f.checkpoint)
	default:
		return nil, fmt.Errorf("profiler is not supported %v", profile)
	}
}
