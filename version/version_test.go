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

package version

import (
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	// Test that we get a valid Info struct
	if info.GitVersion == "" {
		t.Error("GitVersion should not be empty")
	}

	// Default values should be set
	if info.GitCommit == "" {
		t.Error("GitCommit should not be empty (should default to 'unknown')")
	}

	if info.BuildDate == "" {
		t.Error("BuildDate should not be empty (should default to 'unknown')")
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		version  Version
		expected []string // Parts that should be in the string
	}{
		{
			name: "default values",
			version: Version{
				GitVersion: "v0.1.0",
				GitCommit:  "unknown",
				BuildDate:  "unknown",
			},
			expected: []string{"v0.1.0", "commit unknown", "built at unknown"},
		},
		{
			name: "with git commit",
			version: Version{
				GitVersion: "v1.2.3",
				GitCommit:  "abc123def",
				BuildDate:  "2024-01-15T10:30:00Z",
			},
			expected: []string{"v1.2.3", "commit abc123def", "built at 2024-01-15T10:30:00Z"},
		},
		{
			name: "development version",
			version: Version{
				GitVersion: "v0.0.0-dev",
				GitCommit:  "dirty",
				BuildDate:  "now",
			},
			expected: []string{"v0.0.0-dev", "commit dirty", "built at now"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("String() = %q, should contain %q", result, expected)
				}
			}
		})
	}
}

func TestStringFormat(t *testing.T) {
	version := Version{
		GitVersion: "v1.0.0",
		GitCommit:  "abcdef123",
		BuildDate:  "2024-06-28T12:00:00Z",
	}

	result := version.String()
	expected := "v1.0.0, commit abcdef123, built at 2024-06-28T12:00:00Z"

	if result != expected {
		t.Errorf("String() = %q, want %q", result, expected)
	}
}

func TestGetReturnsConsistentInfo(t *testing.T) {
	// Test that multiple calls to Get() return the same info
	info1 := Get()
	info2 := Get()

	if info1.GitVersion != info2.GitVersion {
		t.Error("GitVersion should be consistent between calls")
	}

	if info1.GitCommit != info2.GitCommit {
		t.Error("GitCommit should be consistent between calls")
	}

	if info1.BuildDate != info2.BuildDate {
		t.Error("BuildDate should be consistent between calls")
	}
}
