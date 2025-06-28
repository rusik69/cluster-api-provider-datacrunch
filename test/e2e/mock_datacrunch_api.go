/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/rusik69/cluster-api-provider-datacrunch/pkg/cloud"
)

// MockDataCrunchAPI provides a mock implementation of the DataCrunch API
type MockDataCrunchAPI struct {
	server    *httptest.Server
	instances map[string]*cloud.Instance
	images    map[string]*cloud.Image
	sshKeys   map[string]*cloud.SSHKey
	lbs       map[string]*cloud.LoadBalancer
	mutex     sync.RWMutex
}

// NewMockDataCrunchAPI creates a new mock API server
func NewMockDataCrunchAPI() *MockDataCrunchAPI {
	mock := &MockDataCrunchAPI{
		instances: make(map[string]*cloud.Instance),
		images:    make(map[string]*cloud.Image),
		sshKeys:   make(map[string]*cloud.SSHKey),
		lbs:       make(map[string]*cloud.LoadBalancer),
	}

	// Pre-populate with some test data
	mock.setupTestData()

	// Create HTTP server
	mux := http.NewServeMux()
	mock.setupRoutes(mux)
	mock.server = httptest.NewServer(mux)

	return mock
}

// Close shuts down the mock server
func (m *MockDataCrunchAPI) Close() {
	m.server.Close()
}

// URL returns the mock server URL
func (m *MockDataCrunchAPI) URL() string {
	return m.server.URL
}

// setupTestData pre-populates the mock with test data
func (m *MockDataCrunchAPI) setupTestData() {
	// Add test images
	m.images["ubuntu-22.04-cuda-12.1"] = &cloud.Image{
		ID:          "ubuntu-22.04-cuda-12.1",
		Name:        "Ubuntu 22.04 with CUDA 12.1",
		Description: "Ubuntu 22.04 LTS with CUDA 12.1 and ML frameworks",
		OSType:      "linux",
		CreatedAt:   "2024-01-01T00:00:00Z",
	}

	m.images["ubuntu-20.04"] = &cloud.Image{
		ID:          "ubuntu-20.04",
		Name:        "Ubuntu 20.04 LTS",
		Description: "Ubuntu 20.04 LTS base image",
		OSType:      "linux",
		CreatedAt:   "2024-01-01T00:00:00Z",
	}

	// Add test SSH keys
	m.sshKeys["test-key"] = &cloud.SSHKey{
		ID:        "key-123",
		Name:      "test-key",
		PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCtest...",
		CreatedAt: "2024-01-01T00:00:00Z",
	}
}

// setupRoutes configures the mock API routes
func (m *MockDataCrunchAPI) setupRoutes(mux *http.ServeMux) {
	// Authentication
	mux.HandleFunc("/oauth/token", m.handleAuth)

	// Instances
	mux.HandleFunc("/instances", m.handleInstances)
	mux.HandleFunc("/instances/", m.handleInstanceByID)

	// Images
	mux.HandleFunc("/images", m.handleImages)
	mux.HandleFunc("/images/", func(w http.ResponseWriter, r *http.Request) {
		imageID := strings.TrimPrefix(r.URL.Path, "/images/")
		m.handleImageByID(w, r, imageID)
	})

	// SSH Keys
	mux.HandleFunc("/ssh-keys", m.handleSSHKeys)
	mux.HandleFunc("/ssh-keys/", func(w http.ResponseWriter, r *http.Request) {
		keyID := strings.TrimPrefix(r.URL.Path, "/ssh-keys/")
		m.handleSSHKeyByID(w, r, keyID)
	})

	// Load Balancers
	mux.HandleFunc("/load-balancers", m.handleLoadBalancers)
	mux.HandleFunc("/load-balancers/", func(w http.ResponseWriter, r *http.Request) {
		lbID := strings.TrimPrefix(r.URL.Path, "/load-balancers/")
		m.handleLoadBalancerByID(w, r, lbID)
	})
}

// Authentication endpoint
func (m *MockDataCrunchAPI) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"access_token": "mock-token-12345",
		"token_type":   "Bearer",
		"expires_in":   3600,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Instance handlers
