package packager

import (
	"testing"
	"time"
)

// TestNewPackager tests packager initialization
func TestNewPackager(t *testing.T) {
	p, err := NewPackager("", "")
	if err != nil {
		t.Errorf("NewPackager failed: %v", err)
	}
	if p == nil {
		t.Error("NewPackager returned nil")
	}
}

// TestPackageInfo tests package info structure
func TestPackageInfo(t *testing.T) {
	info := PackageInfo{
		Name:        "test",
		Version:     "1.0.0",
		Language:    "python",
		Registry:    RegistryPyPI,
		InstallDate: time.Now(),
		Path:        "/test",
	}

	if info.Name != "test" {
		t.Errorf("Expected name 'test', got %s", info.Name)
	}
	if info.Registry != RegistryPyPI {
		t.Errorf("Expected registry %s, got %s", RegistryPyPI, info.Registry)
	}
}

// TestParsePackageVersion tests version parsing
func TestParsePackageVersion(t *testing.T) {
	cases := []struct {
		input       string
		expectedName string
		expectedVer  string
	}{
		{"requests", "requests", ""},
		{"requests@v2.0.0", "requests", "v2.0.0"},
		{"lodash@4.17.21", "lodash", "4.17.21"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			name, version := parsePackageVersion(tc.input)
			if name != tc.expectedName {
				t.Errorf("Expected name %s, got %s", tc.expectedName, name)
			}
			if version != tc.expectedVer {
				t.Errorf("Expected version %s, got %s", tc.expectedVer, version)
			}
		})
	}
}

// TestDetectGitHubLanguage tests language detection
func TestDetectGitHubLanguage(t *testing.T) {
	p := &Packager{}

	cases := []struct {
		input    string
		expected string
	}{
		{"github.com/user/mux-go", "go"},
		{"github.com/user/tokio-rs", "rust"},
		{"github.com/user/rails-rb", "ruby"},
		{"github.com/user/numpy-py", "python"},
		{"github.com/user/react-js", "javascript"},
		{"github.com/user/unknown", "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result := p.detectGitHubLanguage(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s for %s", tc.expected, result, tc.input)
			}
		})
	}
}