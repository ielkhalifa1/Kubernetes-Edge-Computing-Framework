package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DefaultConfigPath = "/etc/edge-agent/config.yaml"
	DefaultHeartbeatInterval = 30 * time.Second
	DefaultTimeout = 10 * time.Second
)

type Config struct {
	OrchestratorURL    string        `yaml:"orchestrator_url"`
	NodeName           string        `yaml:"node_name"`
	NodeAddress        string        `yaml:"node_address"`
	Region             string        `yaml:"region"`
	Zone               string        `yaml:"zone"`
	HeartbeatInterval  time.Duration `yaml:"heartbeat_interval"`
	AuthToken          string        `yaml:"auth_token"`
	TLSCertPath        string        `yaml:"tls_cert_path"`
	TLSKeyPath         string        `yaml:"tls_key_path"`
	KubeconfigPath     string        `yaml:"kubeconfig_path"`
	Labels             map[string]string `yaml:"labels"`
	Capabilities       []string      `yaml:"capabilities"`
}

type EdgeAgent struct {
	config          *Config
	logger          *logrus.Logger
	httpClient      *http.Client
	kubeClient      kubernetes.Interface
	nodeID          string
	registrationCtx context.Context
	cancel          context.CancelFunc
}

type NodeStatus string

const (
	NodeStatusOnline      NodeStatus = "online"
	NodeStatusOffline     NodeStatus = "offline"
	NodeStatusDegraded    NodeStatus = "degraded"
	NodeStatusMaintenance NodeStatus = "maintenance"
)

type NodeResources struct {
	CPU struct {
		Capacity   string  `json:"capacity"`
		Usage      string  `json:"usage"`
		Percentage float64 `json:"percentage"`
	} `json:"cpu"`
	Memory struct {
		Capacity   string  `json:"capacity"`
		Usage      string  `json:"usage"`
		Percentage float64 `json:"percentage"`
	} `json:"memory"`
	Storage struct {
		Capacity   string  `json:"capacity"`
		Usage      string  `json:"usage"`
		Percentage float64 `json:"percentage"`
	} `json:"storage"`
	NetworkBandwidth string `json:"network_bandwidth"`
	GPUs            int    `json:"gpus"`
}

type HeartbeatRequest struct {
	Status    NodeStatus    `json:"status"`
	Resources NodeResources `json:"resources"`
	Timestamp time.Time     `json:"timestamp"`
}

type RegistrationRequest struct {
	Name             string            `json:"name"`
	Address          string            `json:"address"`
	Labels           map[string]string `json:"labels"`
	Capabilities     []string          `json:"capabilities"`
	Region           string            `json:"region"`
	Zone             string            `json:"zone"`
	KubernetesVersion string           `json:"kubernetes_version"`
	ContainerRuntime string            `json:"container_runtime"`
}

type RegistrationResponse struct {
	ID   string `json:"id"`
	Node interface{} `json:"node"`
}

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("Starting Kubernetes Edge Agent")

	// Load configuration
	configPath := os.Getenv("EDGE_AGENT_CONFIG")
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	config, err := loadConfig(configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize edge agent
	agent, err := NewEdgeAgent(config, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize edge agent: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	agent.registrationCtx = ctx
	agent.cancel = cancel

	// Register with central orchestrator
	if err := agent.register(); err != nil {
		logger.Fatalf("Failed to register with orchestrator: %v", err)
	}

	// Start background services
	go agent.startHeartbeat()
	go agent.startResourceMonitoring()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down edge agent...")
	cancel()

	// Give some time for cleanup
	time.Sleep(2 * time.Second)
	logger.Info("Edge agent stopped")
}

func loadConfig(path string) (*Config, error) {
	// Set defaults
	config := &Config{
		HeartbeatInterval: DefaultHeartbeatInterval,
		Labels:           make(map[string]string),
		Capabilities:     []string{},
		Region:           "default",
		Zone:             "default",
	}

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Use environment variables if config file doesn't exist
		config.OrchestratorURL = os.Getenv("ORCHESTRATOR_URL")
		config.NodeName = os.Getenv("NODE_NAME")
		config.NodeAddress = os.Getenv("NODE_ADDRESS")
		config.AuthToken = os.Getenv("AUTH_TOKEN")
		
		if config.OrchestratorURL == "" {
			return nil, fmt.Errorf("ORCHESTRATOR_URL is required")
		}
		if config.NodeName == "" {
			return nil, fmt.Errorf("NODE_NAME is required")
		}
		if config.NodeAddress == "" {
			return nil, fmt.Errorf("NODE_ADDRESS is required")
		}
		
		return config, nil
	}

	// Load from YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return config, nil
}

