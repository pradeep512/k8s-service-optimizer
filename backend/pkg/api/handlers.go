package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/k8s-service-optimizer/backend/internal/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondWithSuccess(w, HealthResponse{
		Status: "healthy",
	})
}

// handleReady handles the readiness check endpoint
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Try to collect node metrics to verify collector is working
	_, err := s.collector.CollectNodeMetrics()
	if err != nil {
		respondWithError(w, http.StatusServiceUnavailable, "NOT_READY", "Metrics collector is not responding")
		return
	}

	respondWithSuccess(w, ReadyResponse{
		Status:  "ready",
		Message: "All systems operational",
	})
}

// handleStatus handles the system status endpoint
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime)

	// Check if collector is running by trying to collect metrics
	_, err := s.collector.CollectNodeMetrics()
	collectorRunning := err == nil

	status := StatusResponse{
		Version:          "1.0.0",
		Uptime:           uptime.String(),
		CollectorRunning: collectorRunning,
		Timestamp:        time.Now(),
	}

	respondWithSuccess(w, status)
}

// handleClusterOverview handles the cluster overview endpoint
func (s *Server) handleClusterOverview(w http.ResponseWriter, r *http.Request) {
	// Get node metrics
	nodeMetrics, err := s.collector.CollectNodeMetrics()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "METRICS_ERROR", fmt.Sprintf("Failed to collect node metrics: %v", err))
		return
	}

	// Get all namespaces
	ctx := context.Background()
	namespaces, err := s.k8sClient.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "K8S_ERROR", fmt.Sprintf("Failed to list namespaces: %v", err))
		return
	}

	// Count healthy nodes
	healthyNodes := 0
	var totalCPUCapacity, totalCPUUsage, totalMemCapacity, totalMemUsage int64
	for _, node := range nodeMetrics {
		healthyNodes++ // Simplified: assume all nodes are healthy
		totalCPUUsage += node.CPU
		totalMemUsage += node.Memory
	}

	// Get node details for capacity
	nodes, err := s.k8sClient.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, node := range nodes.Items {
			cpu := node.Status.Capacity.Cpu()
			mem := node.Status.Capacity.Memory()
			if cpu != nil {
				totalCPUCapacity += cpu.MilliValue()
			}
			if mem != nil {
				totalMemCapacity += mem.Value()
			}
		}
	}

	// Count total pods across all namespaces
	totalPods := 0
	healthyPods := 0
	for _, ns := range namespaces.Items {
		pods, err := s.k8sClient.Clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if err == nil {
			totalPods += len(pods.Items)
			for _, pod := range pods.Items {
				if pod.Status.Phase == "Running" {
					healthyPods++
				}
			}
		}
	}

	// Build namespace list
	namespaceList := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		namespaceList[i] = ns.Name
	}

	overview := models.ClusterOverview{
		TotalNodes:     len(nodeMetrics),
		HealthyNodes:   healthyNodes,
		TotalPods:      totalPods,
		HealthyPods:    healthyPods,
		CPUCapacity:    totalCPUCapacity,
		CPUUsage:       totalCPUUsage,
		MemoryCapacity: totalMemCapacity,
		MemoryUsage:    totalMemUsage,
		Namespaces:     namespaceList,
		Timestamp:      time.Now(),
	}

	respondWithSuccess(w, overview)
}

// handleListServices handles listing all services
func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get all namespaces
	namespaces, err := s.k8sClient.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "K8S_ERROR", fmt.Sprintf("Failed to list namespaces: %v", err))
		return
	}

	var allServices []map[string]interface{}

	// List services in each namespace
	for _, ns := range namespaces.Items {
		services, err := s.k8sClient.Clientset.CoreV1().Services(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Warning: failed to list services in namespace %s: %v", ns.Name, err)
			continue
		}

		for _, svc := range services.Items {
			serviceInfo := map[string]interface{}{
				"name":      svc.Name,
				"namespace": svc.Namespace,
				"type":      string(svc.Spec.Type),
				"clusterIP": svc.Spec.ClusterIP,
				"ports":     len(svc.Spec.Ports),
				"age":       time.Since(svc.CreationTimestamp.Time).String(),
			}
			allServices = append(allServices, serviceInfo)
		}
	}

	respondWithSuccess(w, allServices)
}

