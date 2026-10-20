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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScaleTargetRef defines the target resource to scale
type ScaleTargetRef struct {
	// APIVersion of the target resource (e.g., autoscaling/v2 or apps/v1)
	// +kubebuilder:validation:Required
	APIVersion string `json:"apiVersion"`
	// Kind of the target resource (HorizontalPodAutoscaler or Deployment)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HorizontalPodAutoscaler;Deployment
	Kind string `json:"kind"`
	// Name of the target resource
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// ScalingConfig defines the scaling configuration for a trigger
type ScalingConfig struct {
	// MinReplicas is the target replica count (for Deployment) or minReplicas (for HPA)
	// +kubebuilder:validation:Minimum=0
	MinReplicas int32 `json:"minReplicas"`
}

// PromTrigger defines a Prometheus-based trigger
type PromTrigger struct {
	// Metric name to query
	// +kubebuilder:validation:Required
	Metric string `json:"metric"`
	// Labels to filter the metric query
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// CronTrigger defines a Cron-based trigger
type CronTrigger struct {
	// Schedule when the trigger becomes active (start time)
	// +kubebuilder:validation:Required
	StartSchedule string `json:"startSchedule"`
	// Schedule when the trigger should stop being active (end time)
	// +kubebuilder:validation:Required
	EndSchedule string `json:"endSchedule"`
}

// Trigger defines a scaling trigger (Prometheus or Cron)
type Trigger struct {
	// Type of the trigger (prometheus or cron)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=prometheus;cron
	Type string `json:"type"`
	// Name of the trigger (used for status tracking)
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Prometheus configuration (if type is prometheus)
	// +optional
	Prometheus *PromTrigger `json:"prometheus,omitempty"`
	// Cron configuration (if type is cron)
	// +optional
	Cron *CronTrigger `json:"cron,omitempty"`
	// Scaling configuration for this trigger
	// +kubebuilder:validation:Required
	Scaling ScalingConfig `json:"scaling"`
}

// LiveSession defines the current live streaming session state
type LiveSession struct {
	// StartTime is the start timestamp of the live session (milliseconds)
	StartTime string `json:"startTime"`
	// EndTime is the end timestamp of the live session (milliseconds)
	EndTime string `json:"endTime"`
	// State is always "Active" (cleared when session ends)
	// +kubebuilder:validation:Enum=Active
	State string `json:"state"`
}

// ScalerSpec defines the desired state of Scaler
type ScalerSpec struct {
	// ScaleTargetRef is the target resource to scale (HPA or Deployment)
	// +kubebuilder:validation:Required
	ScaleTargetRef ScaleTargetRef `json:"scaleTargetRef"`

	// Restore is the default scaling configuration when no trigger is active
	// +kubebuilder:validation:Required
	Restore ScalingConfig `json:"restore"`

	// Triggers is a list of scaling triggers (prometheus and/or cron)
	// +kubebuilder:validation:Required
	// +listType=map
	// +listMapKey=name
	Triggers []Trigger `json:"triggers"`
}

// ScalerStatus defines the observed state of Scaler.
type ScalerStatus struct {
	// ActiveTrigger is the name of the currently active trigger
	// +optional
	ActiveTrigger string `json:"activeTrigger,omitempty"`

	// ScalingApplied indicates whether the current trigger's scaling config has been applied
	// +optional
	ScalingApplied bool `json:"scalingApplied,omitempty"`

	// LiveSession contains the current live session information (if active)
	// +optional
	LiveSession *LiveSession `json:"liveSession,omitempty"`

	// NextCronTrigger is the next scheduled cron trigger time
	// +optional
	NextCronTrigger string `json:"nextCronTrigger,omitempty"`

	// Conditions represent the current state of the Scaler resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ActiveTrigger",type=string,JSONPath=`.status.activeTrigger`
// +kubebuilder:printcolumn:name="LiveState",type=string,JSONPath=`.status.liveSession.state`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// printcolumn 用于在 kubectl get scalers 输出时显示自定义列
// ActiveTrigger: 当前激活的触发器名称
// LiveState: 直播会话状态 (Active/Ended)
// Age: 资源创建时间

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
