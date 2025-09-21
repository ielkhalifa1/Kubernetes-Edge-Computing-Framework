const { v4: uuidv4 } = require('uuid');

class WorkloadManager {
  constructor(logger) {
    this.workloads = new Map();
    this.logger = logger;
  }

  // Create a new workload
  createWorkload(workloadSpec) {
    const workloadId = uuidv4();
    const workload = {
      id: workloadId,
      name: workloadSpec.name,
      type: workloadSpec.type,
      image: workloadSpec.image,
      resources: workloadSpec.resources || {},
      config: workloadSpec.config || {},
      status: 'CREATED',
      createdAt: new Date(),
      targetNodes: workloadSpec.targetNodes || []
    };

    this.workloads.set(workloadId, workload);
    this.logger.info(`Created new workload: ${workloadId}`);
    return workload;
  }

  // Get all workloads
  getAllWorkloads() {
    return Array.from(this.workloads.values());
  }

  // Get workload by ID
  getWorkload(workloadId) {
    return this.workloads.get(workloadId);
  }

  // Update workload status
  updateWorkloadStatus(workloadId, status) {
    const workload = this.workloads.get(workloadId);
    if (!workload) {
      throw new Error(`Workload not found: ${workloadId}`);
    }

    workload.status = status;
    workload.updatedAt = new Date();
    this.workloads.set(workloadId, workload);
    this.logger.info(`Updated workload ${workloadId} status to ${status}`);
    return workload;
  }

  // Delete workload
  deleteWorkload(workloadId) {
    if (!this.workloads.has(workloadId)) {
      throw new Error(`Workload not found: ${workloadId}`);
    }

    this.workloads.delete(workloadId);
    this.logger.info(`Deleted workload: ${workloadId}`);
    return true;
  }

  // Deploy workload to node
  deployWorkload(workloadId, nodeId) {
    const workload = this.workloads.get(workloadId);
    if (!workload) {
      throw new Error(`Workload not found: ${workloadId}`);
    }

    if (!workload.targetNodes.includes(nodeId)) {
      workload.targetNodes.push(nodeId);
      workload.status = 'DEPLOYING';
      workload.updatedAt = new Date();
      this.workloads.set(workloadId, workload);
      this.logger.info(`Deploying workload ${workloadId} to node ${nodeId}`);
    }

    return workload;
  }
}

module.exports = { WorkloadManager };