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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cp
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Model",type="string",JSONPath=".spec.modelname"
// +kubebuilder:printcolumn:name="namespace",type="string",JSONPath=".spec.workloadnamspace"
// +kubebuilder:printcolumn:name="Name",type="string",JSONPath=".spec.workloadname"
// +kubebuilder:printcolumn:name="LastUpdateTime",type="date",JSONPath=".status.lastupdateTime"

// Checkpoint is the model data that is used for recovery after restart.
type Checkpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the checkpoint.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status.
	// +optional
	Spec CheckpointSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Data of the checkpoint.
	// +optional
	Status CheckpointStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +kubebuilder:object:root=true

// CheckpointList is a list of Checkpoint objects.
type CheckpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Checkpoint `json:"items"`
}

// CheckpointSpec is the specification of the checkpoint object.
type CheckpointSpec struct {
	// Namespace of the workload that this checkpoint belongs to
	WorkLoadNamespace string `json:"workloadNamespace,omitempty" protobuf:"bytes,1,opt,name=workloadNamespace"`
	// Name of the workload that this checkpoint belongs to
	WorkLoadName string `json:"workloadName,omitempty" protobuf:"bytes,1,opt,name=workloadName"`
	// Name of Model that this checkpoint using
	ModelName string `json:"modelName,omitempty" protobuf:"bytes,1,opt,name=modelName"`
	// Arguments of the model using
	ModelArgs *ModelMap `json:"modelArgs,omitempty"`
}

// CheckpointStatus contains data of the checkpoint.
type CheckpointStatus struct {
	// The time when the status was last refreshed.
	// +nullable
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" protobuf:"bytes,1,opt,name=lastUpdateTime"`

	// data of the model
	ModelData *ModelMap `json:"modelData,omitempty"`

	// Timestamp of the fist sample from the model.
	// +nullable
	FirstSampleStart metav1.Time `json:"firstSampleStart,omitempty" protobuf:"bytes,4,opt,name=firstSampleStart"`

	// Timestamp of the last sample from the model.
	// +nullable
	LastSampleStart metav1.Time `json:"lastSampleStart,omitempty" protobuf:"bytes,5,opt,name=lastSampleStart"`

	// Total number of samples in the model.
	TotalSamplesCount int `json:"totalSamplesCount,omitempty" protobuf:"bytes,6,opt,name=totalSamplesCount"`
}

func init() {
	SchemeBuilder.Register(&Checkpoint{}, &CheckpointList{})
}
