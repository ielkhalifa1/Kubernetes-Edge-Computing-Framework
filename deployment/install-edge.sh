#!/bin/bash

# Install script for the Kubernetes Edge Computing Edge Agent (JavaScript version)

set -e

echo "Installing Kubernetes Edge Computing Edge Agent..."

# Create namespace if it doesn't exist
kubectl create namespace edge-computing 2>/dev/null || true

# Set orchestrator URL
ORCHESTRATOR_URL=${ORCHESTRATOR_URL:-"https://edge-orchestrator.edge-computing.svc.cluster.local:8443"}
NODE_NAME=${NODE_NAME:-$(hostname)}

# Build and push Docker image
echo "Building edge agent Docker image..."
docker build -t edge-agent:latest ../edge-agent/

# Apply Kubernetes manifests
echo "Deploying edge agent to Kubernetes..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: edge-agent
  namespace: edge-computing
spec:
  selector:
    matchLabels:
      app: edge-agent
  template:
    metadata:
      labels:
        app: edge-agent
    spec:
      containers:
      - name: agent
        image: edge-agent:latest
        imagePullPolicy: IfNotPresent
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: ORCHESTRATOR_URL
          value: "${ORCHESTRATOR_URL}"
        - name: NODE_ENV
          value: "production"
        volumeMounts:
        - name: config
          mountPath: /usr/src/app/config
        - name: varrun
          mountPath: /var/run
        - name: varlibkubelet
          mountPath: /var/lib/kubelet
          readOnly: true
      volumes:
      - name: config
        emptyDir: {}
      - name: varrun
        hostPath:
          path: /var/run
      - name: varlibkubelet
        hostPath:
          path: /var/lib/kubelet
EOF

echo "Kubernetes Edge Computing Edge Agent installed successfully!"
echo "The agent will automatically register with the orchestrator at ${ORCHESTRATOR_URL}"
