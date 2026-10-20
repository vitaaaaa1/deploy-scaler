package scaling

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1alpha1 "github.com/vita/deploy-scaler/api/v1alpha1"
)

// Scaler handles scaling operations for HPA and Deployment
type Scaler struct {
	client.Client
}

// NewScaler creates a new Scaler
func NewScaler(c client.Client) *Scaler {
	return &Scaler{Client: c}
}

// ApplyScaling applies scaling configuration based on ScaleTargetRef
func (s *Scaler) ApplyScaling(ctx context.Context, targetRef apiv1alpha1.ScaleTargetRef, namespace string, config apiv1alpha1.ScalingConfig) error {
	switch targetRef.Kind {
	case "HorizontalPodAutoscaler":
		return s.scaleHPA(ctx, targetRef, namespace, config)
	case "Deployment":
		return s.scaleDeployment(ctx, targetRef, namespace, config)
	default:
		return fmt.Errorf("unsupported target kind: %s", targetRef.Kind)
	}
}

// scaleHPA updates HPA minReplicas only (maxReplicas is managed in HPA spec)
func (s *Scaler) scaleHPA(ctx context.Context, targetRef apiv1alpha1.ScaleTargetRef, namespace string, config apiv1alpha1.ScalingConfig) error {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := s.Get(ctx, types.NamespacedName{
		Name:      targetRef.Name,
		Namespace: namespace,
	}, hpa); err != nil {
		return fmt.Errorf("get HPA: %w", err)
	}

	// Check if update is needed
	if hpa.Spec.MinReplicas != nil && *hpa.Spec.MinReplicas == config.MinReplicas {
		return nil
	}

	// Update HPA minReplicas only
	hpa.Spec.MinReplicas = &config.MinReplicas

	if err := s.Update(ctx, hpa); err != nil {
		return fmt.Errorf("update HPA: %w", err)
	}

	return nil
}

// scaleDeployment updates Deployment replicas
func (s *Scaler) scaleDeployment(ctx context.Context, targetRef apiv1alpha1.ScaleTargetRef, namespace string, config apiv1alpha1.ScalingConfig) error {
	deploy := &appsv1.Deployment{}
	if err := s.Get(ctx, types.NamespacedName{
		Name:      targetRef.Name,
		Namespace: namespace,
	}, deploy); err != nil {
		return fmt.Errorf("get Deployment: %w", err)
	}

	// Check if update is needed
	if deploy.Spec.Replicas != nil && *deploy.Spec.Replicas == config.MinReplicas {
		return nil
	}

	// Update Deployment replicas
	deploy.Spec.Replicas = &config.MinReplicas

	if err := s.Update(ctx, deploy); err != nil {
		return fmt.Errorf("update Deployment: %w", err)
	}

	return nil
}
