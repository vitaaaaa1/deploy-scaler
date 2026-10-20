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
	"strconv"
	"time"

	"github.com/vita/deploy-scaler/internal/utils"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	apiv1alpha1 "github.com/vita/deploy-scaler/api/v1alpha1"
	"github.com/vita/deploy-scaler/internal/trigger"
)

// scalingDecision holds the result of scaling configuration resolution
type scalingDecision struct {
	Config                 apiv1alpha1.ScalingConfig
	EffectiveTrigger       string
	NeedsStatusUpdate      bool
	NeedsAnnotationCleanup bool
}

// resolveScalingConfig determines the scaling config based on live window and cron
func (r *ScalerReconciler) resolveScalingConfig(ctx context.Context, scaler *apiv1alpha1.Scaler) scalingDecision {
	log := logf.FromContext(ctx)
	decision := scalingDecision{}

	currentTimeMs := time.Now().UnixMilli()
	startStr, hasStart := scaler.Annotations[trigger.AnnotationLiveStart]
	endStr, hasEnd := scaler.Annotations[trigger.AnnotationLiveEnd]

	if hasStart && hasEnd {
		decision = r.resolveLiveWindow(ctx, scaler, startStr, endStr, currentTimeMs)
	} else {

		log.Info("Into resolveScalingConfig not hasStart/hasEnd")
		decision = r.resolveDefaultOrCron(ctx, scaler)
	}

	// 清理旧的 LiveSession (没有直播窗口或直播已结束)
	if !hasStart || !hasEnd || decision.NeedsAnnotationCleanup {
		if scaler.Status.LiveSession != nil {
			scaler.Status.LiveSession = nil
			decision.NeedsStatusUpdate = true
		}
	}

	// 设置 LiveSession (直播窗口内)
	if decision.EffectiveTrigger == "prometheus" {
		startMs, _ := strconv.ParseInt(startStr, 10, 64)
		endMs, _ := strconv.ParseInt(endStr, 10, 64)
		if startMs > 0 && endMs > 0 && currentTimeMs >= startMs && currentTimeMs < endMs {
			scaler.Status.LiveSession = &apiv1alpha1.LiveSession{
				StartTime: startStr,
				EndTime:   endStr,
				State:     "Active",
			}
			decision.NeedsStatusUpdate = true
		}
	}

	log.Info("Scaling decision resolved",
		"effectiveTrigger", decision.EffectiveTrigger,
		"minReplicas", decision.Config.MinReplicas,
		"needsCleanup", decision.NeedsAnnotationCleanup)

	return decision
}

// resolveLiveWindow handles the case when live window annotation exists
func (r *ScalerReconciler) resolveLiveWindow(ctx context.Context, scaler *apiv1alpha1.Scaler, startStr, endStr string, currentTimeMs int64) scalingDecision {
	log := logf.FromContext(ctx)
	decision := scalingDecision{}

	startMs, parseErr := strconv.ParseInt(startStr, 10, 64)
	if parseErr != nil {
		log.Error(parseErr, "Invalid live start time, will clear annotation", "start", startStr)
		decision.NeedsAnnotationCleanup = true
		return decision
	}

	endMs, parseErr := strconv.ParseInt(endStr, 10, 64)
	if parseErr != nil {
		log.Error(parseErr, "Invalid live end time, will clear annotation", "end", endStr)
		decision.NeedsAnnotationCleanup = true
		return decision
	}

	if currentTimeMs >= startMs && currentTimeMs < endMs {
		// 当前时间在直播窗口内 → 用 Prometheus trigger 的配置
		for _, t := range scaler.Spec.Triggers {
			if t.Type == "prometheus" {
				decision.Config = t.Scaling
				decision.EffectiveTrigger = t.Name
				break
			}
		}
	} else {
		// 不在直播窗口内 → 检查 Cron
		for _, t := range scaler.Spec.Triggers {
			if t.Type == "cron" && r.TriggerEval.IsCronTriggerActive(&t) {
				decision.Config = t.Scaling
				decision.EffectiveTrigger = t.Name
				break
			}
		}
		if decision.EffectiveTrigger == "" {
			decision.Config = scaler.Spec.Restore
		}
	}

	return decision
}

// resolveDefaultOrCron handles the case when no live window annotation exists
func (r *ScalerReconciler) resolveDefaultOrCron(ctx context.Context, scaler *apiv1alpha1.Scaler) scalingDecision {
	decision := scalingDecision{}

	log := logf.FromContext(ctx) // DEBUG

	for _, t := range scaler.Spec.Triggers {
		if t.Type == "cron" && r.TriggerEval.IsCronTriggerActive(&t) {
			decision.Config = t.Scaling
			decision.EffectiveTrigger = t.Name

			log.Info("Into resolveDefaultOrCron", "result", utils.ToJSON(decision)) // DEBUG

			break
		}
	}
	if decision.EffectiveTrigger == "" {
		decision.Config = scaler.Spec.Restore
	}

	return decision
}