func (m *MockDataCrunchAPI) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.listInstances(w, r)
	case http.MethodPost:
		m.createInstance(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockDataCrunchAPI) handleInstanceByID(w http.ResponseWriter, r *http.Request) {
	instanceID := strings.TrimPrefix(r.URL.Path, "/instances/")

	// Handle instance actions
	if strings.Contains(instanceID, "/") {
		parts := strings.Split(instanceID, "/")
		if len(parts) >= 2 {
			instanceID = parts[0]
			action := parts[1]
			m.handleInstanceAction(w, r, instanceID, action)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		m.getInstance(w, r, instanceID)
	case http.MethodDelete:
		m.deleteInstance(w, r, instanceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockDataCrunchAPI) listInstances(w http.ResponseWriter, r *http.Request) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	instances := make([]*cloud.Instance, 0, len(m.instances))
	for _, instance := range m.instances {
		instances = append(instances, instance)
	}

	response := map[string]interface{}{
		"instances": instances,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockDataCrunchAPI) createInstance(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	instanceID := fmt.Sprintf("instance-%d", time.Now().Unix())

	// Handle field name mapping from client request
	name := ""
	if hostname, ok := req["hostname"].(string); ok {
		name = hostname
	} else if nameVal, ok := req["name"].(string); ok {
		name = nameVal
	}

	sshKey := ""
	if sshKeyName, ok := req["ssh_key"].(string); ok {
		sshKey = sshKeyName
	} else if sshKeyName, ok := req["ssh_key_name"].(string); ok {
		sshKey = sshKeyName
	}

	instance := &cloud.Instance{
		ID:           instanceID,
		Name:         name,
		State:        "pending",
		InstanceType: req["instance_type"].(string),
		ImageID:      req["image"].(string),
		PublicIP:     "192.168.1.100",
		PrivateIP:    "10.0.1.100",
		SSHKeyName:   sshKey,
		CreatedAt:    time.Now().Format(time.RFC3339),
		Region:       "FIN-01",
	}

	m.mutex.Lock()
	m.instances[instanceID] = instance
	m.mutex.Unlock()

	// Simulate async provisioning - instance will become running after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		m.mutex.Lock()
		if inst, exists := m.instances[instanceID]; exists {
			inst.State = "running"
		}
		m.mutex.Unlock()
	}()

	// Return the instance ID in the response format expected by the client
	response := map[string]interface{}{
		"id": instanceID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (m *MockDataCrunchAPI) getInstance(w http.ResponseWriter, r *http.Request, instanceID string) {
	m.mutex.RLock()
	instance, exists := m.instances[instanceID]
	m.mutex.RUnlock()

	if !exists {
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instance)
}

func (m *MockDataCrunchAPI) deleteInstance(w http.ResponseWriter, r *http.Request, instanceID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.instances[instanceID]; !exists {
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	delete(m.instances, instanceID)
	w.WriteHeader(http.StatusNoContent)
}

func (m *MockDataCrunchAPI) handleInstanceAction(w http.ResponseWriter, r *http.Request, instanceID, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.instances[instanceID]
	if !exists {
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	switch action {
	case "start":
		instance.State = "running"
	case "stop":
		instance.State = "stopped"
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Image handlers
func (m *MockDataCrunchAPI) handleImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	images := make([]*cloud.Image, 0, len(m.images))
	for _, image := range m.images {
		images = append(images, image)
	}

	response := map[string]interface{}{
		"images": images,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockDataCrunchAPI) handleImageByID(w http.ResponseWriter, r *http.Request, imageID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mutex.RLock()
	image, exists := m.images[imageID]
	m.mutex.RUnlock()

	if !exists {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(image)
}

// SSH Key handlers
func (m *MockDataCrunchAPI) handleSSHKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.listSSHKeys(w, r)
	case http.MethodPost:
		m.createSSHKey(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockDataCrunchAPI) handleSSHKeyByID(w http.ResponseWriter, r *http.Request, keyID string) {
	switch r.Method {
	case http.MethodDelete:
		m.deleteSSHKey(w, r, keyID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockDataCrunchAPI) listSSHKeys(w http.ResponseWriter, r *http.Request) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	keys := make([]*cloud.SSHKey, 0, len(m.sshKeys))
	for _, key := range m.sshKeys {
		keys = append(keys, key)
	}

	response := map[string]interface{}{
		"ssh_keys": keys,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockDataCrunchAPI) createSSHKey(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	keyID := fmt.Sprintf("key-%d", time.Now().Unix())

	sshKey := &cloud.SSHKey{
		ID:        keyID,
		Name:      req["name"].(string),
		PublicKey: req["public_key"].(string),
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	m.mutex.Lock()
	m.sshKeys[keyID] = sshKey
	m.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sshKey)
}

func (m *MockDataCrunchAPI) deleteSSHKey(w http.ResponseWriter, r *http.Request, keyID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.sshKeys[keyID]; !exists {
		http.Error(w, "SSH key not found", http.StatusNotFound)
		return
	}

	delete(m.sshKeys, keyID)
	w.WriteHeader(http.StatusNoContent)
}

// Load Balancer handlers (placeholder implementations)
func (m *MockDataCrunchAPI) handleLoadBalancers(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"error": "Load balancer operations not yet implemented",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}

func (m *MockDataCrunchAPI) handleLoadBalancerByID(w http.ResponseWriter, r *http.Request, lbID string) {
	response := map[string]interface{}{
		"error": "Load balancer operations not yet implemented",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}
