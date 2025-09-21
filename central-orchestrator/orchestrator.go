package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NewNodeManager creates a new node manager
func NewNodeManager(logger *logrus.Logger) *NodeManager {
	return &NodeManager{
		nodes:  make(map[string]*EdgeNode),
		logger: logger,
	}
}

// NewWorkloadManager creates a new workload manager
func NewWorkloadManager(logger *logrus.Logger) *WorkloadManager {
	return &WorkloadManager{
		workloads: make(map[string]*Workload),
		logger:    logger,
	}
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(logger *logrus.Logger) *SecurityManager {
	return &SecurityManager{
		certificates: make(map[string]*Certificate),
		logger:       logger,
	}
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(logger *logrus.Logger) *MonitoringService {
	return &MonitoringService{
		metrics: make(map[string]interface{}),
		logger:  logger,
	}
}

// StartBackgroundServices starts background services
func (co *CentralOrchestrator) StartBackgroundServices() {
	// Start node health checker
	go co.nodeHealthChecker()
	
	// Start workload scheduler
	go co.workloadScheduler()
	
	// Start metrics collector
	go co.metricsCollector()
}

// nodeHealthChecker checks node health periodically
func (co *CentralOrchestrator) nodeHealthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			co.checkNodeHealth()
		}
	}
}

// checkNodeHealth checks the health of all nodes
func (co *CentralOrchestrator) checkNodeHealth() {
	co.NodeManager.mutex.RLock()
	defer co.NodeManager.mutex.RUnlock()

	for _, node := range co.NodeManager.nodes {
		if time.Since(node.LastHeartbeat) > 2*time.Minute {
			if node.Status != NodeStatusOffline {
				co.Logger.Warnf("Node %s (%s) is offline", node.Name, node.ID)
				node.Status = NodeStatusOffline
				node.UpdatedAt = time.Now()
			}
		}
	}
}

// workloadScheduler handles workload scheduling and deployment
func (co *CentralOrchestrator) workloadScheduler() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			co.scheduleWorkloads()
		}
	}
}

// scheduleWorkloads schedules pending workloads to available nodes
func (co *CentralOrchestrator) scheduleWorkloads() {
	co.WorkloadManager.mutex.Lock()
	defer co.WorkloadManager.mutex.Unlock()

	for _, workload := range co.WorkloadManager.workloads {
		if workload.Status == WorkloadStatusPending {
			co.Logger.Infof("Scheduling workload %s", workload.Name)
			if err := co.scheduleWorkload(workload); err != nil {
				co.Logger.Errorf("Failed to schedule workload %s: %v", workload.Name, err)
			}
		}
	}
}

// scheduleWorkload schedules a specific workload based on placement policy
func (co *CentralOrchestrator) scheduleWorkload(workload *Workload) error {
	nodes := co.selectNodesForWorkload(workload)
	if len(nodes) == 0 {
		return fmt.Errorf("no suitable nodes found for workload %s", workload.Name)
	}

	// Deploy to selected nodes
	for _, node := range nodes {
		deployment := WorkloadDeployment{
			NodeID:     node.ID,
			Status:     WorkloadStatusRunning,
			Replicas:   1, // For now, deploy 1 replica per node
			DeployedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}
		workload.Deployments = append(workload.Deployments, deployment)
	}

	workload.Status = WorkloadStatusRunning
	workload.UpdatedAt = time.Now()
	
	co.Logger.Infof("Workload %s scheduled to %d nodes", workload.Name, len(nodes))
	return nil
}

// selectNodesForWorkload selects appropriate nodes based on placement policy
func (co *CentralOrchestrator) selectNodesForWorkload(workload *Workload) []*EdgeNode {
	co.NodeManager.mutex.RLock()
	defer co.NodeManager.mutex.RUnlock()

	var candidates []*EdgeNode
	
	// Filter nodes based on constraints
	for _, node := range co.NodeManager.nodes {
		if node.Status == NodeStatusOnline && co.nodeMatchesConstraints(node, workload.Placement.Constraints) {
			candidates = append(candidates, node)
		}
	}

	// Apply placement strategy
	switch workload.Placement.Strategy {
	case PlacementStrategyEdgeFirst:
		return co.selectEdgeFirstNodes(candidates, workload)
	case PlacementStrategyLoadBalance:
		return co.selectLoadBalancedNodes(candidates, workload)
	case PlacementStrategyResource:
		return co.selectResourceAwareNodes(candidates, workload)
	default:
		// Default to edge-first
		return co.selectEdgeFirstNodes(candidates, workload)
	}
}

// nodeMatchesConstraints checks if a node matches placement constraints
func (co *CentralOrchestrator) nodeMatchesConstraints(node *EdgeNode, constraints []PlacementConstraint) bool {
	for _, constraint := range constraints {
		switch constraint.Key {
		case "region":
			if !contains(constraint.Values, node.Region) {
				return false
			}
		case "zone":
			if !contains(constraint.Values, node.Zone) {
				return false
			}
		default:
			if labelValue, exists := node.Labels[constraint.Key]; exists {
				if !contains(constraint.Values, labelValue) {
					return false
				}
			} else {
				return false
			}
		}
	}
	return true
}

