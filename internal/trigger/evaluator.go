package trigger

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/vita/deploy-scaler/internal/utils"

	"github.com/vita/deploy-scaler/internal/prometheus"

	apiv1alpha1 "github.com/vita/deploy-scaler/api/v1alpha1"
)

const (
	AnnotationLiveStart = "scaler.com/live-start"
	AnnotationLiveEnd   = "scaler.com/live-end"
)

// Evaluator evaluates triggers and determines which one should be active
type Evaluator struct {
	promClient *prometheus.Client
	logger     logr.Logger
}

// NewEvaluator creates a new trigger evaluator
func NewEvaluator(promClient *prometheus.Client, logger logr.Logger) *Evaluator {
	return &Evaluator{
		promClient: promClient,
		logger:     logger.WithName("trigger-evaluator"),
	}
}

// EvaluationResult contains the result of trigger evaluation
type EvaluationResult struct {
	ActiveTriggerName     string
	LiveStartTime         string
	LiveEndTime           string
	NeedsAnnotationUpdate bool
	NextCronTime          string
}

// Evaluate checks all triggers and returns the highest priority active one
func (e *Evaluator) Evaluate(ctx context.Context, scaler *apiv1alpha1.Scaler) (*EvaluationResult, error) {
	result := &EvaluationResult{}

	// Priority 1: Check Prometheus triggers (查找新的直播计划)
	// Prometheus 只负责提供时间窗口，不直接决定是否扩容
	for _, trigger := range scaler.Spec.Triggers {
		if trigger.Type == "prometheus" && trigger.Prometheus != nil {
			startTime, endTime, err := e.evaluatePromTrigger(ctx, &trigger)
			if err != nil {
				e.logger.Error(err, "Failed to evaluate Prometheus trigger", "trigger", trigger.Name)
				continue
			}

			if startTime != "" && endTime != "" {
				// Prometheus 查到指标 → 写入 Annotation（等下次 Reconcile 才生效扩容）
				e.logger.Info("Prometheus found live window, will update annotation",
					"trigger", trigger.Name,
					"startTime", startTime,
					"endTime", endTime)
				result.LiveStartTime = startTime
				result.LiveEndTime = endTime
				result.NeedsAnnotationUpdate = true
				// 不设置 ActiveTriggerName，不当场扩容
				return result, nil
			}
		}
	}

	// Priority 2: Check Cron triggers
	for _, trigger := range scaler.Spec.Triggers {
		if trigger.Type == "cron" && trigger.Cron != nil {

			e.logger.Info("Into Cron triggers") // DEBUG

			if e.evaluateCronTrigger(&trigger) {
				result.ActiveTriggerName = trigger.Name
				e.logger.Info("Into Cron evaluateCronTrigger")                          // DEBUG
				e.logger.Info("Cron trigger activated", "result", utils.ToJSON(result)) //DEBUG
				return result, nil
			}
		}
	}

	// No trigger active
	e.logger.Info("No triggers active, will restore to default")
	return result, nil
}

//func toJSON(result *EvaluationResult) string {
//	b, _ := json.Marshal(result)
//	return string(b)
//}

// evaluatePromTrigger queries Prometheus and extracts live time windows
func (e *Evaluator) evaluatePromTrigger(ctx context.Context, trigger *apiv1alpha1.Trigger) (string, string, error) {
	if trigger.Prometheus == nil {
		return "", "", nil
	}

	// Build PromQL
	labelStr := ""
	for k, v := range trigger.Prometheus.Labels {
		if labelStr != "" {
			labelStr += ","
		}
		labelStr += fmt.Sprintf("%s=\"%s\"", k, v)
	}

	var query string
	if labelStr != "" {
		query = fmt.Sprintf("%s{%s} == 1", trigger.Prometheus.Metric, labelStr)
	} else {
		query = fmt.Sprintf("%s == 1", trigger.Prometheus.Metric)
	}

	result, err := e.promClient.Query(ctx, query)
	if err != nil {
		return "", "", fmt.Errorf("query prometheus: %w", err)
	}

	if len(result.Data.Result) == 0 {
		return "", "", nil
	}

	// Extract time windows from metric labels
	startTime := ""
	endTime := ""
	for _, r := range result.Data.Result {
		if t, ok := r.Metric["liveStartTimeStr"]; ok {
			startTime = t
		}
		if t, ok := r.Metric["liveEndTimeStr"]; ok {
			endTime = t
		}
	}

	if startTime == "" || endTime == "" {
		return "", "", fmt.Errorf("liveStartTimeStr or liveEndTimeStr label not found in metric")
	}

	return startTime, endTime, nil
}

// evaluateCronTrigger checks if current time is within the Cron trigger window
// Schedule format: standard cron like "0 6 * * *" (at 06:00 every day)
func (e *Evaluator) evaluateCronTrigger(trigger *apiv1alpha1.Trigger) bool {
	if trigger.Cron == nil {
		return false
	}

	startHour, startMin, err := parseCronSchedule(trigger.Cron.StartSchedule)
	if err != nil {
		e.logger.Error(err, "Failed to parse cron start schedule", "schedule", trigger.Cron.StartSchedule)
		return false
	}

	endHour, endMin, err := parseCronSchedule(trigger.Cron.EndSchedule)
	if err != nil {
		e.logger.Error(err, "Failed to parse cron end schedule", "schedule", trigger.Cron.EndSchedule)
		return false
	}

	now := currentTime()
	nowMin := now.Hour()*60 + now.Minute()
	startMinTotal := startHour*60 + startMin
	endMinTotal := endHour*60 + endMin

	// 跨天情况：end 在 start 之前（如 22:00 ~ 06:00）
	if endMinTotal <= startMinTotal {
		return nowMin >= startMinTotal || nowMin < endMinTotal
	}

	// 正常情况：now 在 [start, end) 内
	return nowMin >= startMinTotal && nowMin < endMinTotal
}

// IsCronActive checks if a Cron trigger is currently active
func (e *Evaluator) IsCronTriggerActive(trigger *apiv1alpha1.Trigger) bool {
	return e.evaluateCronTrigger(trigger)
}

// parseCronSchedule extracts hour and minute from a standard cron expression like "0 6 * * *"
func parseCronSchedule(schedule string) (hour, minute int, err error) {
	// Standard cron: minute hour day month weekday
	// e.g., "0 6 * * *" → minute=0, hour=6
	var min, h int
	_, err = fmt.Sscanf(schedule, "%d %d", &min, &h)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid cron schedule %q: %w", schedule, err)
	}
	if min < 0 || min > 59 || h < 0 || h > 23 {
		return 0, 0, fmt.Errorf("invalid cron schedule %q: minute=%d, hour=%d", schedule, min, h)
	}
	return h, min, nil
}

// Helper functions for time
var currentTime = func() time.Time {
	return time.Now()
}
