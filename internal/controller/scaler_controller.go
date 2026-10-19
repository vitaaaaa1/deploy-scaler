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
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	apiv1vita "github.com/vita/deploy-scaler/api/v1vita"
)

// logger is initialized from the controller-runtime log package
var logger = logf.Log.WithName("controller-scaler")

// 记录 deployment 原始信息
// key: deployment_name; value: deployment_info(namespace/replicas)
var originalDeploymentInfo = make(map[string]apiv1vita.DeploymentInfo)
var annotation = make(map[string]string)

// ScalerReconciler reconciles a Scaler object
type ScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func scaleDeployment(scaler *apiv1vita.Scaler, r *ScalerReconciler, ctx context.Context, replicas int32) error {
	// 从 scaled instance 遍历 deployment(map)
	for _, deploy := range scaler.Spec.Deployments {
		// create new deployment instance
		deployment := &appsv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      deploy.Name,
			Namespace: scaler.Namespace,
		}, deployment) // 将查询结构赋值给 deployment
		if err != nil {
			return err
		}

		// 判断当前k8s集群中的Deployment replicas 是否等于 scaler 中指定的 replicas
		// 1. 如果不等于
		if *deployment.Spec.Replicas != replicas {
			deployment.Spec.Replicas = &replicas
			err := r.Update(ctx, deployment)
			if err != nil {
				scaler.Status.Status = apiv1vita.FAILED
				r.Status().Update(ctx, scaler)
				return err
			}
			scaler.Status.Status = apiv1vita.SCALED
			r.Status().Update(ctx, scaler)
			logger.Info(fmt.Sprintf("Successful Scaled Deployment: %s,scaled to: %d", deploy.Name, replicas))

		} else { // 2. 如果等于

		}
	}
	return nil
}

// +kubebuilder:rbac:groups=api.scaler.com,resources=scalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.scaler.com,resources=scalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=api.scaler.com,resources=scalers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Scaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.25.0/pkg/reconcile
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
	 req ctrl.Request 的作用
	    req 是 controller-runtime 传递给你的工作单元标识，告诉控制器需要处理哪个资源。
	    结构体定义
	       type Request struct {
	           // 资源的 namespace
	           Namespace string

	           // 资源的 name
	           Name string

	           // 内部字段，通常不需要直接访问
	           NamespacedName types.NamespacedName
	       }
	*/
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here
	log := logger.WithValues("Request.Namepsace", req.Namespace, "Request.Name", req.Name)
	log.Info("Reconcile Called")

	// Create Scaled instance
	scaler := &apiv1vita.Scaler{}
	err := r.Get(ctx, req.NamespacedName, scaler)
	if err != nil {
		// 如果没有 scaled 就忽略错误，使用 client.IgnoreNotFound ，使进程不中断
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	startTime := scaler.Spec.Start
	endTime := scaler.Spec.End
	replicas := scaler.Spec.Replicas

	currentHour := time.Now().Local().Hour()
	log.Info(fmt.Sprintf("currentTime: %d", currentHour))

	// 如果 scaler 状态为空
	if scaler.Status.Status == "" {
		scaler.Status.Status = apiv1vita.PENDING
		err := r.Status().Update(ctx, scaler)
		if err != nil {
			return ctrl.Result{}, err
		}

		// 将 scaler 中管理的 deployment 的 replica/namespace 加入到 annotaions 中
		err = addAnnotations(scaler, r, ctx)
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// if in startTime ~ endTime
	if currentHour >= startTime && currentHour < endTime {

		if scaler.Status.Status != apiv1vita.SCALED {
			// 从 scaled instance 遍历 deployment(map)
			log.Info("Currently within the scheduling period")
			err := scaleDeployment(scaler, r, ctx, replicas)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if scaler.Status.Status == apiv1vita.SCALED {
			err := restoreDeployment(scaler, r, ctx)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{
		// 10 秒执行一次 Reconcile 函数
		RequeueAfter: time.Duration(10 * time.Second),
	}, nil
}

func restoreDeployment(scaler *apiv1vita.Scaler, r *ScalerReconciler, ctx context.Context) error {
	logger.Info("starting to return the original state")
	for name, originalDeployInfo := range scaler.Annotations {

		if !strings.HasPrefix(name, "scaler.com/original") {
			continue
		}

		// "scaler.com/original-my-app" -> "my-app"
		realName := strings.TrimPrefix(name, "scaler.com/original-")

		var info apiv1vita.DeploymentInfo
		err := json.Unmarshal([]byte(originalDeployInfo), &info)
		if err != nil {
			return err
		}

		deployment := &appsv1.Deployment{}
		if err := r.Get(
			ctx, types.NamespacedName{
				Name:      realName,
				Namespace: scaler.Namespace,
			}, deployment,
		); err != nil {
			return err
		}

		// 	如果当前 deployment replicas 不等于 annotations 中记录的 original replicas 就执行回滚操作
		if *deployment.Spec.Replicas != info.Replicas {
			*deployment.Spec.Replicas = info.Replicas
			err := r.Update(ctx, deployment)
			if err != nil {
				return err
			}
		}
	}

	//
	scaler.Status.Status = apiv1vita.RESTORE
	err := r.Update(ctx, scaler)
	if err != nil {
		return err
	}

	return nil
}

func addAnnotations(scaler *apiv1vita.Scaler, r *ScalerReconciler, ctx context.Context) error {

	logger.Info("into addAnnotations")

	// 记录 deployments 原始 replica/namespace
	for _, deploy := range scaler.Spec.Deployments {
		deployment := &appsv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      deploy.Name,
			Namespace: scaler.Namespace,
		}, deployment)
		if err != nil {
			return err
		}

		// 开始记录
		logger.Info("add origninal state to originalDeploymentInfo map")
		originalDeploymentInfo[deploy.Name] = apiv1vita.DeploymentInfo{
			Replicas: *deployment.Spec.Replicas,
		}
		// 将原始记录加入 Annotations
		for deploymentName, info := range originalDeploymentInfo {
			// 将 info 转换为 json
			infoJson, err := json.Marshal(info)
			if err != nil {
				return err
			}

			// annotation 标识添加前缀，防止冲突
			key := fmt.Sprintf("scaler.com/original-%s", deploymentName)
			annotation[key] = string(infoJson)
		}

		// 更新 scaler annotations
		scaler.ObjectMeta.Annotations = annotation
		err = r.Update(ctx, scaler)
		if err != nil {
			return err
		}

	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1vita.Scaler{}).
		Named("scaler").
		Complete(r)
}
