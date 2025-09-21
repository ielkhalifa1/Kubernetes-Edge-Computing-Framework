const axios = require('axios');
const winston = require('winston');
const cron = require('node-cron');
const os = require('os');
const osUtils = require('os-utils');
const { v4: uuidv4 } = require('uuid');
const fs = require('fs');
const path = require('path');
const { KubeConfig, CoreV1Api } = require('kubernetes-client');

// Constants
const ORCHESTRATOR_URL = process.env.ORCHESTRATOR_URL || 'https://orchestrator:8443';
const NODE_NAME = process.env.NODE_NAME || os.hostname();
const CONFIG_PATH = process.env.CONFIG_PATH || './config.json';

// Initialize logger
const logger = winston.createLogger({
  level: 'info',
  format: winston.format.json(),
  defaultMeta: { service: 'edge-agent' },
  transports: [
    new winston.transports.Console({
      format: winston.format.combine(
        winston.format.colorize(),
        winston.format.simple()
      )
    }),
    new winston.transports.File({ filename: 'error.log', level: 'error' }),
    new winston.transports.File({ filename: 'combined.log' })
  ]
});

class EdgeAgent {
  constructor() {
    this.nodeId = null;
    this.token = null;
    this.status = 'INITIALIZING';
    this.workloads = [];
    this.capabilities = this.detectCapabilities();
    
    // Load config if exists
    this.loadConfig();
    
    // Initialize Kubernetes client
    this.initKubernetesClient();
  }

  // Initialize Kubernetes client
  initKubernetesClient() {
    try {
      this.kubeConfig = new KubeConfig();
      this.kubeConfig.loadFromDefault();
      this.k8sApi = this.kubeConfig.makeApiClient(CoreV1Api);
      logger.info('Kubernetes client initialized');
    } catch (error) {
      logger.error(`Failed to initialize Kubernetes client: ${error.message}`);
    }
  }

  // Load configuration
  loadConfig() {
    try {
      if (fs.existsSync(CONFIG_PATH)) {
        const config = JSON.parse(fs.readFileSync(CONFIG_PATH, 'utf8'));
        this.nodeId = config.nodeId;
        this.token = config.token;
        logger.info('Loaded configuration from file');
      }
    } catch (error) {
      logger.error(`Failed to load config: ${error.message}`);
    }
  }

  // Save configuration
  saveConfig() {
    try {
      const config = {
        nodeId: this.nodeId,
        token: this.token
      };
      fs.writeFileSync(CONFIG_PATH, JSON.stringify(config, null, 2));
      logger.info('Saved configuration to file');
    } catch (error) {
      logger.error(`Failed to save config: ${error.message}`);
    }
  }

  // Detect node capabilities
  detectCapabilities() {
    return {
      cpu: {
        cores: os.cpus().length,
        model: os.cpus()[0].model,
        architecture: os.arch()
      },
      memory: {
        total: os.totalmem(),
        free: os.freemem()
      },
      os: {
        type: os.type(),
        platform: os.platform(),
        release: os.release()
      },
      network: {
        interfaces: Object.keys(os.networkInterfaces())
      }
    };
  }

  // Register with orchestrator
  async register() {
    try {
      // Skip if already registered
      if (this.nodeId && this.token) {
        logger.info(`Already registered with ID: ${this.nodeId}`);
        return;
      }

      const response = await axios.post(`${ORCHESTRATOR_URL}/api/nodes/register`, {
        name: NODE_NAME,
        ipAddress: this.getIpAddress(),
        capabilities: this.capabilities
      });

      this.nodeId = response.data.nodeId;
      this.token = response.data.token;
      this.status = 'ONLINE';
      
      // Save configuration
      this.saveConfig();
      
      logger.info(`Registered with orchestrator, node ID: ${this.nodeId}`);
    } catch (error) {
      logger.error(`Registration failed: ${error.message}`);
      throw error;
    }
  }

  // Get IP address
  getIpAddress() {
    const interfaces = os.networkInterfaces();
    for (const name of Object.keys(interfaces)) {
      for (const iface of interfaces[name]) {
        if (iface.family === 'IPv4' && !iface.internal) {
          return iface.address;
        }
      }
    }
    return '127.0.0.1';
  }

  // Send heartbeat to orchestrator
  async sendHeartbeat() {
    try {
      if (!this.nodeId || !this.token) {
        logger.warn('Cannot send heartbeat: not registered');
        return;
      }

      await axios.put(
        `${ORCHESTRATOR_URL}/api/nodes/${this.nodeId}/status`,
        { status: this.status },
        {
          headers: {
            'x-node-id': this.nodeId,
            'x-auth-token': this.token
          }
        }
      );
      
      logger.debug('Heartbeat sent successfully');
    } catch (error) {
      logger.error(`Failed to send heartbeat: ${error.message}`);
    }
  }

  // Send metrics to orchestrator
  async sendMetrics() {
    try {
      if (!this.nodeId || !this.token) {
        logger.warn('Cannot send metrics: not registered');
        return;
      }

      // Collect current metrics
      const metrics = await this.collectMetrics();

      await axios.post(
        `${ORCHESTRATOR_URL}/api/nodes/${this.nodeId}/metrics`,
        metrics,
        {
          headers: {
            'x-node-id': this.nodeId,
            'x-auth-token': this.token
          }
        }
      );
      
      logger.debug('Metrics sent successfully');
    } catch (error) {
      logger.error(`Failed to send metrics: ${error.message}`);
    }
  }

  // Collect system metrics
  async collectMetrics() {
    return new Promise((resolve) => {
      osUtils.cpuUsage((cpuUsage) => {
        const metrics = {
          timestamp: new Date(),
          cpu: {
            usage: cpuUsage,
            loadavg: os.loadavg()
          },
          memory: {
            total: os.totalmem(),
            free: os.freemem(),
            usage: 1 - (os.freemem() / os.totalmem())
          },
          uptime: os.uptime(),
          workloads: this.workloads.length
        };
        
        resolve(metrics);
      });
    });
  }

  // Start the agent
  async start() {
    logger.info('Starting Edge Agent');
    
    try {
      // Register with orchestrator
      await this.register();
      
      // Schedule heartbeat (every minute)
      cron.schedule('* * * * *', () => {
        this.sendHeartbeat();
      });
      
      // Schedule metrics collection (every 5 minutes)
      cron.schedule('*/5 * * * *', () => {
        this.sendMetrics();
      });
      
      logger.info('Edge Agent started successfully');
    } catch (error) {
      logger.error(`Failed to start Edge Agent: ${error.message}`);
      // Retry after 30 seconds
      setTimeout(() => this.start(), 30000);
    }
  }
}

// Create and start the agent
const agent = new EdgeAgent();
agent.start().catch(err => {
  logger.error(`Error starting agent: ${err.message}`);
  process.exit(1);
});

// Handle graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM received, shutting down');
  process.exit(0);
});

process.on('SIGINT', () => {
  logger.info('SIGINT received, shutting down');
  process.exit(0);
});

module.exports = { agent };