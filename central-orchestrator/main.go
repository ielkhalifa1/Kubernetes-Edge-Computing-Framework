package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	DefaultPort = "8443"
	CertPath    = "/etc/certs/tls.crt"
	KeyPath     = "/etc/certs/tls.key"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("Starting Kubernetes Edge Computing Central Orchestrator")

	// Initialize components
	nodeManager := NewNodeManager(logger)
	workloadManager := NewWorkloadManager(logger)
	securityManager := NewSecurityManager(logger)
	monitoringService := NewMonitoringService(logger)

	// Initialize orchestrator
	orchestrator := &CentralOrchestrator{
		NodeManager:        nodeManager,
		WorkloadManager:    workloadManager,
		SecurityManager:    securityManager,
		MonitoringService:  monitoringService,
		Logger:             logger,
	}

	// Setup HTTP router
	router := setupRouter(orchestrator)

	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	// Create HTTPS server
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	server := &http.Server{
		Addr:      ":" + port,
		Handler:   router,
		TLSConfig: tlsConfig,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Starting HTTPS server on port %s", port)
		if err := server.ListenAndServeTLS(CertPath, KeyPath); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start background services
	go orchestrator.StartBackgroundServices()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func setupRouter(orchestrator *CentralOrchestrator) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(orchestrator.SecurityManager.AuthMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"timestamp": time.Now(),
		})
	})

	// Node management endpoints
	v1 := router.Group("/api/v1")
	{
		// Node registration and management
		v1.POST("/nodes/register", orchestrator.RegisterNode)
		v1.GET("/nodes", orchestrator.ListNodes)
		v1.GET("/nodes/:id", orchestrator.GetNode)
		v1.DELETE("/nodes/:id", orchestrator.UnregisterNode)
		v1.POST("/nodes/:id/heartbeat", orchestrator.NodeHeartbeat)

		// Workload management
		v1.POST("/workloads", orchestrator.DeployWorkload)
		v1.GET("/workloads", orchestrator.ListWorkloads)
		v1.GET("/workloads/:id", orchestrator.GetWorkload)
		v1.DELETE("/workloads/:id", orchestrator.DeleteWorkload)
		v1.POST("/workloads/:id/scale", orchestrator.ScaleWorkload)

		// Monitoring and metrics
		v1.GET("/metrics", orchestrator.GetMetrics)
		v1.GET("/nodes/:id/metrics", orchestrator.GetNodeMetrics)
		v1.GET("/workloads/:id/metrics", orchestrator.GetWorkloadMetrics)

		// Security management
		v1.POST("/certificates/issue", orchestrator.IssueCertificate)
		v1.POST("/certificates/revoke", orchestrator.RevokeCertificate)
	}

	return router
}
