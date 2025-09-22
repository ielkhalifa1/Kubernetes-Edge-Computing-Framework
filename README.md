# Kubernetes-Based Edge Computing Framework

A modern JavaScript-based framework for managing edge computing workloads using Kubernetes. This framework enables efficient deployment, management, and monitoring of applications across distributed edge nodes.

## Architecture

The framework consists of two main components:

1. **Central Orchestrator**: A Node.js-based service that manages edge nodes, workloads, and security.
2. **Edge Agent**: A lightweight Node.js application that runs on edge nodes and communicates with the central orchestrator.

## Features

- **Edge Node Management**: Register, monitor, and manage edge nodes
- **Workload Orchestration**: Deploy and manage containerized applications on edge nodes
- **Security**: Secure communication between orchestrator and edge nodes
- **Monitoring**: Collect and visualize metrics from edge nodes
- **Kubernetes Integration**: Leverage Kubernetes for container orchestration at the edge

## Prerequisites

- Node.js 18 or higher
- Docker
- Kubernetes cluster (or K3s for lightweight deployments)

## Installation

### Central Orchestrator

```bash
# Clone the repository
git clone https://github.com/yourusername/kubernetes-edge-framework.git
cd kubernetes-edge-framework

# Install dependencies
cd central-orchestrator
npm install

# Start the orchestrator
npm start
```

### Edge Agent

```bash
# Navigate to edge-agent directory
cd edge-agent

# Install dependencies
npm install

# Start the agent
npm start
```

## Deployment with Kubernetes

The framework includes deployment scripts for Kubernetes:

```bash
# Deploy the central orchestrator
cd deployment
./install-orchestrator.sh

# Deploy edge agents
./install-edge.sh
```

## API Documentation

### Central Orchestrator API

- `POST /api/nodes/register`: Register a new edge node
- `GET /api/nodes`: List all registered nodes
- `GET /api/nodes/:nodeId`: Get node details
- `PUT /api/nodes/:nodeId/status`: Update node status
- `POST /api/workloads`: Create a new workload
- `GET /api/workloads`: List all workloads
- `POST /api/workloads/:workloadId/deploy`: Deploy workload to node
- `POST /api/nodes/:nodeId/metrics`: Record node metrics
- `GET /api/nodes/:nodeId/metrics`: Get node metrics

## Development

### Project Structure

```
kubernetes-edge-framework/
├── central-orchestrator/     # Central orchestrator service
│   ├── server.js             # Main server file
│   ├── nodeManager.js        # Node management logic
│   ├── workloadManager.js    # Workload management logic
│   ├── securityManager.js    # Security and authentication
│   ├── monitoringService.js  # Metrics and monitoring
│   └── routes.js             # API routes
├── edge-agent/               # Edge node agent
│   └── agent.js              # Main agent file
├── deployment/               # Deployment scripts
│   ├── install-orchestrator.sh
│   └── install-edge.sh
└── docs/                     # Documentation
```

## License

MIT
