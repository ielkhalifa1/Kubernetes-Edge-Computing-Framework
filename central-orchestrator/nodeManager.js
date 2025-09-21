const { v4: uuidv4 } = require('uuid');
const cron = require('node-cron');

class NodeManager {
  constructor(logger) {
    this.nodes = new Map();
    this.logger = logger;
    this.healthCheckInterval = 60000; // 1 minute
  }

  // Register a new edge node
  registerNode(nodeInfo) {
    const nodeId = uuidv4();
    const node = {
      id: nodeId,
      name: nodeInfo.name,
      ipAddress: nodeInfo.ipAddress,
      status: 'REGISTERED',
      capabilities: nodeInfo.capabilities || {},
      lastSeen: new Date(),
      workloads: []
    };

    this.nodes.set(nodeId, node);
    this.logger.info(`Registered new edge node: ${nodeId}`);
    return nodeId;
  }

  // Get all registered nodes
  getAllNodes() {
    return Array.from(this.nodes.values());
  }

  // Get node by ID
  getNode(nodeId) {
    return this.nodes.get(nodeId);
  }

  // Update node status
  updateNodeStatus(nodeId, status) {
    const node = this.nodes.get(nodeId);
    if (!node) {
      throw new Error(`Node not found: ${nodeId}`);
    }

    node.status = status;
    node.lastSeen = new Date();
    this.nodes.set(nodeId, node);
    this.logger.info(`Updated node ${nodeId} status to ${status}`);
    return node;
  }

  // Remove node
  removeNode(nodeId) {
    if (!this.nodes.has(nodeId)) {
      throw new Error(`Node not found: ${nodeId}`);
    }

    this.nodes.delete(nodeId);
    this.logger.info(`Removed node: ${nodeId}`);
    return true;
  }

  // Start health checker
  startHealthChecker() {
    this.logger.info('Starting node health checker');
    
    // Run every minute
    cron.schedule('* * * * *', () => {
      this.checkNodesHealth();
    });
  }

  // Check health of all nodes
  checkNodesHealth() {
    const now = new Date();
    
    for (const [nodeId, node] of this.nodes.entries()) {
      const lastSeen = new Date(node.lastSeen);
      const timeDiff = now - lastSeen;
      
      // If node hasn't been seen in 5 minutes, mark as OFFLINE
      if (timeDiff > 5 * 60 * 1000) {
        if (node.status !== 'OFFLINE') {
          node.status = 'OFFLINE';
          this.nodes.set(nodeId, node);
          this.logger.warn(`Node ${nodeId} marked as OFFLINE`);
        }
      }
    }
  }
}

module.exports = { NodeManager };