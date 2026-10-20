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

// finalizeReconcile cleans up invalid annotations and updates status
func (r *ScalerReconciler) finalizeReconcile(ctx context.Context, scaler *apiv1alpha1.Scaler, decision *scalingDecision) error {
	log := logf.FromContext(ctx)

	if decision.NeedsAnnotationCleanup {
		delete(scaler.Annotations, trigger.AnnotationLiveStart)
		delete(scaler.Annotations, trigger.AnnotationLiveEnd)
		if err := r.Update(ctx, scaler); err != nil {
			log.Error(err, "Failed to clear invalid live window annotation")
			return err
		}
		log.Info("Cleared invalid live window annotation")
	}

	if decision.NeedsStatusUpdate {
		if err := r.Status().Update(ctx, scaler); err != nil {
			log.Error(err, "Failed to update Scaler status")
			return err
		}
	}

	return nil
}
