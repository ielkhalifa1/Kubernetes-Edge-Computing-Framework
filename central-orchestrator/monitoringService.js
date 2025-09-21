const cron = require('node-cron');

class MonitoringService {
  constructor(logger) {
    this.metrics = new Map();
    this.logger = logger;
  }

  // Start metrics collection
  startMetricsCollection() {
    this.logger.info('Starting metrics collection service');
    
    // Run every 5 minutes
    cron.schedule('*/5 * * * *', () => {
      this.collectMetrics();
    });
  }

  // Collect metrics from all nodes
  collectMetrics() {
    this.logger.info('Collecting metrics from edge nodes');
    // In a real implementation, this would query all nodes for metrics
  }

  // Record metrics for a node
  recordNodeMetrics(nodeId, metrics) {
    this.metrics.set(nodeId, {
      ...metrics,
      timestamp: new Date()
    });
    
    this.logger.info(`Recorded metrics for node: ${nodeId}`);
    return true;
  }

  // Get metrics for a node
  getNodeMetrics(nodeId) {
    return this.metrics.get(nodeId);
  }

  // Get all metrics
  getAllMetrics() {
    return Array.from(this.metrics.entries()).map(([nodeId, metrics]) => ({
      nodeId,
      ...metrics
    }));
  }
}

module.exports = { MonitoringService };