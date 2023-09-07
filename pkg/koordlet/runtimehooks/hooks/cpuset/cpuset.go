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

package cpuset

import (
	"fmt"
	"sync"

	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"

	apiext "github.com/koordinator-sh/koordinator/apis/extension"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/resourceexecutor"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/runtimehooks/hooks"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/runtimehooks/protocol"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/runtimehooks/reconciler"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/runtimehooks/rule"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/statesinformer"
	sysutil "github.com/koordinator-sh/koordinator/pkg/koordlet/util/system"
	rmconfig "github.com/koordinator-sh/koordinator/pkg/runtimeproxy/config"
	"github.com/koordinator-sh/koordinator/pkg/util"
)

const (
	name        = "CPUSetAllocator"
	description = "set cpuset value by pod allocation"
)

type cpusetPlugin struct {
	rule        *cpusetRule
	ruleRWMutex sync.RWMutex
	executor    resourceexecutor.ResourceUpdateExecutor
}

var (
	nonBEPodQOSConditions = []string{string(apiext.QoSSystem), string(apiext.QoSLSE), string(apiext.QoSLSR)}
	bePodQOSConditions    = []string{string(apiext.QoSBE)}
)

func (p *cpusetPlugin) Register(op hooks.Options) {
	klog.V(5).Infof("register hook %v", name)
	hooks.Register(rmconfig.PreCreateContainer, name, description, p.SetContainerCPUSetAndUnsetCFS)
	hooks.Register(rmconfig.PreUpdateContainerResources, name, description, p.SetContainerCPUSetAndUnsetCFS)
	hooks.Register(rmconfig.PreRunPodSandbox, name, "unset pod cpu quota if needed", UnsetPodCPUQuota)
	rule.Register(name, description,
		rule.WithParseFunc(statesinformer.RegisterTypeNodeTopology, p.parseRule),
		rule.WithUpdateCallback(p.ruleUpdateCb))
	reconciler.RegisterCgroupReconciler(reconciler.ContainerLevel, sysutil.CPUSet,
		"set container cpuset and unset container cpu quota if needed",
		p.SetContainerCPUSetAndUnsetCFS, reconciler.PodQOSFilter(), nonBEPodQOSConditions...)
	reconciler.RegisterCgroupReconciler(reconciler.SandboxLevel, sysutil.CPUSet,
		"set sandbox container cpuset and unset container cpu quota if needed",
		p.SetContainerCPUSetAndUnsetCFS, reconciler.PodQOSFilter(), nonBEPodQOSConditions...)
	reconciler.RegisterCgroupReconciler(reconciler.PodLevel, sysutil.CPUCFSQuota, "unset pod cpu quota if needed",
		UnsetPodCPUQuota, reconciler.PodQOSFilter(), nonBEPodQOSConditions...)

	reconciler.RegisterCgroupReconciler(reconciler.ContainerLevel, sysutil.CPUSet,
		"set container cpuset for be pod is specified",
		p.SetContainerCPUSet, reconciler.PodQOSFilter(), bePodQOSConditions...)
	reconciler.RegisterCgroupReconciler(reconciler.SandboxLevel, sysutil.CPUSet,
		"set sandbox container cpuset for be pod is specified",
		p.SetContainerCPUSet, reconciler.PodQOSFilter(), bePodQOSConditions...)
	p.executor = op.Executor
}

var singleton *cpusetPlugin

func Object() *cpusetPlugin {
	if singleton == nil {
		singleton = &cpusetPlugin{}
	}
	return singleton
}

func (p *cpusetPlugin) SetContainerCPUSetAndUnsetCFS(proto protocol.HooksProtocol) error {
	// set container-level cpuset.cpus
	err := p.SetContainerCPUSet(proto)
	if err != nil {
		return err
	}

	// unset container-level cpu.cfs_quota_us if needed
	return UnsetContainerCPUQuota(proto)
}

func (p *cpusetPlugin) SetContainerCPUSet(proto protocol.HooksProtocol) error {
	containerCtx := proto.(*protocol.ContainerContext)
	if containerCtx == nil {
		return fmt.Errorf("container protocol is nil for plugin %v", name)
	}
	containerReq := containerCtx.Request
	klog.V(5).Infof("getting container cpuset for %v/%v", containerReq.PodMeta.String(), containerReq.ContainerMeta.Name)

	// cpuset from pod annotation (LSE, LSR)
	if cpusetVal, err := util.GetCPUSetFromPod(containerReq.PodAnnotations); err != nil {
		return err
	} else if cpusetVal != "" {
		containerCtx.Response.Resources.CPUSet = pointer.String(cpusetVal)
		klog.V(5).Infof("get cpuset %v for container %v/%v from pod annotation", cpusetVal,
			containerCtx.Request.PodMeta.String(), containerCtx.Request.ContainerMeta.Name)
		return nil
	}

	r := p.getRule()
	if r == nil {
		klog.V(5).Infof("hook plugin rule is nil, nothing to do for plugin %v", name)
		return nil
	}

	// cpuset from rule according to pod QoS
	cpusetValue, err := r.getContainerCPUSet(&containerReq)
	if err != nil {
		return err
	}
	if cpusetValue != nil {
		klog.V(5).Infof("get cpuset %v for container %v/%v from rule", *cpusetValue,
			containerCtx.Request.PodMeta.String(), containerCtx.Request.ContainerMeta.Name)
	} else {
		klog.V(5).Infof("get empty cpuset for container %v/%v from rule", *cpusetValue,
			containerCtx.Request.PodMeta.String(), containerCtx.Request.ContainerMeta.Name)
	}
	containerCtx.Response.Resources.CPUSet = cpusetValue
	return nil
}

func UnsetPodCPUQuota(proto protocol.HooksProtocol) error {
	podCtx := proto.(*protocol.PodContext)
	if podCtx == nil {
		return fmt.Errorf("pod protocol is nil for plugin %v", name)
	}
	req := podCtx.Request

	// cpuset from pod annotation (LSE, LSR)
	// NOTE: unset cfs quota for cpuset pods to avoid unexpected throttles.
	// https://github.com/koordinator-sh/koordinator/issues/489
	if needUnset, err := util.IsPodCfsQuotaNeedUnset(req.Annotations); err != nil {
		return err
	} else if needUnset {
		podCtx.Response.Resources.CFSQuota = pointer.Int64(-1)
		return nil
	}

	// do nothing for cpushare pod
	return nil
}

func UnsetContainerCPUQuota(proto protocol.HooksProtocol) error {
	containerCtx := proto.(*protocol.ContainerContext)
	if containerCtx == nil {
		return fmt.Errorf("container protocol is nil for plugin %v", name)
	}
	containerReq := containerCtx.Request

	// cpuset from pod annotation (LSE, LSR)
	// NOTE: unset cfs quota for cpuset pods to avoid unexpected throttles.
	// https://github.com/koordinator-sh/koordinator/issues/489
	if needUnset, err := util.IsPodCfsQuotaNeedUnset(containerReq.PodAnnotations); err != nil {
		return err
	} else if needUnset {
		containerCtx.Response.Resources.CFSQuota = pointer.Int64(-1)
		return nil
	}

	// do nothing for cpushare pod
	return nil
}
