# Kubernetes Edge Computing Framework API Reference

This document provides a comprehensive reference for the REST API endpoints exposed by the JavaScript implementation of the Kubernetes Edge Computing Framework.

## Base URL

All API endpoints are relative to the base URL:

```
https://{orchestrator-address}:8443/api/v1
```

## Authentication

All API requests require authentication using a JWT token in the Authorization header:

```
Authorization: Bearer {token}
```

## Endpoints

### Health Check

```
GET /health
```

Returns the health status of the orchestrator.

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2023-07-01T12:00:00Z"
}
```

### Node Management

#### Register Node

```
POST /nodes
```

Registers a new edge node with the orchestrator.

**Request Body:**
```json
{
  "name": "edge-node-1",
  "address": "192.168.1.100",
  "capabilities": ["compute", "storage", "gpu"],
  "resources": {
    "cpu": "4",
    "memory": "8Gi",
    "storage": "100Gi"
  },
  "labels": {
    "location": "datacenter-1",
    "environment": "production"
  }
}
```

**Response:**
```json
{
  "id": "node-uuid",
  "name": "edge-node-1",
  "address": "192.168.1.100",
  "status": "registered",
  "registered_at": "2023-07-01T12:00:00Z"
}
```

#### Get All Nodes

```
GET /nodes
```

Returns a list of all registered edge nodes.

**Response:**
```json
{
  "nodes": [
    {
      "id": "node-uuid-1",
      "name": "edge-node-1",
      "address": "192.168.1.100",
      "status": "active",
      "last_heartbeat": "2023-07-01T12:05:00Z"
    },
    {
      "id": "node-uuid-2",
      "name": "edge-node-2",
      "address": "192.168.1.101",
      "status": "active",
      "last_heartbeat": "2023-07-01T12:04:30Z"
    }
  ]
}
```

#### Get Node Details

```
GET /nodes/{node-id}
```

Returns detailed information about a specific node.

**Response:**
```json
{
  "id": "node-uuid-1",
  "name": "edge-node-1",
  "address": "192.168.1.100",
  "status": "active",
  "capabilities": ["compute", "storage", "gpu"],
  "resources": {
    "cpu": "4",
    "memory": "8Gi",
    "storage": "100Gi"
  },
  "labels": {
    "location": "datacenter-1",
    "environment": "production"
  },
  "registered_at": "2023-07-01T10:00:00Z",
  "last_heartbeat": "2023-07-01T12:05:00Z"
}
```

#### Update Node Status

```
PUT /nodes/{node-id}/status
```

Updates the status of a node.

**Request Body:**
```json
{
  "status": "active",
  "heartbeat": true,
  "resources": {
    "cpu_usage": "2.5",
    "memory_usage": "4Gi",
    "storage_usage": "50Gi"
  }
}
```

**Response:**
```json
{
  "id": "node-uuid-1",
  "name": "edge-node-1",
  "status": "active",
  "last_heartbeat": "2023-07-01T12:10:00Z"
}
```

#### Delete Node

```
DELETE /nodes/{node-id}
```

Removes a node from the orchestrator.

**Response:**
```json
{
  "success": true,
  "message": "Node successfully removed"
}
```

### Workload Management

#### Create Workload

```
POST /workloads
```

Creates a new workload definition.

**Request Body:**
```json
{
  "name": "web-app",
  "namespace": "default",
  "type": "deployment",
  "image": "nginx:latest",
  "replicas": 3,
  "resources": {
    "requests": {
      "cpu": "100m",
      "memory": "128Mi"
    },
    "limits": {
      "cpu": "200m",
      "memory": "256Mi"
    }
  },
  "placement": {
    "strategy": "edge-first",
    "constraints": [
      {
        "key": "location",
        "operator": "In",
        "values": ["datacenter-1"]
      }
    ]
  }
}
```

**Response:**
```json
{
  "id": "workload-uuid",
  "name": "web-app",
  "namespace": "default",
  "status": "created",
  "created_at": "2023-07-01T12:15:00Z"
}
```

#### Get All Workloads

```
GET /workloads
```

Returns a list of all workloads.

**Response:**
```json
{
  "workloads": [
    {
      "id": "workload-uuid-1",
      "name": "web-app",
      "namespace": "default",
      "type": "deployment",
      "status": "running",
      "created_at": "2023-07-01T12:15:00Z"
    },
    {
      "id": "workload-uuid-2",
      "name": "database",
      "namespace": "data",
      "type": "statefulset",
      "status": "pending",
      "created_at": "2023-07-01T12:20:00Z"
    }
  ]
}
```

#### Get Workload Details

```
GET /workloads/{workload-id}
```

Returns detailed information about a specific workload.

**Response:**
```json
{
  "id": "workload-uuid-1",
  "name": "web-app",
  "namespace": "default",
  "type": "deployment",
  "image": "nginx:latest",
  "replicas": 3,
  "resources": {
    "requests": {
      "cpu": "100m",
      "memory": "128Mi"
    },
    "limits": {
      "cpu": "200m",
      "memory": "256Mi"
    }
  },
  "placement": {
    "strategy": "edge-first",
    "constraints": [
      {
        "key": "location",
        "operator": "In",
        "values": ["datacenter-1"]
      }
    ]
  },
  "status": "running",
  "nodes": ["node-uuid-1", "node-uuid-2"],
  "created_at": "2023-07-01T12:15:00Z",
  "updated_at": "2023-07-01T12:25:00Z"
}
```

#### Update Workload

```
PUT /workloads/{workload-id}
```

Updates an existing workload.

**Request Body:**
```json
{
  "replicas": 5,
  "resources": {
    "requests": {
      "cpu": "200m",
      "memory": "256Mi"
    },
    "limits": {
      "cpu": "400m",
      "memory": "512Mi"
    }
  }
}
```

**Response:**
```json
{
  "id": "workload-uuid-1",
  "name": "web-app",
  "status": "updating",
  "updated_at": "2023-07-01T12:30:00Z"
}
```

#### Delete Workload

```
DELETE /workloads/{workload-id}
```

Removes a workload.

**Response:**
```json
{
  "success": true,
  "message": "Workload successfully removed"
}
```

#### Deploy Workload

```
POST /workloads/{workload-id}/deploy
```

Deploys a workload to specified nodes.

**Request Body:**
```json
{
  "nodes": ["node-uuid-1", "node-uuid-2"],
  "strategy": "rolling"
}
```

**Response:**
```json
{
  "id": "workload-uuid-1",
  "name": "web-app",
  "status": "deploying",
  "deployment_id": "deploy-uuid",
  "target_nodes": ["node-uuid-1", "node-uuid-2"],
  "started_at": "2023-07-01T12:35:00Z"
}
```

### Monitoring

#### Record Node Metrics

```
POST /metrics/nodes/{node-id}
```

Records metrics for a specific node.

**Request Body:**
```json
{
  "timestamp": "2023-07-01T12:40:00Z",
  "cpu_usage": 45.5,
  "memory_usage": 3.2,
  "memory_total": 8.0,
  "disk_usage": 42.8,
  "disk_total": 100.0,
  "network": {
    "rx_bytes": 1024000,
    "tx_bytes": 512000
  },
  "containers": [
    {
      "name": "web-app-1",
      "cpu_usage": 15.2,
      "memory_usage": 120.5
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "recorded_at": "2023-07-01T12:40:00Z"
}
```

#### Get Node Metrics

```
GET /metrics/nodes/{node-id}
```

Returns metrics for a specific node.

**Query Parameters:**
- `timeframe`: Time period for metrics (e.g., "1h", "24h", "7d")
- `resolution`: Data point resolution (e.g., "1m", "5m", "1h")

**Response:**
```json
{
  "node_id": "node-uuid-1",
  "node_name": "edge-node-1",
  "timeframe": "1h",
  "resolution": "5m",
  "metrics": {
    "cpu_usage": [
      {"timestamp": "2023-07-01T11:40:00Z", "value": 42.1},
      {"timestamp": "2023-07-01T11:45:00Z", "value": 43.5},
      {"timestamp": "2023-07-01T11:50:00Z", "value": 44.2}
    ],
    "memory_usage": [
      {"timestamp": "2023-07-01T11:40:00Z", "value": 3.1},
      {"timestamp": "2023-07-01T11:45:00Z", "value": 3.2},
      {"timestamp": "2023-07-01T11:50:00Z", "value": 3.2}
    ]
  }
}
```

#### Get All Metrics

```
GET /metrics
```

Returns aggregated metrics for all nodes.

**Query Parameters:**
- `timeframe`: Time period for metrics (e.g., "1h", "24h", "7d")
- `resolution`: Data point resolution (e.g., "1m", "5m", "1h")

**Response:**
```json
{
  "timeframe": "1h",
  "resolution": "5m",
  "nodes_count": 2,
  "aggregated": {
    "cpu_usage_avg": [
      {"timestamp": "2023-07-01T11:40:00Z", "value": 40.5},
      {"timestamp": "2023-07-01T11:45:00Z", "value": 41.2},
      {"timestamp": "2023-07-01T11:50:00Z", "value": 42.1}
    ],
    "memory_usage_avg": [
      {"timestamp": "2023-07-01T11:40:00Z", "value": 3.0},
      {"timestamp": "2023-07-01T11:45:00Z", "value": 3.1},
      {"timestamp": "2023-07-01T11:50:00Z", "value": 3.1}
    ]
  },
  "nodes": {
    "node-uuid-1": {
      "name": "edge-node-1",
      "cpu_usage_avg": 43.5,
      "memory_usage_avg": 3.2
    },
    "node-uuid-2": {
      "name": "edge-node-2",
      "cpu_usage_avg": 38.7,
      "memory_usage_avg": 2.9
    }
  }
}
```

### Security

#### Generate Token

```
POST /security/tokens
```

Generates a new authentication token.

**Request Body:**
```json
{
  "node_id": "node-uuid-1",
  "expiration": "24h"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2023-07-02T12:45:00Z"
}
```

#### Validate Token

```
POST /security/tokens/validate
```

Validates an authentication token.

**Request Body:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "valid": true,
  "node_id": "node-uuid-1",
  "expires_at": "2023-07-02T12:45:00Z"
}
```

#### Revoke Token

```
POST /security/tokens/revoke
```

Revokes an authentication token.

**Request Body:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "success": true,
  "message": "Token successfully revoked"
}
```

## Error Responses

All API endpoints return standard error responses in the following format:

```json
{
  "error": true,
  "code": "ERROR_CODE",
  "message": "Human-readable error message",
  "details": {
    "field": "Additional error details"
  }
}
```

### Common Error Codes

- `UNAUTHORIZED`: Authentication failed or token expired
- `FORBIDDEN`: Insufficient permissions for the requested operation
- `NOT_FOUND`: Requested resource not found
- `BAD_REQUEST`: Invalid request parameters
- `INTERNAL_ERROR`: Server-side error
- `CONFLICT`: Resource conflict (e.g., duplicate name)
- `SERVICE_UNAVAILABLE`: Service temporarily unavailable

## Rate Limiting

The API implements rate limiting to prevent abuse. When rate limits are exceeded, the API returns a `429 Too Many Requests` status code with headers indicating the rate limit and reset time:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1625140800
```

## Pagination

Endpoints that return collections support pagination using the following query parameters:

- `page`: Page number (starting from 1)
- `limit`: Number of items per page (default: 20, max: 100)

Paginated responses include metadata:

```json
{
  "items": [...],
  "pagination": {
    "total": 42,
    "page": 1,
    "limit": 20,
    "pages": 3
  }
}
```