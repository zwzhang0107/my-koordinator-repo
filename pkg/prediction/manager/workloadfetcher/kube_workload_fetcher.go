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

package workloadfetcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/informers"

	"github.com/koordinator-sh/koordinator/pkg/prediction/manager/apis"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	v1listers "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DeploymentName  = "Deployment"
	DaemonSetName   = "DaemonSet"
	ReplicaSetName  = "ReplicaSet"
	StatefulSetName = "StatefulSet"
)

type KubeWorkloadFetcher interface {
	GetPodsByWorkload(workload *apis.ControllerRef) ([]*corev1.Pod, error)
	GetPodTemplateOfWorkload(workload *apis.ControllerRef) (*corev1.PodTemplateSpec, error)
	GetPodsBySelector(selector labels.Selector) ([]*corev1.Pod, error)
}

type KubeWorkloadFetcherImpl struct {
	client.Client
	Status           bool
	deploymentLister v1listers.DeploymentLister
	dsLister         v1listers.DaemonSetLister
	replicasetLister v1listers.ReplicaSetLister
	ssLister         v1listers.StatefulSetLister
	podLister        corev1listers.PodLister
	context          context.Context
}

func NewWorkloadFetcher(context context.Context) KubeWorkloadFetcher {
	fetcher := KubeWorkloadFetcherImpl{
		Status:  false,
		context: context,
	}
	fetcher.Init()
	klog.Info("Success Init workload fetcher.")
	return &fetcher
}

func (f *KubeWorkloadFetcherImpl) Init() {
	var wg sync.WaitGroup

	f.Status = true

	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to create in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create Kubernetes clientset: %v", err)
	}
	if err != nil {
		klog.Fatalf("Failed to create discovery clientset: %v", err)
	}

	factory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)

	deploymentInformer := factory.Apps().V1().Deployments()
	dsInformer := factory.Apps().V1().DaemonSets()
	replicasetInformer := factory.Apps().V1().ReplicaSets()
	statefulsetInformer := factory.Apps().V1().StatefulSets()

	podInformer := factory.Core().V1().Pods()

	wg.Add(1)
	go func() {
		defer wg.Done()
		go deploymentInformer.Informer().Run(f.context.Done())
		go dsInformer.Informer().Run(f.context.Done())
		go replicasetInformer.Informer().Run(f.context.Done())
		go statefulsetInformer.Informer().Run(f.context.Done())
		go podInformer.Informer().Run(f.context.Done())
	}()
	wg.Wait()

	if !cache.WaitForCacheSync(f.context.Done(),
		deploymentInformer.Informer().HasSynced,
		dsInformer.Informer().HasSynced,
		replicasetInformer.Informer().HasSynced,
		statefulsetInformer.Informer().HasSynced,
		podInformer.Informer().HasSynced) {
		klog.Fatalf("Failed to sync cache")
	}

	f.deploymentLister = deploymentInformer.Lister()
	f.dsLister = dsInformer.Lister()
	f.replicasetLister = replicasetInformer.Lister()
	f.ssLister = statefulsetInformer.Lister()

	f.podLister = podInformer.Lister()
}

// GetPodsByWorkload returns the pods and template of the workload
func (f *KubeWorkloadFetcherImpl) GetPodsByWorkload(workload *apis.ControllerRef) ([]*corev1.Pod, error) {
	switch workload.Kind {
	case DeploymentName:
		deployment, err := f.deploymentLister.Deployments(workload.Namespace).Get(workload.Name)
		if err != nil {
			return nil, err
		}
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if err != nil {
			return nil, err
		}
		return f.GetPodsBySelector(selector)
	case DaemonSetName:
		ds, err := f.dsLister.DaemonSets(workload.Namespace).Get(workload.Name)
		if err != nil {
			return nil, err
		}
		selector, err := metav1.LabelSelectorAsSelector(ds.Spec.Selector)
		if err != nil {
			return nil, err
		}
		return f.GetPodsBySelector(selector)
	case ReplicaSetName:
		rs, err := f.replicasetLister.ReplicaSets(workload.Namespace).Get(workload.Name)
		if err != nil {
			return nil, err
		}
		selector, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector)
		if err != nil {
			return nil, err
		}
		return f.GetPodsBySelector(selector)
	case StatefulSetName:
		ss, err := f.ssLister.StatefulSets(workload.Namespace).Get(workload.Name)
		if err != nil {
			return nil, err
		}
		selector, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector)
		if err != nil {
			return nil, err
		}
		return f.GetPodsBySelector(selector)
	default:
		return nil, fmt.Errorf("Invalid Type: %+v", workload)
	}
}

// GetPodsBySelector returns the pods by selector
func (f *KubeWorkloadFetcherImpl) GetPodsBySelector(selector labels.Selector) ([]*corev1.Pod, error) {
	return f.podLister.List(selector)
}

func (f *KubeWorkloadFetcherImpl) GetPodTemplateOfWorkload(workload *apis.ControllerRef) (*corev1.PodTemplateSpec, error) {
	var obj client.Object
	switch workload.Kind {
	case DeploymentName:
		t, err := f.deploymentLister.Deployments(workload.Namespace).Get(workload.Name)
		if err != nil {
			return nil, err
		}
		return &t.Spec.Template, nil
	case DaemonSetName:
		t, err := f.dsLister.DaemonSets(workload.Namespace).Get(workload.Name)
		if err != nil {
			return nil, err
		}
		return &t.Spec.Template, nil
	case ReplicaSetName:
		t, err := f.replicasetLister.ReplicaSets(workload.Namespace).Get(workload.Name)
		if err != nil {
			return nil, err
		}
		return &t.Spec.Template, nil
	case StatefulSetName:
		t, err := f.ssLister.StatefulSets(workload.Namespace).Get(workload.Name)
		if err != nil {
			return nil, err
		}
		return &t.Spec.Template, nil
	default:
		return nil, fmt.Errorf("Invalid Type %+v", obj)
	}
}

// ControllerWorkloadKey identifies a k8s workload
type ControllerWorkloadKey struct {
	ApiVersion string
	Kind       string
	Name       string
	Namespace  string
}

// ControllerWorkloadStatus contains the pod template and all pods of a k8s workload
type ControllerWorkloadStatus struct {
	Pods        []*corev1.Pod
	PodTemplate corev1.PodTemplateSpec
}