// selectEdgeFirstNodes selects nodes with edge-first strategy
func (co *CentralOrchestrator) selectEdgeFirstNodes(candidates []*EdgeNode, workload *Workload) []*EdgeNode {
	if len(candidates) == 0 {
		return nil
	}
	
	// For simplicity, select up to replicas count of nodes
	maxNodes := int(workload.Replicas)
	if maxNodes == 0 {
		maxNodes = 1
	}
	
	if len(candidates) <= maxNodes {
		return candidates
	}
	
	return candidates[:maxNodes]
}

// selectLoadBalancedNodes selects nodes with load balancing
func (co *CentralOrchestrator) selectLoadBalancedNodes(candidates []*EdgeNode, workload *Workload) []*EdgeNode {
	// TODO: Implement proper load balancing based on current workloads
	return co.selectEdgeFirstNodes(candidates, workload)
}

// selectResourceAwareNodes selects nodes based on resource availability
func (co *CentralOrchestrator) selectResourceAwareNodes(candidates []*EdgeNode, workload *Workload) []*EdgeNode {
	// TODO: Implement resource-aware selection
	return co.selectEdgeFirstNodes(candidates, workload)
}

// metricsCollector collects metrics from nodes and workloads
func (co *CentralOrchestrator) metricsCollector() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			co.collectMetrics()
		}
	}
}

// collectMetrics collects metrics from all nodes
func (co *CentralOrchestrator) collectMetrics() {
	co.MonitoringService.mutex.Lock()
	defer co.MonitoringService.mutex.Unlock()

	// Collect node metrics
	nodeCount := len(co.NodeManager.nodes)
	onlineNodes := 0
	
	co.NodeManager.mutex.RLock()
	for _, node := range co.NodeManager.nodes {
		if node.Status == NodeStatusOnline {
			onlineNodes++
		}
	}
	co.NodeManager.mutex.RUnlock()

	// Collect workload metrics
	workloadCount := len(co.WorkloadManager.workloads)
	runningWorkloads := 0
	
	co.WorkloadManager.mutex.RLock()
	for _, workload := range co.WorkloadManager.workloads {
		if workload.Status == WorkloadStatusRunning {
			runningWorkloads++
		}
	}
	co.WorkloadManager.mutex.RUnlock()

	// Update metrics
	co.MonitoringService.metrics = map[string]interface{}{
		"nodes_total":        nodeCount,
		"nodes_online":       onlineNodes,
		"workloads_total":    workloadCount,
		"workloads_running":  runningWorkloads,
		"last_updated":       time.Now(),
	}
}

// RegisterNode registers a new edge node
func (co *CentralOrchestrator) RegisterNode(c *gin.Context) {
	var req NodeRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nodeID := generateID()
	now := time.Now()
	
	node := &EdgeNode{
		ID:               nodeID,
		Name:             req.Name,
		Address:          req.Address,
		Status:           NodeStatusOnline,
		LastHeartbeat:    now,
		Labels:           req.Labels,
		Capabilities:     req.Capabilities,
		Region:           req.Region,
		Zone:             req.Zone,
		KubernetesVersion: req.KubernetesVersion,
		ContainerRuntime: req.ContainerRuntime,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	if node.Region == "" {
		node.Region = "default"
	}
	if node.Zone == "" {
		node.Zone = "default"
	}

	co.NodeManager.mutex.Lock()
	co.NodeManager.nodes[nodeID] = node
	co.NodeManager.mutex.Unlock()

	co.Logger.Infof("Node %s registered with ID %s", req.Name, nodeID)
	
	c.JSON(http.StatusCreated, gin.H{
		"id": nodeID,
		"node": node,
	})
}

// ListNodes returns all registered nodes
func (co *CentralOrchestrator) ListNodes(c *gin.Context) {
	co.NodeManager.mutex.RLock()
	defer co.NodeManager.mutex.RUnlock()

	nodes := make([]*EdgeNode, 0, len(co.NodeManager.nodes))
	for _, node := range co.NodeManager.nodes {
		nodes = append(nodes, node)
	}

	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

// GetNode returns a specific node
func (co *CentralOrchestrator) GetNode(c *gin.Context) {
	nodeID := c.Param("id")
	
	co.NodeManager.mutex.RLock()
	node, exists := co.NodeManager.nodes[nodeID]
	co.NodeManager.mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"node": node})
}

// UnregisterNode removes a node from the cluster
func (co *CentralOrchestrator) UnregisterNode(c *gin.Context) {
	nodeID := c.Param("id")
	
	co.NodeManager.mutex.Lock()
	defer co.NodeManager.mutex.Unlock()

	if _, exists := co.NodeManager.nodes[nodeID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	delete(co.NodeManager.nodes, nodeID)
	co.Logger.Infof("Node %s unregistered", nodeID)
	
	c.JSON(http.StatusOK, gin.H{"message": "Node unregistered successfully"})
}

// NodeHeartbeat handles node heartbeat updates
func (co *CentralOrchestrator) NodeHeartbeat(c *gin.Context) {
	nodeID := c.Param("id")
	
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	co.NodeManager.mutex.Lock()
	defer co.NodeManager.mutex.Unlock()

	node, exists := co.NodeManager.nodes[nodeID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	node.Status = req.Status
	node.Resources = req.Resources
	node.LastHeartbeat = time.Now()
	node.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, gin.H{"message": "Heartbeat received"})
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
