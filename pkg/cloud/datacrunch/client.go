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

package datacrunch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rusik69/cluster-api-provider-datacrunch/pkg/cloud"
)

const (
	defaultBaseURL = "https://api.datacrunch.io/v1"
	defaultTimeout = 30 * time.Second
)

// Client implements the cloud.Client interface for DataCrunch
type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	token        string
	tokenExpiry  time.Time
}

// NewClient creates a new DataCrunch client
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		baseURL:      defaultBaseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewClientWithURL creates a new DataCrunch client with a custom base URL
func NewClientWithURL(clientID, clientSecret, baseURL string) *Client {
	return &Client{
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// authenticate obtains an access token from DataCrunch
func (c *Client) authenticate(ctx context.Context) error {
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	payload := map[string]string{
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"grant_type":    "client_credentials",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal auth payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/oauth/token", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var authResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.token = authResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	return nil
}

// makeRequest makes an authenticated request to the DataCrunch API
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// CreateInstance creates a new DataCrunch instance
func (c *Client) CreateInstance(ctx context.Context, spec *cloud.InstanceSpec) (*cloud.Instance, error) {
	payload := map[string]interface{}{
		"hostname":      spec.Name,
		"instance_type": spec.InstanceType,
		"image":         spec.ImageID,
		"ssh_key":       spec.SSHKeyName,
		"user_data":     spec.UserData,
	}

	resp, err := c.makeRequest(ctx, "POST", "/instances", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create instance, status: %d", resp.StatusCode)
	}

	var instanceResp struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&instanceResp); err != nil {
		return nil, fmt.Errorf("failed to decode create instance response: %w", err)
	}

	// Return the instance details
	return c.GetInstance(ctx, instanceResp.ID)
}

// GetInstance retrieves an instance by ID
func (c *Client) GetInstance(ctx context.Context, instanceID string) (*cloud.Instance, error) {
	resp, err := c.makeRequest(ctx, "GET", "/instances/"+instanceID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get instance, status: %d", resp.StatusCode)
	}

	var instanceData struct {
		ID           string `json:"id"`
		Hostname     string `json:"hostname"`
		Status       string `json:"status"`
		InstanceType string `json:"instance_type"`
		Image        string `json:"image"`
		PublicIP     string `json:"public_ip"`
		PrivateIP    string `json:"private_ip"`
		SSHKey       string `json:"ssh_key"`
		CreatedAt    string `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&instanceData); err != nil {
		return nil, fmt.Errorf("failed to decode instance response: %w", err)
	}

	return &cloud.Instance{
		ID:           instanceData.ID,
		Name:         instanceData.Hostname,
		State:        instanceData.Status,
		InstanceType: instanceData.InstanceType,
		ImageID:      instanceData.Image,
		PublicIP:     instanceData.PublicIP,
		PrivateIP:    instanceData.PrivateIP,
		SSHKeyName:   instanceData.SSHKey,
		CreatedAt:    instanceData.CreatedAt,
	}, nil
}

// DeleteInstance deletes an instance
func (c *Client) DeleteInstance(ctx context.Context, instanceID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/instances/"+instanceID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete instance, status: %d", resp.StatusCode)
	}

	return nil
}

// StartInstance starts an instance
func (c *Client) StartInstance(ctx context.Context, instanceID string) error {
	resp, err := c.makeRequest(ctx, "POST", "/instances/"+instanceID+"/start", nil)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to start instance, status: %d", resp.StatusCode)
	}

	return nil
}

// StopInstance stops an instance
func (c *Client) StopInstance(ctx context.Context, instanceID string) error {
	resp, err := c.makeRequest(ctx, "POST", "/instances/"+instanceID+"/stop", nil)
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to stop instance, status: %d", resp.StatusCode)
	}

	return nil
}

// ListImages lists available images
func (c *Client) ListImages(ctx context.Context) ([]*cloud.Image, error) {
	resp, err := c.makeRequest(ctx, "GET", "/images", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list images, status: %d", resp.StatusCode)
	}

	var imagesResp struct {
		Images []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			OSType      string `json:"os_type"`
			CreatedAt   string `json:"created_at"`
		} `json:"images"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&imagesResp); err != nil {
		return nil, fmt.Errorf("failed to decode images response: %w", err)
	}

	images := make([]*cloud.Image, len(imagesResp.Images))
	for i, img := range imagesResp.Images {
		images[i] = &cloud.Image{
			ID:          img.ID,
			Name:        img.Name,
			Description: img.Description,
			OSType:      img.OSType,
			CreatedAt:   img.CreatedAt,
		}
	}

	return images, nil
}

// GetImage retrieves an image by ID
func (c *Client) GetImage(ctx context.Context, imageID string) (*cloud.Image, error) {
	resp, err := c.makeRequest(ctx, "GET", "/images/"+imageID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("image not found: %s", imageID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get image, status: %d", resp.StatusCode)
	}

	var imageData struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		OSType      string `json:"os_type"`
		CreatedAt   string `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&imageData); err != nil {
		return nil, fmt.Errorf("failed to decode image response: %w", err)
	}

	return &cloud.Image{
		ID:          imageData.ID,
		Name:        imageData.Name,
		Description: imageData.Description,
		OSType:      imageData.OSType,
		CreatedAt:   imageData.CreatedAt,
	}, nil
}

// ListSSHKeys lists SSH keys
func (c *Client) ListSSHKeys(ctx context.Context) ([]*cloud.SSHKey, error) {
	resp, err := c.makeRequest(ctx, "GET", "/ssh-keys", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list SSH keys, status: %d", resp.StatusCode)
	}

	var keysResp struct {
		Keys []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			PublicKey string `json:"public_key"`
			CreatedAt string `json:"created_at"`
		} `json:"ssh_keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&keysResp); err != nil {
		return nil, fmt.Errorf("failed to decode SSH keys response: %w", err)
	}

	keys := make([]*cloud.SSHKey, len(keysResp.Keys))
	for i, key := range keysResp.Keys {
		keys[i] = &cloud.SSHKey{
			ID:        key.ID,
			Name:      key.Name,
			PublicKey: key.PublicKey,
			CreatedAt: key.CreatedAt,
		}
	}

	return keys, nil
}

// CreateSSHKey creates a new SSH key
func (c *Client) CreateSSHKey(ctx context.Context, name, publicKey string) (*cloud.SSHKey, error) {
	payload := map[string]string{
		"name":       name,
		"public_key": publicKey,
	}

	resp, err := c.makeRequest(ctx, "POST", "/ssh-keys", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create SSH key, status: %d", resp.StatusCode)
	}

	var keyData struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		PublicKey string `json:"public_key"`
		CreatedAt string `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&keyData); err != nil {
		return nil, fmt.Errorf("failed to decode SSH key response: %w", err)
	}

	return &cloud.SSHKey{
		ID:        keyData.ID,
		Name:      keyData.Name,
		PublicKey: keyData.PublicKey,
		CreatedAt: keyData.CreatedAt,
	}, nil
}

// DeleteSSHKey deletes an SSH key
func (c *Client) DeleteSSHKey(ctx context.Context, keyID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/ssh-keys/"+keyID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete SSH key, status: %d", resp.StatusCode)
	}

	return nil
}

// Load balancer methods (placeholder implementations as DataCrunch may not have native LB support)
func (c *Client) CreateLoadBalancer(ctx context.Context, spec *cloud.LoadBalancerSpec) (*cloud.LoadBalancer, error) {
	// This is a placeholder - DataCrunch may not have native load balancer support
	// In a real implementation, you might use an external load balancer service
	return nil, fmt.Errorf("load balancer creation not yet implemented")
}

func (c *Client) GetLoadBalancer(ctx context.Context, lbID string) (*cloud.LoadBalancer, error) {
	return nil, fmt.Errorf("load balancer retrieval not yet implemented")
}

func (c *Client) DeleteLoadBalancer(ctx context.Context, lbID string) error {
	return fmt.Errorf("load balancer deletion not yet implemented")
}

func (c *Client) UpdateLoadBalancerTargets(ctx context.Context, lbID string, targets []string) error {
	return fmt.Errorf("load balancer target update not yet implemented")
}
