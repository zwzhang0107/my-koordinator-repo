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

import corev1 "k8s.io/api/core/v1"

// MetricSourceType defines the type of metric source
type MetricSourceType string

var (
	PrometheusCPU             = corev1.ResourceCPU.String()
	PrometheusMEM             = corev1.ResourceMemory.String()
	PrometheusStorageWrite    = StorageWrite.String()
	PrometheusStorageRead     = StorageRead.String()
	PrometheusMemoryBandwidth = MemoryBandwidth.String()
	PrometheusDiskRead        = DiskRead.String()
	PrometheusDiskWrite       = DiskWrite.String()
	PrometheusMetrics         = []string{PrometheusCPU, PrometheusMEM, PrometheusStorageWrite, PrometheusStorageRead, PrometheusMemoryBandwidth, PrometheusDiskRead, PrometheusDiskWrite}
	PrometheusInfraMetrics    = []string{PrometheusCPU, PrometheusMEM, PrometheusStorageWrite, PrometheusStorageRead}
	PrometheusKubeMetrics     = []string{PrometheusMemoryBandwidth, PrometheusDiskRead, PrometheusDiskWrite}

	InfraResourceList = []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, StorageRead, StorageWrite}
	KubeResourceList  = []corev1.ResourceName{MemoryBandwidth, DiskRead, DiskWrite}
	ResourceList      = []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, StorageRead, StorageWrite, MemoryBandwidth, DiskRead, DiskWrite}
)

const (
	StorageWrite    corev1.ResourceName = "fswrite"
	StorageRead     corev1.ResourceName = "fsread"
	MemoryBandwidth corev1.ResourceName = "memory_bandwidth"
	DiskRead        corev1.ResourceName = "disk_read"
	DiskWrite       corev1.ResourceName = "disk_write"
)

const (
	// MetricSourceTypeMetricsAPI means the metric is from metric server
	MetricSourceTypeMetricsAPI MetricSourceType = "MetricsAPI"
	// MetricSourceTypePrometheus means the metric is from prometheus
	MetricSourceTypePrometheus MetricSourceType = "Prometheus"
)

func MakeMetricSpec() MetricSpec {
	return MetricSpec{
		Source:       MetricSourceTypeMetricsAPI,
		MetricServer: &MetricServerSource{},
		Prometheus:   nil,
	}
}

// MetricSpec defines the metric to be analysis
type MetricSpec struct {
	// Source defines the source of metric, which can be metric server or prometheus
	Source MetricSourceType `json:"source"`
	// MetricServer defines the metric server source, which is effective when source is metric server
	MetricServer *MetricServerSource `json:"metricServer,omitempty"`
	// Prometheus defines the prometheus source, which is effective when source is prometheus
	Prometheus *PrometheusMetricSource `json:"prometheus,omitempty"`
}

// MetricServerSource defines the metric server source
type MetricServerSource struct {
	// Resources defines the key to indicates the resources to be analyzed, only cpu and memory supported for metric server
	Names []corev1.ResourceName `json:"names,omitempty"`
}

// PrometheusMetricSource defines the prometheus metric source
type PrometheusMetricSource struct {
	// Metrics defines the prometheus metrics to be analyzed
	Metrics []PrometheusMetric `json:"metrics,omitempty"`
}

// PrometheusMetric defines the prometheus metric to be analyzed
type PrometheusMetric struct {
	// Source defines the data source
	Source string `json:"source,omitempty"`
	// Name defines the key of resource to be analyzed
	Name []string `json:"name,omitempty"`
	// Metric is the name of prometheus metric, such as container_cpu_usage
	Metric []string `json:"metric,omitempty"`
	// TODO more fields for metric label mapping to workload target
}