// handleServiceDetail handles getting detailed information about a specific service
func (s *Server) handleServiceDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	ctx := context.Background()

	// Get deployment (assuming service name matches deployment name)
	deployment, err := s.k8sClient.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "DEPLOYMENT_NOT_FOUND", fmt.Sprintf("Deployment not found: %v", err))
		return
	}

	// Get analysis from optimizer
	analysis, err := s.optimizer.AnalyzeDeployment(namespace, name)
	if err != nil {
		log.Printf("Warning: failed to analyze deployment %s/%s: %v", namespace, name, err)
		analysis = &models.Analysis{
			Namespace:  namespace,
			Deployment: name,
			Timestamp:  time.Now(),
		}
	}

	// Get traffic analysis
	traffic, err := s.analyzer.AnalyzeTrafficPatterns(namespace, name, 24*time.Hour)
	if err != nil {
		log.Printf("Warning: failed to analyze traffic for %s/%s: %v", namespace, name, err)
		traffic = &models.TrafficAnalysis{
			Service:   name,
			Namespace: namespace,
			Timestamp: time.Now(),
		}
	}

	// Get cost breakdown
	cost, err := s.analyzer.CalculateServiceCost(namespace, name)
	if err != nil {
		log.Printf("Warning: failed to calculate cost for %s/%s: %v", namespace, name, err)
		cost = &models.CostBreakdown{
			Service:   name,
			Namespace: namespace,
			Timestamp: time.Now(),
		}
	}

	// Get pods for this deployment
	pods, err := s.k8sClient.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", name),
	})

	var podInfos []models.PodInfo
	if err == nil {
		for _, pod := range pods.Items {
			restarts := int32(0)
			if len(pod.Status.ContainerStatuses) > 0 {
				restarts = pod.Status.ContainerStatuses[0].RestartCount
			}

			podInfo := models.PodInfo{
				Name:     pod.Name,
				Status:   string(pod.Status.Phase),
				Restarts: restarts,
				Age:      time.Since(pod.CreationTimestamp.Time),
				Node:     pod.Spec.NodeName,
			}
			podInfos = append(podInfos, podInfo)
		}
	}

	// Build service detail
	detail := models.ServiceDetail{
		Name:        name,
		Namespace:   namespace,
		Type:        "Deployment",
		Replicas:    *deployment.Spec.Replicas,
		HealthScore: analysis.HealthScore,
		CPUUsage:    analysis.CPUUsage,
		MemoryUsage: analysis.MemoryUsage,
		Traffic:     *traffic,
		Cost:        *cost,
		Pods:        podInfos,
		Timestamp:   time.Now(),
	}

	respondWithSuccess(w, detail)
}

// handleNodeMetrics handles getting node metrics
func (s *Server) handleNodeMetrics(w http.ResponseWriter, r *http.Request) {
	nodeMetrics, err := s.collector.CollectNodeMetrics()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "METRICS_ERROR", fmt.Sprintf("Failed to collect node metrics: %v", err))
		return
	}

	respondWithSuccess(w, nodeMetrics)
}

// handlePodMetrics handles getting pod metrics for a namespace
func (s *Server) handlePodMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	podMetrics, err := s.collector.CollectPodMetrics(namespace)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "METRICS_ERROR", fmt.Sprintf("Failed to collect pod metrics: %v", err))
		return
	}

	respondWithSuccess(w, podMetrics)
}

// handleHPAMetrics handles getting HPA metrics for a namespace
func (s *Server) handleHPAMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	hpaMetrics, err := s.collector.CollectHPAMetrics(namespace)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "METRICS_ERROR", fmt.Sprintf("Failed to collect HPA metrics: %v", err))
		return
	}

	respondWithSuccess(w, hpaMetrics)
}

// handleTimeSeries handles getting time series data
func (s *Server) handleTimeSeries(w http.ResponseWriter, r *http.Request) {
	params, err := parseTimeSeriesQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_PARAMS", fmt.Sprintf("Invalid query parameters: %v", err))
		return
	}

	if params.Resource == "" || params.Metric == "" {
		respondWithError(w, http.StatusBadRequest, "MISSING_PARAMS", "Resource and metric parameters are required")
		return
	}

	timeSeriesData, err := s.collector.GetTimeSeriesData(params.Resource, params.Metric, params.Duration)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "METRICS_ERROR", fmt.Sprintf("Failed to get time series data: %v", err))
		return
	}

	respondWithSuccess(w, timeSeriesData)
}