func NewEdgeAgent(config *Config, logger *logrus.Logger) (*EdgeAgent, error) {
	// Create HTTP client with TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // For demo purposes, in production verify certificates
	}

	httpClient := &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Initialize Kubernetes client
	var kubeClient kubernetes.Interface
	var err error

	if config.KubeconfigPath != "" {
		kubeconfig, err := clientcmd.BuildConfigFromFlags("", config.KubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %v", err)
		}
		kubeClient, err = kubernetes.NewForConfig(kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
		}
	} else {
		// Use in-cluster config
		kubeconfig, err := rest.InClusterConfig()
		if err != nil {
			logger.Warnf("Failed to get in-cluster config: %v", err)
		} else {
			kubeClient, err = kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				logger.Warnf("Failed to create in-cluster Kubernetes client: %v", err)
			}
		}
	}

	return &EdgeAgent{
		config:     config,
		logger:     logger,
		httpClient: httpClient,
		kubeClient: kubeClient,
	}, nil
}

func (ea *EdgeAgent) register() error {
	ea.logger.Info("Registering with central orchestrator")

	// Get Kubernetes version and container runtime info
	k8sVersion := "unknown"
	containerRuntime := "unknown"

	if ea.kubeClient != nil {
		if version, err := ea.kubeClient.Discovery().ServerVersion(); err == nil {
			k8sVersion = version.String()
		}
	}

	// For simplicity, assume containerd
	containerRuntime = "containerd"

	req := RegistrationRequest{
		Name:             ea.config.NodeName,
		Address:          ea.config.NodeAddress,
		Labels:           ea.config.Labels,
		Capabilities:     ea.config.Capabilities,
		Region:           ea.config.Region,
		Zone:             ea.config.Zone,
		KubernetesVersion: k8sVersion,
		ContainerRuntime: containerRuntime,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", ea.config.OrchestratorURL+"/api/v1/nodes/register", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+ea.config.AuthToken)

	resp, err := ea.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send registration request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	var regResp RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to decode registration response: %v", err)
	}

	ea.nodeID = regResp.ID
	ea.logger.Infof("Successfully registered with node ID: %s", ea.nodeID)

	return nil
}

func (ea *EdgeAgent) startHeartbeat() {
	ticker := time.NewTicker(ea.config.HeartbeatInterval)
	defer ticker.Stop()

	ea.logger.Info("Starting heartbeat service")

	for {
		select {
		case <-ea.registrationCtx.Done():
			return
		case <-ticker.C:
			if err := ea.sendHeartbeat(); err != nil {
				ea.logger.Errorf("Failed to send heartbeat: %v", err)
			}
		}
	}
}

func (ea *EdgeAgent) sendHeartbeat() error {
	resources, err := ea.collectResources()
	if err != nil {
		ea.logger.Warnf("Failed to collect resources: %v", err)
		resources = NodeResources{} // Send empty resources on error
	}

	req := HeartbeatRequest{
		Status:    NodeStatusOnline,
		Resources: resources,
		Timestamp: time.Now(),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat request: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/nodes/%s/heartbeat", ea.config.OrchestratorURL, ea.nodeID)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+ea.config.AuthToken)

	resp, err := ea.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (ea *EdgeAgent) collectResources() (NodeResources, error) {
	var resources NodeResources

	// Collect CPU information
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		resources.CPU.Percentage = cpuPercent[0]
		resources.CPU.Usage = fmt.Sprintf("%.1f%%", cpuPercent[0])
		resources.CPU.Capacity = "100%" // Simplified
	}

	// Collect memory information
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		resources.Memory.Capacity = fmt.Sprintf("%.0f MB", float64(memInfo.Total)/1024/1024)
		resources.Memory.Usage = fmt.Sprintf("%.0f MB", float64(memInfo.Used)/1024/1024)
		resources.Memory.Percentage = memInfo.UsedPercent
	}

	// Collect disk information
	diskInfo, err := disk.Usage("/")
	if err == nil {
		resources.Storage.Capacity = fmt.Sprintf("%.0f GB", float64(diskInfo.Total)/1024/1024/1024)
		resources.Storage.Usage = fmt.Sprintf("%.0f GB", float64(diskInfo.Used)/1024/1024/1024)
		resources.Storage.Percentage = diskInfo.UsedPercent
	}

	// Collect network information (simplified)
	netStats, err := net.IOCounters(false)
	if err == nil && len(netStats) > 0 {
		resources.NetworkBandwidth = "1 Gbps" // Simplified
	}

	// GPU count (simplified - would need proper GPU detection)
	resources.GPUs = 0

	return resources, nil
}

func (ea *EdgeAgent) startResourceMonitoring() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	ea.logger.Info("Starting resource monitoring")

	for {
		select {
		case <-ea.registrationCtx.Done():
			return
		case <-ticker.C:
			resources, err := ea.collectResources()
			if err != nil {
				ea.logger.Errorf("Failed to collect resources: %v", err)
				continue
			}

			ea.logger.Infof("Resource usage: CPU %.1f%%, Memory %.1f%%, Storage %.1f%%",
				resources.CPU.Percentage,
				resources.Memory.Percentage,
				resources.Storage.Percentage)
		}
	}
}
