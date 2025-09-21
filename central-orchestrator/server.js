const express = require('express');
const https = require('https');
const fs = require('fs');
const bodyParser = require('body-parser');
const cors = require('cors');
const helmet = require('helmet');
const winston = require('winston');
const { NodeManager } = require('./nodeManager');
const { WorkloadManager } = require('./workloadManager');
const { SecurityManager } = require('./securityManager');
const { MonitoringService } = require('./monitoringService');
const { setupRoutes } = require('./routes');

// Constants
const DEFAULT_PORT = process.env.PORT || 8443;
const CERT_PATH = process.env.CERT_PATH || './certs/tls.crt';
const KEY_PATH = process.env.KEY_PATH || './certs/tls.key';

// Initialize logger
const logger = winston.createLogger({
  level: 'info',
  format: winston.format.json(),
  defaultMeta: { service: 'central-orchestrator' },
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

// Initialize components
const nodeManager = new NodeManager(logger);
const workloadManager = new WorkloadManager(logger);
const securityManager = new SecurityManager(logger);
const monitoringService = new MonitoringService(logger);

// Initialize orchestrator
const orchestrator = {
  nodeManager,
  workloadManager,
  securityManager,
  monitoringService,
  logger
};

// Create Express app
const app = express();

// Middleware
app.use(helmet());
app.use(cors());
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: true }));

// Setup routes
setupRoutes(app, orchestrator);

// Start background services
nodeManager.startHealthChecker();
monitoringService.startMetricsCollection();

// Create HTTPS server
let server;
try {
  const httpsOptions = {
    key: fs.readFileSync(KEY_PATH),
    cert: fs.readFileSync(CERT_PATH)
  };
  server = https.createServer(httpsOptions, app);
} catch (error) {
  logger.warn(`Unable to load SSL certificates: ${error.message}. Falling back to HTTP.`);
  server = require('http').createServer(app);
}

// Start server
server.listen(DEFAULT_PORT, () => {
  logger.info(`Kubernetes Edge Computing Central Orchestrator running on port ${DEFAULT_PORT}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM signal received: closing HTTP server');
  server.close(() => {
    logger.info('HTTP server closed');
  });
});

process.on('SIGINT', () => {
  logger.info('SIGINT signal received: closing HTTP server');
  server.close(() => {
    logger.info('HTTP server closed');
  });
});

module.exports = { app, server };