// handleRecommendations handles getting all recommendations
func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	recommendations, err := s.optimizer.GetAllRecommendations()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "OPTIMIZER_ERROR", fmt.Sprintf("Failed to get recommendations: %v", err))
		return
	}

	respondWithSuccess(w, recommendations)
}

// handleRecommendationByID handles getting a specific recommendation
func (s *Server) handleRecommendationByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get all recommendations and find the one with the matching ID
	recommendations, err := s.optimizer.GetAllRecommendations()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "OPTIMIZER_ERROR", fmt.Sprintf("Failed to get recommendations: %v", err))
		return
	}

	// Find recommendation by ID
	for _, rec := range recommendations {
		if rec.ID == id {
			respondWithSuccess(w, rec)
			return
		}
	}

	respondWithError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Recommendation not found: %s", id))
}

// handleApplyRecommendation handles applying a recommendation
func (s *Server) handleApplyRecommendation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := s.optimizer.ApplyRecommendation(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "APPLY_FAILED", fmt.Sprintf("Failed to apply recommendation: %v", err))
		return
	}

	response := ApplyRecommendationResponse{
		Status:  "applied",
		ID:      id,
		Message: "Recommendation applied successfully",
	}

	respondWithSuccess(w, response)
}

// handleAnalysis handles getting analysis for a specific service
func (s *Server) handleAnalysis(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	service := vars["service"]

	analysis, err := s.optimizer.AnalyzeDeployment(namespace, service)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ANALYSIS_ERROR", fmt.Sprintf("Failed to analyze service: %v", err))
		return
	}

	respondWithSuccess(w, analysis)
}

// handleTraffic handles getting traffic analysis for a specific service
func (s *Server) handleTraffic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	service := vars["service"]

	// Default to 24 hours
	duration := 24 * time.Hour
	if durationStr := r.URL.Query().Get("duration"); durationStr != "" {
		parsedDuration, err := time.ParseDuration(durationStr)
		if err == nil {
			duration = parsedDuration
		}
	}

	traffic, err := s.analyzer.AnalyzeTrafficPatterns(namespace, service, duration)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "TRAFFIC_ERROR", fmt.Sprintf("Failed to analyze traffic: %v", err))
		return
	}

	respondWithSuccess(w, traffic)
}

// handleCost handles getting cost breakdown for a specific service
func (s *Server) handleCost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	service := vars["service"]

	cost, err := s.analyzer.CalculateServiceCost(namespace, service)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "COST_ERROR", fmt.Sprintf("Failed to calculate cost: %v", err))
		return
	}

	respondWithSuccess(w, cost)
}

// handleAnomalies handles getting detected anomalies
func (s *Server) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	params, err := parseAnomalyQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_PARAMS", fmt.Sprintf("Invalid query parameters: %v", err))
		return
	}

	// If no resource specified, return empty array
	if params.Resource == "" {
		respondWithSuccess(w, []models.Anomaly{})
		return
	}

	// Default metric is "cpu"
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "cpu"
	}

	anomalies, err := s.analyzer.DetectAnomalies(params.Resource, metric, params.Duration)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ANOMALY_ERROR", fmt.Sprintf("Failed to detect anomalies: %v", err))
		return
	}

	respondWithSuccess(w, anomalies)
}

