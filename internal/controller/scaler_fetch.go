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

	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1alpha1 "github.com/vita/deploy-scaler/api/v1alpha1"
	"github.com/vita/deploy-scaler/internal/trigger"
)

// fetchScaler retrieves the Scaler CR, returns nil if not found
func (r *ScalerReconciler) fetchScaler(ctx context.Context, namespacedName client.ObjectKey) (*apiv1alpha1.Scaler, error) {
	scaler := &apiv1alpha1.Scaler{}
	if err := r.Get(ctx, namespacedName, scaler); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return scaler, nil
}

// evaluateTriggers calls the trigger evaluator
func (r *ScalerReconciler) evaluateTriggers(ctx context.Context, scaler *apiv1alpha1.Scaler) (*trigger.EvaluationResult, error) {
	return r.TriggerEval.Evaluate(ctx, scaler)
}
