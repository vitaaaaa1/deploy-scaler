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
	"github.com/vita/deploy-scaler/internal/trigger"
)

// updateLiveWindowAnnotation updates annotation only when values differ
func (r *ScalerReconciler) updateLiveWindowAnnotation(ctx context.Context, scaler *apiv1alpha1.Scaler, evalResult *trigger.EvaluationResult) error {
	if !evalResult.NeedsAnnotationUpdate {
		return nil
	}

	currentStart := scaler.Annotations[trigger.AnnotationLiveStart]
	currentEnd := scaler.Annotations[trigger.AnnotationLiveEnd]

	if currentStart == evalResult.LiveStartTime && currentEnd == evalResult.LiveEndTime {
		logf.FromContext(ctx).Info("Live window annotation unchanged, skipping update")
		return nil
	}

	if scaler.Annotations == nil {
		scaler.Annotations = make(map[string]string)
	}

	scaler.Annotations[trigger.AnnotationLiveStart] = evalResult.LiveStartTime
	scaler.Annotations[trigger.AnnotationLiveEnd] = evalResult.LiveEndTime
	logf.FromContext(ctx).Info("Updated live window annotation",
		"start", evalResult.LiveStartTime,
		"end", evalResult.LiveEndTime)

	return r.Update(ctx, scaler)
}
