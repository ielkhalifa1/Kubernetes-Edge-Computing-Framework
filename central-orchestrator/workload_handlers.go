package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DeployWorkload handles workload deployment requests
func (co *CentralOrchestrator) DeployWorkload(c *gin.Context) {
	var req WorkloadDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workloadID := generateID()
	now := time.Now()
	
	workload := &Workload{
		ID:          workloadID,
		Name:        req.Name,
		Namespace:   req.Namespace,
		Type:        req.Type,
		Image:       req.Image,
		Replicas:    req.Replicas,
		Resources:   req.Resources,
		Environment: req.Environment,
		Labels:      req.Labels,
		Placement:   req.Placement,
		Status:      WorkloadStatusPending,
		Deployments: make([]WorkloadDeployment, 0),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set defaults
	if workload.Namespace == "" {
		workload.Namespace = "default"
	}
	if workload.Replicas == 0 {
		workload.Replicas = 1
	}
	if workload.Labels == nil {
		workload.Labels = make(map[string]string)
	}
	if workload.Environment == nil {
		workload.Environment = make(map[string]string)
	}
	if workload.Placement.Strategy == "" {
		workload.Placement.Strategy = PlacementStrategyEdgeFirst
	}

	// Generate selector from labels
	workload.Selector = make(map[string]string)
	workload.Selector["app"] = workload.Name
	workload.Selector["workload-id"] = workloadID

	co.WorkloadManager.mutex.Lock()
	co.WorkloadManager.workloads[workloadID] = workload
	co.WorkloadManager.mutex.Unlock()

	co.Logger.Infof("Workload %s created with ID %s", req.Name, workloadID)
	
	c.JSON(http.StatusCreated, gin.H{
		"id":       workloadID,
		"workload": workload,
	})
}

// ListWorkloads returns all workloads
func (co *CentralOrchestrator) ListWorkloads(c *gin.Context) {
	co.WorkloadManager.mutex.RLock()
	defer co.WorkloadManager.mutex.RUnlock()

	workloads := make([]*Workload, 0, len(co.WorkloadManager.workloads))
	for _, workload := range co.WorkloadManager.workloads {
		workloads = append(workloads, workload)
	}

	c.JSON(http.StatusOK, gin.H{"workloads": workloads})
}

// GetWorkload returns a specific workload
func (co *CentralOrchestrator) GetWorkload(c *gin.Context) {
	workloadID := c.Param("id")
	
	co.WorkloadManager.mutex.RLock()
	workload, exists := co.WorkloadManager.workloads[workloadID]
	co.WorkloadManager.mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workload not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"workload": workload})
}

// DeleteWorkload removes a workload
func (co *CentralOrchestrator) DeleteWorkload(c *gin.Context) {
	workloadID := c.Param("id")
	
	co.WorkloadManager.mutex.Lock()
	defer co.WorkloadManager.mutex.Unlock()

	workload, exists := co.WorkloadManager.workloads[workloadID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workload not found"})
		return
	}

	// TODO: Actually delete workload from edge nodes
	// For now, just mark as stopped and remove from memory
	workload.Status = WorkloadStatusStopped
	workload.UpdatedAt = time.Now()
	
	delete(co.WorkloadManager.workloads, workloadID)
	co.Logger.Infof("Workload %s deleted", workloadID)
	
	c.JSON(http.StatusOK, gin.H{"message": "Workload deleted successfully"})
}

// ScaleWorkload scales a workload
func (co *CentralOrchestrator) ScaleWorkload(c *gin.Context) {
	workloadID := c.Param("id")
	
	var req ScaleWorkloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	co.WorkloadManager.mutex.Lock()
	defer co.WorkloadManager.mutex.Unlock()

	workload, exists := co.WorkloadManager.workloads[workloadID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workload not found"})
		return
	}

	oldReplicas := workload.Replicas
	workload.Replicas = req.Replicas
	workload.Status = WorkloadStatusPending // Trigger rescheduling
	workload.UpdatedAt = time.Now()

	co.Logger.Infof("Workload %s scaled from %d to %d replicas", workloadID, oldReplicas, req.Replicas)
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Workload scaled successfully",
		"workload": workload,
	})
}

// GetMetrics returns overall system metrics
func (co *CentralOrchestrator) GetMetrics(c *gin.Context) {
	co.MonitoringService.mutex.RLock()
	defer co.MonitoringService.mutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{"metrics": co.MonitoringService.metrics})
}

// GetNodeMetrics returns metrics for a specific node
func (co *CentralOrchestrator) GetNodeMetrics(c *gin.Context) {
	nodeID := c.Param("id")
	
	co.NodeManager.mutex.RLock()
	node, exists := co.NodeManager.nodes[nodeID]
	co.NodeManager.mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	metrics := map[string]interface{}{
		"node_id":     node.ID,
		"name":        node.Name,
		"status":      node.Status,
		"resources":   node.Resources,
		"last_heartbeat": node.LastHeartbeat,
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

// GetWorkloadMetrics returns metrics for a specific workload
func (co *CentralOrchestrator) GetWorkloadMetrics(c *gin.Context) {
	workloadID := c.Param("id")
	
	co.WorkloadManager.mutex.RLock()
	workload, exists := co.WorkloadManager.workloads[workloadID]
	co.WorkloadManager.mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workload not found"})
		return
	}

	// Count running deployments
	runningDeployments := 0
	totalReplicas := int32(0)
	
	for _, deployment := range workload.Deployments {
		if deployment.Status == WorkloadStatusRunning {
			runningDeployments++
			totalReplicas += deployment.Replicas
		}
	}

	metrics := map[string]interface{}{
		"workload_id":         workload.ID,
		"name":               workload.Name,
		"status":             workload.Status,
		"desired_replicas":   workload.Replicas,
		"running_replicas":   totalReplicas,
		"running_deployments": runningDeployments,
		"total_deployments":  len(workload.Deployments),
		"last_updated":       workload.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}
