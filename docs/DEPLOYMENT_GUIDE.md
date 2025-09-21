# Kubernetes Edge Computing Framework Deployment Guide

This guide provides instructions for deploying the JavaScript-based Kubernetes Edge Computing Framework in various environments.

## Prerequisites

- Node.js 18 or higher
- Docker
- Kubernetes cluster or K3s
- kubectl configured to access your cluster

## Central Orchestrator Deployment

### Local Development Deployment

1. Install dependencies:
   ```bash
   cd central-orchestrator
   npm install
   ```

2. Start the orchestrator in development mode:
   ```bash
   npm run dev
   ```

### Production Kubernetes Deployment

1. Build the Docker image:
   ```bash
   cd central-orchestrator
   docker build -t edge-orchestrator:latest .
   ```

2. Deploy using the provided script:
   ```bash
   cd deployment
   ./install-orchestrator.sh
   ```

3. Verify the deployment:
   ```bash
   kubectl get pods -n edge-computing
   ```

## Edge Agent Deployment

### Local Development Deployment

1. Install dependencies:
   ```bash
   cd edge-agent
   npm install
   ```

2. Configure the agent by setting environment variables:
   ```bash
   export ORCHESTRATOR_URL=https://orchestrator-address:8443
   export NODE_NAME=my-edge-node
   ```

3. Start the agent in development mode:
   ```bash
   npm run dev
   ```

### Production Kubernetes Deployment

1. Build the Docker image:
   ```bash
   cd edge-agent
   docker build -t edge-agent:latest .
   ```

2. Deploy using the provided script:
   ```bash
   cd deployment
   ./install-edge.sh
   ```

3. Verify the deployment:
   ```bash
   kubectl get pods -n edge-computing
   ```

## Configuration Options

### Central Orchestrator

The orchestrator can be configured using environment variables:

- `PORT`: HTTP server port (default: 8443)
- `CERT_PATH`: Path to TLS certificate (default: ./certs/tls.crt)
- `KEY_PATH`: Path to TLS key (default: ./certs/tls.key)
- `NODE_ENV`: Environment mode (development/production)

### Edge Agent

The edge agent can be configured using environment variables:

- `ORCHESTRATOR_URL`: URL of the central orchestrator
- `NODE_NAME`: Name of the edge node
- `CONFIG_PATH`: Path to configuration file (default: ./config.json)

## Security Considerations

- Always use HTTPS for production deployments
- Generate proper TLS certificates for production use
- Implement network policies to restrict communication between components
- Use Kubernetes secrets for storing sensitive information

## Troubleshooting

### Common Issues

1. **Connection refused to orchestrator**:
   - Verify the orchestrator is running
   - Check network connectivity and firewall rules
   - Ensure correct ORCHESTRATOR_URL is set

2. **TLS certificate errors**:
   - Verify certificates are correctly generated and mounted
   - Check certificate expiration dates

3. **Node registration failures**:
   - Check logs for detailed error messages
   - Verify authentication tokens are correct

### Viewing Logs

```bash
# View orchestrator logs
kubectl logs -n edge-computing -l app=edge-orchestrator

# View edge agent logs
kubectl logs -n edge-computing -l app=edge-agent
```

## Scaling

The framework can be scaled horizontally by:

1. Increasing the number of orchestrator replicas for high availability
2. Deploying edge agents on additional edge nodes

## Upgrading

To upgrade components:

1. Build new Docker images with updated code
2. Update the deployment using kubectl or the provided scripts
3. Monitor the rollout to ensure successful upgrade
