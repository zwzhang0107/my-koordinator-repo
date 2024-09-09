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

package metricscollector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type Sample interface {
	Value() (time.Time, float64)
}

type sample struct {
	timestamp time.Time
	value     float64
}

func (s *sample) Value() (time.Time, float64) {
	return s.timestamp, s.value
}

type MetricsServerRepository interface {
	Start() error
	GetAllContainerCPUUsage(containerName string, pods []types.NamespacedName) (map[types.NamespacedName]Sample, error)
	GetAllContainerMemoryUsage(containerName string, pods []types.NamespacedName) (map[types.NamespacedName]Sample, error)
}

type metricsServerRepoImpl struct {
	Client    client.Client
	cache     *unstructured.UnstructuredList
	cacheLock sync.RWMutex
}

func (m *metricsServerRepoImpl) Start() error {
	go m.Update()
	return nil
}

func (m *metricsServerRepoImpl) Update() error {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	err := m.CacheData()
	if err != nil {
		klog.Errorf("Error fetching metrics: %v", err)
	}
	for {
		select {
		case <-ticker.C:
			klog.Info("maliang: cache data from metric server.")
			err := m.CacheData()
			if err != nil {
				klog.Errorf("Error fetching metrics: %v", err)
				continue
			}
		}
	}
}

func (m *metricsServerRepoImpl) Started() bool {
	return true
}

func (m *metricsServerRepoImpl) CacheData() error {
	defer m.cacheLock.Unlock()
	m.cacheLock.Lock()
	return m.Client.List(context.TODO(), m.cache)
}

func (m *metricsServerRepoImpl) GetAllContainerCPUUsage(containerName string, pods []types.NamespacedName) (map[types.NamespacedName]Sample, error) {
	if m.cache == nil {
		return nil, fmt.Errorf("No valid cache.")
	}
	if len(m.cache.Items) == 0 {
		return nil, fmt.Errorf("No valid cache items")
	}
	result := make(map[types.NamespacedName]Sample)
	m.cacheLock.RLock()
	for _, pm := range m.cache.Items {
		found := false
		for _, pod := range pods {
			if pod.Name == pm.GetName() && pod.Namespace == pm.GetNamespace() {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		containers, found, err := unstructured.NestedSlice(pm.Object, "containers")
		if err != nil || !found {
			continue
		}

		for _, container := range containers {
			containerMap, ok := container.(map[string]interface{})
			if !ok {
				klog.Errorf("invalid container data")
				continue
			}
			containerNameInPod, found, err := unstructured.NestedString(containerMap, "name")
			if !found || err != nil || containerNameInPod != containerName {
				//log.Printf("Failed to get container name for pod %s/%s: %v", pm.GetNamespace(), pm.GetName(), err)
				continue
			}
			usage, found, err := unstructured.NestedMap(containerMap, "usage")
			if !found || err != nil {
				klog.Warningf("Failed to get usage for container %s in pod %s/%s: %v", containerNameInPod, pm.GetNamespace(), pm.GetName(), err)
				continue
			}
			cpuUsage, found, err := unstructured.NestedString(usage, "cpu")
			if !found || err != nil {
				klog.Warningf("Failed to get CPU usage for container %s in pod %s/%s: %v", containerNameInPod, pm.GetNamespace(), pm.GetName(), err)
				continue
			}
			cpu, _, err := ParseResourceUsage(cpuUsage, "cpu")
			if err != nil {
				klog.V(6).Infof("Failed to parse CPU usage for container %s in pod %s/%s: %v", containerNameInPod, pm.GetNamespace(), pm.GetName(), err)
				continue
			}
			namespacedName := types.NamespacedName{
				Namespace: pm.GetNamespace(),
				Name:      pm.GetName(),
			}
			result[namespacedName] = &sample{
				timestamp: time.Now(),
				value:     cpu,
			}
		}
	}
	m.cacheLock.RUnlock()
	return result, nil
}

// the pods Namespaced should be same.
func (m *metricsServerRepoImpl) GetAllContainerMemoryUsage(containerName string, pods []types.NamespacedName) (map[types.NamespacedName]Sample, error) {
	if m.cache == nil {
		return nil, fmt.Errorf("No valid cache.")
	}
	if len(m.cache.Items) == 0 {
		return nil, fmt.Errorf("No valid cache items")
	}
	result := make(map[types.NamespacedName]Sample)
	m.cacheLock.RLock()
	for _, pm := range m.cache.Items {
		found := false
		for _, pod := range pods {
			if pod.Name == pm.GetName() && pod.Namespace == pm.GetNamespace() {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		containers, found, err := unstructured.NestedSlice(pm.Object, "containers")
		if err != nil || !found {
			klog.Errorf("Can not get containers of pod %s/%s", pm.GetNamespace(), pm.GetName())
			continue
		}
		for _, container := range containers {
			containerMap, ok := container.(map[string]interface{})
			if !ok {
				klog.Errorf("invalid container data")
				continue
			}
			containerNameInPod, found, err := unstructured.NestedString(containerMap, "name")
			if !found || err != nil || containerNameInPod != containerName {
				//log.Printf("Failed to get container name for pod %s/%s: %v", pm.GetNamespace(), pm.GetName(), err)
				continue
			}

			usage, found, err := unstructured.NestedMap(containerMap, "usage")
			if !found || err != nil {
				klog.Warningf("Failed to get usage for container %s in pod %s/%s: %v", containerNameInPod, pm.GetNamespace(), pm.GetName(), err)
				continue
			}
			memoryUsage, found, err := unstructured.NestedString(usage, "memory")
			if !found || err != nil {
				klog.Warningf("Failed to get memory usage for container %s in pod %s/%s: %v", containerName, pm.GetNamespace(), pm.GetName(), err)
				continue
			}
			memory, _, err := ParseResourceUsage(memoryUsage, "memory")
			if err != nil {
				klog.V(6).Infof("Failed to parse Memory usage for container %s in pod %s/%s: %v", containerNameInPod, pm.GetNamespace(), pm.GetName(), err)
				continue
			}
			namespacedName := types.NamespacedName{
				Namespace: pm.GetNamespace(),
				Name:      pm.GetName(),
			}
			result[namespacedName] = &sample{
				timestamp: time.Now(),
				value:     memory,
			}
		}
	}
	m.cacheLock.RUnlock()
	return result, nil
}

func NewMetricServerRepo(client client.Client) MetricsServerRepository {
	m := &metricsServerRepoImpl{
		Client:    client,
		cache:     nil,
		cacheLock: sync.RWMutex{},
	}
	podMetricsGVK := schema.GroupVersionKind{
		Group:   "metrics.k8s.io",
		Version: "v1beta1",
		Kind:    "PodMetrics",
	}
	m.cache = &unstructured.UnstructuredList{}
	m.cache.SetGroupVersionKind(podMetricsGVK)
	return m
}
