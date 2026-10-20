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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	apiv1alpha1 "github.com/vita/deploy-scaler/api/v1alpha1"
	"github.com/vita/deploy-scaler/internal/prometheus"
	"github.com/vita/deploy-scaler/internal/scaling"
	"github.com/vita/deploy-scaler/internal/trigger"
)

const (
	RequeueInterval = 30 * time.Second
)

// ScalerReconciler reconciles a Scaler object
type ScalerReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	PromClient  *prometheus.Client
	Scaler      *scaling.Scaler
	TriggerEval *trigger.Evaluator
}

// +kubebuilder:rbac:groups=api.scaler.com,resources=scalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.scaler.com,resources=scalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=api.scaler.com,resources=scalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

// Reconcile 核心流程:
// 1. 评估触发器 (查 Prometheus / Cron)
// 2. 如果查到直播计划 → 写入 Scaler Annotation (持久化时间窗口)
// 3. 检查 Annotation 中的时间窗口 → 决定扩缩容
// 4. 直播结束后 → 清理 Annotation，恢复默认值
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	scaler, err := r.fetchScaler(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if scaler == nil {
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling Scaler", "name", scaler.Name)

	// 第 1 步：评估触发器
	evalResult, err := r.evaluateTriggers(ctx, scaler)
	if err != nil {
		log.Error(err, "Failed to evaluate triggers")
		return ctrl.Result{RequeueAfter: RequeueInterval}, nil
	}

	// 第 2 步：更新 Annotation (值变化时才写)
	if err := r.updateLiveWindowAnnotation(ctx, scaler, evalResult); err != nil {
		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}

	// 第 3 步：根据时间窗口决定扩缩容配置
	decision := r.resolveScalingConfig(ctx, scaler)

	// 第 4 步：执行扩缩容 (状态变化时才调用)
	if err := r.applyScalingIfNeeded(ctx, scaler, decision); err != nil {
		return ctrl.Result{}, err
	}

	// 第 5 步：清理脏 Annotation + 更新 Status
	if err := r.finalizeReconcile(ctx, scaler, decision); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: RequeueInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Scaler{}).
		Named("scaler").
		Complete(r)
}
