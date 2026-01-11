package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/internal/models"
)

// Collector implements the MetricsCollector interface
type Collector struct {
	client     *k8s.Client
	k8s        *k8sCollector
	store      *metricsStore
	config     Config
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	runningMu  sync.RWMutex
	namespaces []string // Namespaces to monitor
}

// New creates a new metrics collector with default configuration
func New(client *k8s.Client) *Collector {
	return NewWithConfig(client, DefaultConfig())
}

// NewWithConfig creates a new metrics collector with custom configuration
func NewWithConfig(client *k8s.Client, config Config) *Collector {
	ctx, cancel := context.WithCancel(context.Background())

	return &Collector{
		client:     client,
		k8s:        newK8sCollector(client),
		store:      newMetricsStore(config.RetentionPeriod),
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
		namespaces: []string{"default"}, // Default namespace, can be extended
	}
}

// SetNamespaces sets the namespaces to monitor
func (c *Collector) SetNamespaces(namespaces []string) {
	c.namespaces = namespaces
}

// Start begins the metrics collection loop
func (c *Collector) Start() error {
	c.runningMu.Lock()
	if c.running {
		c.runningMu.Unlock()
		return fmt.Errorf("collector is already running")
	}
	c.running = true
	c.runningMu.Unlock()

	// Start collection goroutine
	c.wg.Add(1)
	go c.collectionLoop()

	// Start cleanup goroutine
	c.wg.Add(1)
	go c.cleanupLoop()

	log.Printf("Metrics collector started (interval: %v, retention: %v)",
		c.config.CollectionInterval, c.config.RetentionPeriod)

	return nil
}

// Stop stops the metrics collection
func (c *Collector) Stop() {
	c.runningMu.Lock()
	if !c.running {
		c.runningMu.Unlock()
		return
	}
	c.running = false
	c.runningMu.Unlock()

	c.cancel()
	c.wg.Wait()

	log.Println("Metrics collector stopped")
}

// IsRunning returns whether the collector is currently running
func (c *Collector) IsRunning() bool {
	c.runningMu.RLock()
	defer c.runningMu.RUnlock()
	return c.running
}

// collectionLoop runs the periodic metrics collection
func (c *Collector) collectionLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	// Collect immediately on start
	c.collectAllMetrics()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectAllMetrics()
		}
	}
}

// cleanupLoop runs the periodic cleanup of old data
func (c *Collector) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			removed := c.store.Cleanup()
			if removed > 0 {
				log.Printf("Cleaned up %d old data points, current store size: %d", removed, c.store.Size())
			}
		}
	}
}

// collectAllMetrics collects all metrics from all monitored namespaces
func (c *Collector) collectAllMetrics() {
	timestamp := time.Now()

	// Collect node metrics (cluster-wide)
	nodeMetrics, err := c.CollectNodeMetrics()
	if err != nil {
		log.Printf("Error collecting node metrics: %v", err)
	} else {
		c.storeNodeMetrics(nodeMetrics, timestamp)
	}

	// Collect pod and HPA metrics for each namespace
	for _, namespace := range c.namespaces {
		// Collect pod metrics
		podMetrics, err := c.CollectPodMetrics(namespace)
		if err != nil {
			log.Printf("Error collecting pod metrics for namespace %s: %v", namespace, err)
		} else {
			c.storePodMetrics(podMetrics, timestamp)
		}

		// Collect HPA metrics
		hpaMetrics, err := c.CollectHPAMetrics(namespace)
		if err != nil {
			log.Printf("Error collecting HPA metrics for namespace %s: %v", namespace, err)
		} else {
			c.storeHPAMetrics(hpaMetrics, timestamp)
		}
	}
}

// storePodMetrics stores pod metrics in the time-series store
func (c *Collector) storePodMetrics(metrics []models.PodMetrics, timestamp time.Time) {
	for _, metric := range metrics {
		resource := fmt.Sprintf("pod/%s", metric.Name)

		// Store CPU metric (convert to float64)
		c.store.Store(resource, "cpu", float64(metric.CPU), metric.Timestamp)

		// Store Memory metric (convert to float64)
		c.store.Store(resource, "memory", float64(metric.Memory), metric.Timestamp)
	}
}

// storeNodeMetrics stores node metrics in the time-series store
func (c *Collector) storeNodeMetrics(metrics []models.NodeMetrics, timestamp time.Time) {
	for _, metric := range metrics {
		resource := fmt.Sprintf("node/%s", metric.Name)

		// Store CPU metric (convert to float64)
		c.store.Store(resource, "cpu", float64(metric.CPU), metric.Timestamp)

		// Store Memory metric (convert to float64)
		c.store.Store(resource, "memory", float64(metric.Memory), metric.Timestamp)
	}
}

// storeHPAMetrics stores HPA metrics in the time-series store
func (c *Collector) storeHPAMetrics(metrics []models.HPAMetrics, timestamp time.Time) {
	for _, metric := range metrics {
		resource := fmt.Sprintf("hpa/%s", metric.Name)

		// Store current replicas
		c.store.Store(resource, "current_replicas", float64(metric.CurrentReplicas), metric.Timestamp)

		// Store desired replicas
		c.store.Store(resource, "desired_replicas", float64(metric.DesiredReplicas), metric.Timestamp)

		// Store target CPU
		c.store.Store(resource, "target_cpu", float64(metric.TargetCPU), metric.Timestamp)

		// Store current CPU
		c.store.Store(resource, "current_cpu", float64(metric.CurrentCPU), metric.Timestamp)
	}
}

// CollectPodMetrics collects current pod metrics for a namespace
func (c *Collector) CollectPodMetrics(namespace string) ([]models.PodMetrics, error) {
	return c.k8s.CollectPodMetrics(namespace)
}

// CollectNodeMetrics collects current node metrics
func (c *Collector) CollectNodeMetrics() ([]models.NodeMetrics, error) {
	return c.k8s.CollectNodeMetrics()
}

// CollectHPAMetrics collects HPA metrics for a namespace
func (c *Collector) CollectHPAMetrics(namespace string) ([]models.HPAMetrics, error) {
	return c.k8s.CollectHPAMetrics(namespace)
}

// GetTimeSeriesData retrieves time-series data for a resource/metric
func (c *Collector) GetTimeSeriesData(resource, metric string, duration time.Duration) (models.TimeSeriesData, error) {
	return c.store.GetTimeSeriesData(resource, metric, duration)
}

// GetResourcePercentiles calculates percentiles for a resource metric
func (c *Collector) GetResourcePercentiles(resource, metric string, duration time.Duration) (p50, p95, p99 float64, err error) {
	return c.store.GetResourcePercentiles(resource, metric, duration)
}

// GetStoreSize returns the current size of the metrics store
func (c *Collector) GetStoreSize() int {
	return c.store.Size()
}

// GetStoredMetricKeys returns all metric keys currently in the store
func (c *Collector) GetStoredMetricKeys() []string {
	keys := c.store.Keys()
	result := make([]string, len(keys))
	for i, key := range keys {
		result[i] = fmt.Sprintf("%s/%s", key.Resource, key.Metric)
	}
	return result
}
