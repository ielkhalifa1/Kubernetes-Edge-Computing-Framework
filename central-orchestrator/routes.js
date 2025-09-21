// API routes for the central orchestrator

function setupRoutes(app, orchestrator) {
  const { nodeManager, workloadManager, securityManager, monitoringService, logger } = orchestrator;

  // Middleware to authenticate requests
  const authenticate = (req, res, next) => {
    const nodeId = req.headers['x-node-id'];
    const token = req.headers['x-auth-token'];
    
    if (!nodeId || !token) {
      return res.status(401).json({ error: 'Authentication required' });
    }
    
    if (!securityManager.validateToken(nodeId, token)) {
      return res.status(403).json({ error: 'Invalid authentication' });
    }
    
    req.nodeId = nodeId;
    next();
  };

  // Health check endpoint
  app.get('/health', (req, res) => {
    res.status(200).json({ status: 'OK' });
  });

  // Node registration
  app.post('/api/nodes/register', (req, res) => {
    try {
      const nodeInfo = req.body;
      
      if (!nodeInfo.name || !nodeInfo.ipAddress) {
        return res.status(400).json({ error: 'Missing required node information' });
      }
      
      const nodeId = nodeManager.registerNode(nodeInfo);
      const token = securityManager.generateToken(nodeId);
      
      res.status(201).json({ nodeId, token });
    } catch (error) {
      logger.error(`Error registering node: ${error.message}`);
      res.status(500).json({ error: 'Failed to register node' });
    }
  });

  // Get all nodes
  app.get('/api/nodes', (req, res) => {
    try {
      const nodes = nodeManager.getAllNodes();
      res.status(200).json(nodes);
    } catch (error) {
      logger.error(`Error getting nodes: ${error.message}`);
      res.status(500).json({ error: 'Failed to get nodes' });
    }
  });

  // Get node by ID
  app.get('/api/nodes/:nodeId', (req, res) => {
    try {
      const node = nodeManager.getNode(req.params.nodeId);
      
      if (!node) {
        return res.status(404).json({ error: 'Node not found' });
      }
      
      res.status(200).json(node);
    } catch (error) {
      logger.error(`Error getting node: ${error.message}`);
      res.status(500).json({ error: 'Failed to get node' });
    }
  });

  // Update node status
  app.put('/api/nodes/:nodeId/status', authenticate, (req, res) => {
    try {
      const { status } = req.body;
      
      if (!status) {
        return res.status(400).json({ error: 'Status is required' });
      }
      
      const node = nodeManager.updateNodeStatus(req.params.nodeId, status);
      res.status(200).json(node);
    } catch (error) {
      logger.error(`Error updating node status: ${error.message}`);
      res.status(500).json({ error: 'Failed to update node status' });
    }
  });

  // Create workload
  app.post('/api/workloads', (req, res) => {
    try {
      const workloadSpec = req.body;
      
      if (!workloadSpec.name || !workloadSpec.type || !workloadSpec.image) {
        return res.status(400).json({ error: 'Missing required workload information' });
      }
      
      const workload = workloadManager.createWorkload(workloadSpec);
      res.status(201).json(workload);
    } catch (error) {
      logger.error(`Error creating workload: ${error.message}`);
      res.status(500).json({ error: 'Failed to create workload' });
    }
  });

  // Get all workloads
  app.get('/api/workloads', (req, res) => {
    try {
      const workloads = workloadManager.getAllWorkloads();
      res.status(200).json(workloads);
    } catch (error) {
      logger.error(`Error getting workloads: ${error.message}`);
      res.status(500).json({ error: 'Failed to get workloads' });
    }
  });

  // Deploy workload to node
  app.post('/api/workloads/:workloadId/deploy', (req, res) => {
    try {
      const { nodeId } = req.body;
      
      if (!nodeId) {
        return res.status(400).json({ error: 'Node ID is required' });
      }
      
      const workload = workloadManager.deployWorkload(req.params.workloadId, nodeId);
      res.status(200).json(workload);
    } catch (error) {
      logger.error(`Error deploying workload: ${error.message}`);
      res.status(500).json({ error: 'Failed to deploy workload' });
    }
  });

  // Record node metrics
  app.post('/api/nodes/:nodeId/metrics', authenticate, (req, res) => {
    try {
      const metrics = req.body;
      
      if (!metrics) {
        return res.status(400).json({ error: 'Metrics are required' });
      }
      
      monitoringService.recordNodeMetrics(req.params.nodeId, metrics);
      res.status(200).json({ success: true });
    } catch (error) {
      logger.error(`Error recording metrics: ${error.message}`);
      res.status(500).json({ error: 'Failed to record metrics' });
    }
  });

  // Get node metrics
  app.get('/api/nodes/:nodeId/metrics', (req, res) => {
    try {
      const metrics = monitoringService.getNodeMetrics(req.params.nodeId);
      
      if (!metrics) {
        return res.status(404).json({ error: 'Metrics not found' });
      }
      
      res.status(200).json(metrics);
    } catch (error) {
      logger.error(`Error getting metrics: ${error.message}`);
      res.status(500).json({ error: 'Failed to get metrics' });
    }
  });
}

module.exports = { setupRoutes };