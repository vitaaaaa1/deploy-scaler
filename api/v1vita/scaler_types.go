/*
Copyright 2026.

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

package v1vita

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	FAILED = "Failed"

	SCALED  = "scaled"
	PENDING = "pending"
	RESTORE = "restore"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type DeploymentInfo struct { // 记录初始副本数
	Replicas  int32  `json:"replicas"`
	Namespace string `json:"namespace"`
}

type NamespaceName struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ScalerSpec defines the desired state of Scaler
type ScalerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of Scaler. Edit scaler_types.go to remove/update
	// +optional
	Foo *string `json:"foo,omitempty"`

	// +kubebuilder:validation:Maximum=23
	// +kubebuilder:validation:Minimum=0
	Start int `json:"start"`

	// +kubebuilder:validation:Maximum=23
	// +kubebuilder:validation:Minimum=0
	End         int             `json:"end"`
	Replicas    int32           `json:"replicas"`
	Deployments []NamespaceName `json:"deployments"`
}

// ScalerStatus defines the observed state of Scaler.
type ScalerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Scaler resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	Status string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// printcolumn 用于在 kubectl get scalers 输出时额外显示自定义列
// 注意：添加 printcolumn 会覆盖默认的 NAME 和 AGE 列，需要显式定义
// name="Status": 从 .status.status 字段提取值
// name="Age": 从 .metadata.creationTimestamp 提取创建时间

// Scaler is the Schema for the scalers API
type Scaler struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Scaler
	// +required
	Spec ScalerSpec `json:"spec"`

	// status defines the observed state of Scaler
	// +optional
	Status ScalerStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ScalerList contains a list of Scaler
type ScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Scaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Scaler{}, &ScalerList{})
		return nil
	})
}
