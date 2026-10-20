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

package controller

import (
	"context"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	apiv1alpha1 "github.com/vita/deploy-scaler/api/v1alpha1"
)

// applyScalingIfNeeded applies scaling only when trigger state changed
func (r *ScalerReconciler) applyScalingIfNeeded(ctx context.Context, scaler *apiv1alpha1.Scaler, decision scalingDecision) error {
	log := logf.FromContext(ctx)

	// Annotation 解析失败时跳过扩缩容，避免影响正在运行的服务
	if decision.NeedsAnnotationCleanup {
		log.Info("Skipping scaling due to invalid annotation, will clear annotation first")
		return nil
	}

	if scaler.Status.ActiveTrigger != decision.EffectiveTrigger || !scaler.Status.ScalingApplied {
		if err := r.Scaler.ApplyScaling(ctx, scaler.Spec.ScaleTargetRef, scaler.Namespace, decision.Config); err != nil {
			log.Error(err, "Failed to apply scaling")
			return err
		}

		scaler.Status.ActiveTrigger = decision.EffectiveTrigger
		scaler.Status.ScalingApplied = true
		decision.NeedsStatusUpdate = true

		log.Info("Applied scaling configuration",
			"target", scaler.Spec.ScaleTargetRef.Name,
			"effectiveTrigger", decision.EffectiveTrigger,
			"minReplicas", decision.Config.MinReplicas)
	} else {
		log.Info("Scaling already applied, skipping (HPA in control)")
	}

	return nil
}
