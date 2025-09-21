#!/bin/bash

# Install script for the Kubernetes Edge Computing Central Orchestrator (JavaScript version)

set -e

echo "Installing Kubernetes Edge Computing Central Orchestrator..."

# Create namespace if it doesn't exist
kubectl create namespace edge-computing 2>/dev/null || true

# Create TLS secret for HTTPS
if [ ! -f ./certs/tls.key ] || [ ! -f ./certs/tls.crt ]; then
  echo "Generating self-signed TLS certificates..."
  mkdir -p ./certs
  openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout ./certs/tls.key -out ./certs/tls.crt \
    -subj "/CN=edge-orchestrator.local"
  
  kubectl create secret tls edge-tls \
    --key ./certs/tls.key \
    --cert ./certs/tls.crt \
    --namespace edge-computing
fi

# Build and push Docker image
echo "Building orchestrator Docker image..."
docker build -t edge-orchestrator:latest ../central-orchestrator/

# Apply Kubernetes manifests
echo "Deploying orchestrator to Kubernetes..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: edge-orchestrator
  namespace: edge-computing
spec:
  replicas: 1
  selector:
    matchLabels:
      app: edge-orchestrator
  template:
    metadata:
      labels:
        app: edge-orchestrator
    spec:
      containers:
      - name: orchestrator
        image: edge-orchestrator:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8443
        volumeMounts:
        - name: tls-certs
          mountPath: /etc/certs
        env:
        - name: PORT
          value: "8443"
        - name: NODE_ENV
          value: "production"
      volumes:
      - name: tls-certs
        secret:
          secretName: edge-tls
---
apiVersion: v1
kind: Service
metadata:
  name: edge-orchestrator
  namespace: edge-computing
spec:
  selector:
    app: edge-orchestrator
  ports:
  - port: 8443
    targetPort: 8443
  type: ClusterIP
EOF

# Create NodePort service for external access
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: edge-orchestrator-external
  namespace: edge-computing
spec:
  selector:
    app: edge-orchestrator
  ports:
  - port: 8443
    targetPort: 8443
    nodePort: 30443
  type: NodePort
EOF

echo "Kubernetes Edge Computing Central Orchestrator installed successfully!"
echo "Access the orchestrator at https://<node-ip>:30443"
