package collector

import (
	"context"
	"fmt"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/internal/models"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// k8sCollector handles the actual collection of metrics from Kubernetes
type k8sCollector struct {
	client *k8s.Client
}

// newK8sCollector creates a new Kubernetes metrics collector
func newK8sCollector(client *k8s.Client) *k8sCollector {
	return &k8sCollector{
		client: client,
	}
}

// CollectPodMetrics collects current pod metrics for a namespace
func (c *k8sCollector) CollectPodMetrics(namespace string) ([]models.PodMetrics, error) {
	ctx := context.Background()

	// Get pod metrics from metrics API
	podMetricsList, err := c.client.MetricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
	}

	var metrics []models.PodMetrics

	for _, podMetrics := range podMetricsList.Items {
		// Sum up resources across all containers in the pod
		var totalCPU int64
		var totalMemory int64

		for _, container := range podMetrics.Containers {
			// CPU is in nanocores, convert to millicores
			cpuNano := container.Usage.Cpu().MilliValue()
			totalCPU += cpuNano

			// Memory is in bytes
			memBytes := container.Usage.Memory().Value()
			totalMemory += memBytes
		}

		metrics = append(metrics, models.PodMetrics{
			Name:      podMetrics.Name,
			Namespace: podMetrics.Namespace,
			CPU:       totalCPU,
			Memory:    totalMemory,
			Timestamp: podMetrics.Timestamp.Time,
		})
	}

	return metrics, nil
}

// CollectNodeMetrics collects current node metrics
func (c *k8sCollector) CollectNodeMetrics() ([]models.NodeMetrics, error) {
	ctx := context.Background()

	// Get node metrics from metrics API
	nodeMetricsList, err := c.client.MetricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %w", err)
	}

	var metrics []models.NodeMetrics

	for _, nodeMetrics := range nodeMetricsList.Items {
		// CPU is in nanocores, convert to millicores
		cpuNano := nodeMetrics.Usage.Cpu().MilliValue()

		// Memory is in bytes
		memBytes := nodeMetrics.Usage.Memory().Value()

		metrics = append(metrics, models.NodeMetrics{
			Name:      nodeMetrics.Name,
			CPU:       cpuNano,
			Memory:    memBytes,
			Timestamp: nodeMetrics.Timestamp.Time,
		})
	}

	return metrics, nil
}

// CollectHPAMetrics collects HPA metrics for a namespace
func (c *k8sCollector) CollectHPAMetrics(namespace string) ([]models.HPAMetrics, error) {
	ctx := context.Background()

	// Get HPAs from the autoscaling API
	hpaList, err := c.client.Clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get HPAs: %w", err)
	}

	var metrics []models.HPAMetrics

	for _, hpa := range hpaList.Items {
		hpaMetric := models.HPAMetrics{
			Name:            hpa.Name,
			Namespace:       hpa.Namespace,
			CurrentReplicas: hpa.Status.CurrentReplicas,
			DesiredReplicas: hpa.Status.DesiredReplicas,
			MinReplicas:     *hpa.Spec.MinReplicas,
			MaxReplicas:     hpa.Spec.MaxReplicas,
			Timestamp:       hpa.Status.LastScaleTime.Time,
		}

		// Extract CPU target and current values from metrics
		for _, metric := range hpa.Spec.Metrics {
			if metric.Type == autoscalingv2.ResourceMetricSourceType &&
				metric.Resource != nil &&
				metric.Resource.Name == "cpu" &&
				metric.Resource.Target.AverageUtilization != nil {
				hpaMetric.TargetCPU = *metric.Resource.Target.AverageUtilization
				break
			}
		}

		for _, currentMetric := range hpa.Status.CurrentMetrics {
			if currentMetric.Type == autoscalingv2.ResourceMetricSourceType &&
				currentMetric.Resource != nil &&
				currentMetric.Resource.Name == "cpu" &&
				currentMetric.Resource.Current.AverageUtilization != nil {
				hpaMetric.CurrentCPU = *currentMetric.Resource.Current.AverageUtilization
				break
			}
		}

		metrics = append(metrics, hpaMetric)
	}

	return metrics, nil
}
