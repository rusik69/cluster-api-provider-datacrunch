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

package cloud

import (
	"context"
)

// Scope defines the interface for passing scope between methods
type Scope interface {
	GetRegion() string
	GetCredentials() (clientID, clientSecret string)
}

// Client defines the interface for interacting with DataCrunch API
type Client interface {
	// Instance management
	CreateInstance(ctx context.Context, spec *InstanceSpec) (*Instance, error)
	GetInstance(ctx context.Context, instanceID string) (*Instance, error)
	DeleteInstance(ctx context.Context, instanceID string) error
	StartInstance(ctx context.Context, instanceID string) error
	StopInstance(ctx context.Context, instanceID string) error

	// Image management
	ListImages(ctx context.Context) ([]*Image, error)
	GetImage(ctx context.Context, imageID string) (*Image, error)

	// SSH Key management
	ListSSHKeys(ctx context.Context) ([]*SSHKey, error)
	CreateSSHKey(ctx context.Context, name, publicKey string) (*SSHKey, error)
	DeleteSSHKey(ctx context.Context, keyID string) error

	// Network management
	CreateLoadBalancer(ctx context.Context, spec *LoadBalancerSpec) (*LoadBalancer, error)
	GetLoadBalancer(ctx context.Context, lbID string) (*LoadBalancer, error)
	DeleteLoadBalancer(ctx context.Context, lbID string) error
	UpdateLoadBalancerTargets(ctx context.Context, lbID string, targets []string) error
}

// InstanceSpec defines the specification for creating an instance
type InstanceSpec struct {
	Name         string
	InstanceType string
	ImageID      string
	SSHKeyName   string
	UserData     string
	Metadata     map[string]string
	Tags         map[string]string
	PublicIP     bool
}

// Instance represents a DataCrunch instance
type Instance struct {
	ID           string
	Name         string
	State        string
	InstanceType string
	ImageID      string
	PublicIP     string
	PrivateIP    string
	SSHKeyName   string
	CreatedAt    string
	Region       string
}

// Image represents a DataCrunch image
type Image struct {
	ID          string
	Name        string
	Description string
	OSType      string
	CreatedAt   string
}

// SSHKey represents a DataCrunch SSH key
type SSHKey struct {
	ID        string
	Name      string
	PublicKey string
	CreatedAt string
}

// LoadBalancerSpec defines the specification for creating a load balancer
type LoadBalancerSpec struct {
	Name            string
	Type            string
	HealthCheckPath string
	Targets         []string
	Tags            map[string]string
}

// LoadBalancer represents a DataCrunch load balancer
type LoadBalancer struct {
	ID      string
	Name    string
	DNSName string
	State   string
	Type    string
	Targets []string
}
