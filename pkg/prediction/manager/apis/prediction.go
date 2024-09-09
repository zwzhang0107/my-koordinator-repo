/*
 Copyright 2024 The Koordinator Authors.

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

package apis

import (
	analysisv1alpha1 "github.com/koordinator-sh/koordinator/apis/analysis/v1alpha1"
)

type ProfileKey interface {
	// Key returns the key of the profile, which should be unique in the scope of the prediction manager.
	Key() string
	Namespace() string
	Name() string
	NamePattern() string
}

func MakeProfilerSpec(profileKey ProfileKey, analysisSpec analysisv1alpha1.Recommendation) *PredictionProfileSpec {
	hierarchy := ProfileHierarchy{
		Level: ProfileHierarchyLevelContainer,
	}
	types := WorkloadTargetType(analysisSpec.Spec.Target.Type)
	if types == WorkloadTargetController {
		return &PredictionProfileSpec{
			Key: profileKey,
			Profiler: Profiler{
				Model:        ProfilerTypeDistribution,
				Distribution: &DistributionModel{},
			},
			Target: WorkloadTarget{
				Type: WorkloadTargetType(analysisSpec.Spec.Target.Type),
				Controller: &ControllerRef{
					Kind:       analysisSpec.Spec.Target.Workload.Kind,
					Name:       analysisSpec.Spec.Target.Workload.Name,
					Namespace:  analysisSpec.Spec.Target.Workload.Namespace,
					APIVersion: analysisSpec.Spec.Target.Workload.APIVersion,
					Hierarchy:  hierarchy,
				},
			},
			Metric: MetricSpec{
				// For workloads, default set to mix type
				Source: MetricSourceTypeMetricsAPI,
				MetricServer: &MetricServerSource{
					Names: ResourceList,
				},
				Prometheus: &PrometheusMetricSource{
					Metrics: []PrometheusMetric{
						{
							Source: "vms-infra",
							Name:   PrometheusInfraMetrics,
						},
						{
							Source: "vms-kube",
							Name:   PrometheusKubeMetrics,
						},
					},
				},
			},
		}
	} else if types == WorkloadTargetPodSelector {
		return &PredictionProfileSpec{
			Key: profileKey,
			Profiler: Profiler{
				Model:        ProfilerTypeDistribution,
				Distribution: &DistributionModel{},
			},
			Target: WorkloadTarget{
				Type: WorkloadTargetType(analysisSpec.Spec.Target.Type),
				PodSelector: &PodSelectorRef{
					Selector:  analysisSpec.Spec.Target.PodSelector,
					Hierarchy: hierarchy,
				},
			},
			Metric: MetricSpec{
				Source: MetricSourceTypeMetricsAPI,
				MetricServer: &MetricServerSource{
					Names: ResourceList,
				},
			},
		}

	} else {
		return nil
	}

}

type PredictionProfileSpec struct {
	Key ProfileKey
	// Profiler defines the profiler type of prediction
	Profiler Profiler `json:"profiler"`
	// Target is the object to be analyzed, which can be a workload or a series of pods
	Target WorkloadTarget `json:"target"`
	// Metric defines the source of metric, including resource name, metric name and how to collect
	Metric MetricSpec `json:"metric"`
}

type GetResultOptions struct {
}