// handleListDeployments handles listing all deployments with metrics
func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get all namespaces
	namespaces, err := s.k8sClient.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "K8S_ERROR", fmt.Sprintf("Failed to list namespaces: %v", err))
		return
	}

	var allDeployments []map[string]interface{}

	for _, ns := range namespaces.Items {
		deployments, err := s.k8sClient.Clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Warning: failed to list deployments in namespace %s: %v", ns.Name, err)
			continue
		}

		for _, deploy := range deployments.Items {
			// Get pod metrics for this deployment
			podMetrics, _ := s.collector.CollectPodMetrics(ns.Name)

			var totalCPU, totalMemory int64
			var healthyPods int32

			// Match pods to deployment by label selector
			if deploy.Spec.Selector != nil && len(deploy.Spec.Selector.MatchLabels) > 0 {
				labelSelector := labels.SelectorFromSet(deploy.Spec.Selector.MatchLabels)
				pods, _ := s.k8sClient.Clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{
					LabelSelector: labelSelector.String(),
				})

				for _, pod := range pods.Items {
					if pod.Status.Phase == "Running" {
						healthyPods++
					}
					// Find metrics for this pod
					for _, pm := range podMetrics {
						if pm.Name == pod.Name {
							totalCPU += pm.CPU
							totalMemory += pm.Memory
						}
					}
				}
			}

			replicas := int32(0)
			if deploy.Spec.Replicas != nil {
				replicas = *deploy.Spec.Replicas
			}

			healthScore := 0.0
			if replicas > 0 {
				healthScore = (float64(healthyPods) / float64(replicas)) * 100
			}

			status := "healthy"
			if healthScore < 80 {
				status = "warning"
			}
			if healthScore < 50 {
				status = "critical"
			}
			if replicas == 0 {
				status = "unknown"
			}

			deploymentInfo := map[string]interface{}{
				"name":        deploy.Name,
				"namespace":   deploy.Namespace,
				"replicas":    replicas,
				"healthScore": healthScore,
				"cpuUsage":    float64(totalCPU) / 1000.0, // millicores to cores
				"memoryUsage": totalMemory,                 // bytes
				"status":      status,
				"age":         time.Since(deploy.CreationTimestamp.Time).String(),
			}
			allDeployments = append(allDeployments, deploymentInfo)
		}
	}

	respondWithSuccess(w, allDeployments)
}

// handleDeploymentDetail handles getting detailed deployment information
func (s *Server) handleDeploymentDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	ctx := context.Background()

	// Get deployment
	deployment, err := s.k8sClient.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Deployment not found: %v", err))
		return
	}

	// Get pod metrics
	podMetrics, _ := s.collector.CollectPodMetrics(namespace)
	metricsMap := make(map[string]models.PodMetrics)
	for _, pm := range podMetrics {
		metricsMap[pm.Name] = pm
	}

	// Get pods for this deployment
	var pods []map[string]interface{}
	var totalCPU, totalMemory int64
	var runningPods int

	if deployment.Spec.Selector != nil && len(deployment.Spec.Selector.MatchLabels) > 0 {
		labelSelector := labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels)
		podList, err := s.k8sClient.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector.String(),
		})

		if err == nil {
			for _, pod := range podList.Items {
				if pod.Status.Phase == "Running" {
					runningPods++
				}

				metrics := metricsMap[pod.Name]
				totalCPU += metrics.CPU
				totalMemory += metrics.Memory

				podInfo := map[string]interface{}{
					"name":         pod.Name,
					"status":       string(pod.Status.Phase),
					"cpuUsage":     float64(metrics.CPU) / 1000.0,
					"memoryUsage":  metrics.Memory,
					"restartCount": sumRestartCounts(pod.Status.ContainerStatuses),
					"age":          time.Since(pod.CreationTimestamp.Time).String(),
				}
				pods = append(pods, podInfo)
			}
		}
	}

	replicas := int32(0)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	// Calculate health score
	healthScore := 0.0
	if replicas > 0 {
		healthScore = (float64(runningPods) / float64(replicas)) * 100
	}

	// Get recommendations for this deployment (empty for now)
	var recommendations []interface{}

	avgCPU := 0.0
	avgMemory := int64(0)
	if len(pods) > 0 {
		avgCPU = float64(totalCPU) / float64(len(pods)) / 1000.0
		avgMemory = totalMemory / int64(len(pods))
	}

	detail := map[string]interface{}{
		"name":        deployment.Name,
		"namespace":   deployment.Namespace,
		"replicas":    replicas,
		"healthScore": healthScore,
		"cpuUsage":    float64(totalCPU) / 1000.0,
		"memoryUsage": totalMemory,
		"status":      "Running",
		"pods":        pods,
		"metrics": map[string]interface{}{
			"avgCPU":     avgCPU,
			"maxCPU":     float64(totalCPU) / 1000.0,
			"p95CPU":     float64(totalCPU) / 1000.0,
			"avgMemory":  avgMemory,
			"maxMemory":  totalMemory,
			"p95Memory":  totalMemory,
		},
		"recommendations": recommendations,
	}

	respondWithSuccess(w, detail)
}

// sumRestartCounts calculates total restart count for all containers in a pod
func sumRestartCounts(statuses []corev1.ContainerStatus) int {
	total := 0
	for _, status := range statuses {
		total += int(status.RestartCount)
	}
	return total
}
