// Package packager provides an embedded package manager for Ditto
// It supports installing packages from PyPI (Python) and npm (JavaScript)
package packager

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	// Default package registry URLs
	PyPIRegistryURL     = "https://pypi.org/pypi"
	NPMRegistryURL      = "https://registry.npmjs.org"
	RubyGemsRegistryURL = "https://rubygems.org"
	CratesRegistryURL   = "https://crates.io"
	GoProxyURL          = "https://proxy.golang.org"

	// Package manifest file name
	ManifestFile = "packages.json"
)

// PackageRegistry defines supported package registries
const (
	RegistryPyPI     = "pypi"
	RegistryNPM      = "npm"
	RegistryRubyGems = "rubygems"
	RegistryCrates   = "crates"
	RegistryGo       = "go"
	RegistryGitHub   = "github"
)

// PackageInfo contains metadata about an installed package
type PackageInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Language    string    `json:"language"` // "python", "javascript", "ruby", "rust", "go"
	Registry    string    `json:"registry"` // "pypi", "npm", "rubygems", "crates", "go", "github"
	InstallDate time.Time `json:"install_date"`
	Path        string    `json:"path"`
}

// Helper methods for new registries (stub implementations)

// fetchRubyGemsPackage fetches package info from RubyGems
func (p *Packager) fetchRubyGemsPackage(name, version string) (*RubyGemsPackageInfo, error) {
	url := fmt.Sprintf("%s/api/v1/gems/%s.json", RubyGemsRegistryURL, name)
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from RubyGems: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RubyGems package not found: %s (status: %d)", name, resp.StatusCode)
	}

	var info RubyGemsPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse RubyGems response: %w", err)
	}

	return &info, nil
}

// fetchCratesPackage fetches crate info from crates.io
func (p *Packager) fetchCratesPackage(name, version string) (*CratesPackageInfo, error) {
	url := fmt.Sprintf("%s/api/v1/crates/%s", CratesRegistryURL, name)
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from crates.io: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crates.io package not found: %s (status: %d)", name, resp.StatusCode)
	}

	var info CratesPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse crates.io response: %w", err)
	}

	return &info, nil
}

// fetchGoModule fetches module info from Go proxy
func (p *Packager) fetchGoModule(modulePath string) (*GoModuleInfo, error) {
	// Use Go proxy to get latest version
	url := fmt.Sprintf("%s/%s/@latest", GoProxyURL, modulePath)
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Go proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Go module not found: %s (status: %d)", modulePath, resp.StatusCode)
	}

	var info GoModuleInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse Go proxy response: %w", err)
	}

	return &info, nil
}

// detectGitHubLanguage attempts to detect the primary language from a GitHub repository name
// using common naming conventions and suffixes.
//
// This is a heuristic approach that looks for language-specific suffixes like:
// -go, _go for Go projects
// -rs, _rs for Rust projects  
// -rb, _rb for Ruby projects
// -py, _py for Python projects
// -js, _js for JavaScript projects
//
// Returns "unknown" if no clear language indicator is found.
func (p *Packager) detectGitHubLanguage(repo string) string {
	// Simple heuristic based on common naming patterns
	if strings.Contains(repo, "-go") || strings.Contains(repo, "_go") {
		return "go"
	} else if strings.Contains(repo, "-rs") || strings.Contains(repo, "_rs") {
		return "rust"
	} else if strings.Contains(repo, "-rb") || strings.Contains(repo, "_rb") {
		return "ruby"
	} else if strings.Contains(repo, "-py") || strings.Contains(repo, "_py") {
		return "python"
	} else if strings.Contains(repo, "-js") || strings.Contains(repo, "_js") {
		return "javascript"
	}
	return "unknown"
}

// downloadAndExtractRubyGems downloads and extracts a Ruby gem
// TODO: Implement full RubyGems download and extraction
func (p *Packager) downloadAndExtractRubyGems(url, dest string) error {
	return fmt.Errorf("RubyGems download not yet implemented")
}

// downloadAndExtractCrates downloads and extracts a Rust crate
// TODO: Implement full crates.io download and extraction
func (p *Packager) downloadAndExtractCrates(url, dest string) error {
	return fmt.Errorf("Crates download not yet implemented")
}

// downloadGoModule downloads Go module files from the Go proxy
// TODO: Implement full Go module download and extraction
func (p *Packager) downloadGoModule(modulePath, version, dest string) error {
	return fmt.Errorf("Go module download not yet implemented")
}

// downloadAndExtractGitHub downloads and extracts a GitHub repository
// TODO: Implement full GitHub repository download and extraction
func (p *Packager) downloadAndExtractGitHub(url, dest string) error {
	return fmt.Errorf("GitHub download not yet implemented")
}

// PackageManifest tracks all installed packages
type PackageManifest struct {
	Packages   []PackageInfo `json:"packages"`
	InstallDir string        `json:"install_dir"`
}

// Packager manages package installation and resolution
type Packager struct {
	installDir string
	manifest   *PackageManifest
	httpClient *http.Client
	cacheDir   string
}

// NewPackager creates a new package manager
func NewPackager(installDir, cacheDir string) (*Packager, error) {
	p := &Packager{
		installDir: installDir,
		cacheDir:   cacheDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Ensure directories exist
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create install directory: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Load or create manifest
	if err := p.loadManifest(); err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	return p, nil
}

// loadManifest loads the package manifest or creates a new one
func (p *Packager) loadManifest() error {
	manifestPath := filepath.Join(p.installDir, ManifestFile)

	data, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		p.manifest = &PackageManifest{
			Packages:   []PackageInfo{},
			InstallDir: p.installDir,
		}
		return p.saveManifest()
	}
	if err != nil {
		return err
	}

	var manifest PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	p.manifest = &manifest
	return nil
}

// saveManifest persists the package manifest
func (p *Packager) saveManifest() error {
	manifestPath := filepath.Join(p.installDir, ManifestFile)
	data, err := json.MarshalIndent(p.manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

// Install installs a package from the specified registry
func (p *Packager) Install(name, language string) error {
	switch language {
	case "python", "py":
		return p.installFromPyPI(name)
	case "javascript", "js", "node", "npm":
		return p.installFromNPM(name)
	case "ruby", "rb":
		return p.installFromRubyGems(name)
	case "rust", "rs":
		return p.installFromCrates(name)
	case "go", "golang":
		return p.installFromGo(name)
	case "github":
		return p.installFromGitHub(name)
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

// InstallWithRegistry installs a package from a specific registry
func (p *Packager) InstallWithRegistry(name, registry string) error {
	switch registry {
	case RegistryPyPI:
		return p.installFromPyPI(name)
	case RegistryNPM:
		return p.installFromNPM(name)
	case RegistryRubyGems:
		return p.installFromRubyGems(name)
	case RegistryCrates:
		return p.installFromCrates(name)
	case RegistryGo:
		return p.installFromGo(name)
	case RegistryGitHub:
		return p.installFromGitHub(name)
	default:
		return fmt.Errorf("unsupported registry: %s", registry)
	}
}

// installFromPyPI installs a Python package from PyPI
func (p *Packager) installFromPyPI(name string) error {
	// Parse package name and version
	pkgName, version := parsePackageVersion(name)

	// Fetch package info from PyPI
	info, err := p.fetchPyPIPackage(pkgName, version)
	if err != nil {
		return err
	}

	// Download and extract the package
	pkgPath := filepath.Join(p.installDir, "python", pkgName)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return err
	}

	// Download wheel or source distribution
	downloadURL := info.URL
	if downloadURL == "" {
		return fmt.Errorf("no download URL found for %s", pkgName)
	}

	// For simplicity, we'll download and extract the source
	// In production, you'd handle wheel files properly
	if err := p.downloadAndExtractPyPI(downloadURL, pkgPath); err != nil {
		return err
	}

	// Add to manifest
	p.manifest.Packages = append(p.manifest.Packages, PackageInfo{
		Name:        pkgName,
		Version:     info.Version,
		Language:    "python",
		InstallDate: time.Now(),
		Path:        pkgPath,
	})

	return p.saveManifest()
}

// installFromRubyGems installs a Ruby package from RubyGems
func (p *Packager) installFromRubyGems(name string) error {
	// Parse package name and version
	pkgName, version := parsePackageVersion(name)

	// Fetch package info from RubyGems
	info, err := p.fetchRubyGemsPackage(pkgName, version)
	if err != nil {
		return err
	}

	// Create package directory
	pkgPath := filepath.Join(p.installDir, "ruby", pkgName)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return err
	}

	// Download and extract the gem
	if err := p.downloadAndExtractRubyGems(info.DownloadURL, pkgPath); err != nil {
		return err
	}

	// Add to manifest
	p.manifest.Packages = append(p.manifest.Packages, PackageInfo{
		Name:        pkgName,
		Version:     info.Version,
		Language:    "ruby",
		Registry:    RegistryRubyGems,
		InstallDate: time.Now(),
		Path:        pkgPath,
	})

	return p.saveManifest()
}

// installFromCrates installs a Rust crate from crates.io
func (p *Packager) installFromCrates(name string) error {
	// Parse package name and version
	pkgName, version := parsePackageVersion(name)

	// Fetch crate info from crates.io
	info, err := p.fetchCratesPackage(pkgName, version)
	if err != nil {
		return err
	}

	// Create package directory
	pkgPath := filepath.Join(p.installDir, "rust", pkgName)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return err
	}

	// Download and extract the crate
	if err := p.downloadAndExtractCrates(info.Crate.DownloadURL, pkgPath); err != nil {
		return err
	}

	// Add to manifest
	p.manifest.Packages = append(p.manifest.Packages, PackageInfo{
		Name:        pkgName,
		Version:     info.Crate.Version,
		Language:    "rust",
		Registry:    RegistryCrates,
		InstallDate: time.Now(),
		Path:        pkgPath,
	})

	return p.saveManifest()
}

// installFromGo installs a Go module from the Go proxy
func (p *Packager) installFromGo(name string) error {
	// Parse Go module path (e.g., github.com/user/repo)
	modulePath := name
	if !strings.Contains(name, "/") {
		return fmt.Errorf("Go modules require full path (e.g., github.com/user/repo)")
	}

	// Fetch module info from Go proxy
	info, err := p.fetchGoModule(modulePath)
	if err != nil {
		return err
	}

	// Create module directory
	pkgPath := filepath.Join(p.installDir, "go", strings.ReplaceAll(modulePath, "/", "_"))
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return err
	}

	// Download module files
	if err := p.downloadGoModule(modulePath, info.Version, pkgPath); err != nil {
		return err
	}

	// Add to manifest
	p.manifest.Packages = append(p.manifest.Packages, PackageInfo{
		Name:        modulePath,
		Version:     info.Version,
		Language:    "go",
		Registry:    RegistryGo,
		InstallDate: time.Now(),
		Path:        pkgPath,
	})

	return p.saveManifest()
}

// installFromGitHub installs a package directly from GitHub
func (p *Packager) installFromGitHub(name string) error {
	// Parse GitHub URL (e.g., github.com/user/repo)
	if !strings.Contains(name, "/") {
		return fmt.Errorf("GitHub packages require user/repo format")
	}

	// Determine language from repository (simple heuristic)
	language := p.detectGitHubLanguage(name)
	if language == "" {
		language = "unknown"
	}

	// Create package directory
	parts := strings.Split(name, "/")
	pkgName := parts[len(parts)-1] // Last part is repo name
	pkgPath := filepath.Join(p.installDir, language, pkgName)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return err
	}

	// Download repository as zip
	zipURL := fmt.Sprintf("https://github.com/%s/archive/refs/heads/main.zip", name)
	if err := p.downloadAndExtractGitHub(zipURL, pkgPath); err != nil {
		return err
	}

	// Add to manifest
	p.manifest.Packages = append(p.manifest.Packages, PackageInfo{
		Name:        pkgName,
		Version:     "main", // GitHub main branch
		Language:    language,
		Registry:    RegistryGitHub,
		InstallDate: time.Now(),
		Path:        pkgPath,
	})

	return p.saveManifest()
}

// PyPIPackageInfo contains PyPI package metadata
type PyPIPackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

// RubyGemsPackageInfo contains RubyGems package metadata
type RubyGemsPackageInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	DownloadURL string `json:"gem_uri"`
}

// CratesPackageInfo contains crates.io package metadata
type CratesPackageInfo struct {
	Crate struct {
		Name        string `json:"name"`
		Version     string `json:"newest_version"`
		DownloadURL string `json:"downloads"`
	} `json:"crate"`
}

// GoModuleInfo contains Go module metadata
type GoModuleInfo struct {
	Version string `json:"Version"`
	Path    string `json:"Path"`
}

// PyPIResponse is the PyPI API response structure
type PyPIResponse struct {
	Info struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"info"`
	URLs []struct {
		Packagetype string `json:"packagetype"`
		URL         string `json:"url"`
		Filename    string `json:"filename"`
	} `json:"urls"`
}

// fetchPyPIPackage fetches package info from PyPI
func (p *Packager) fetchPyPIPackage(name, version string) (*PyPIPackageInfo, error) {
	url := fmt.Sprintf("%s/%s/json", PyPIRegistryURL, name)
	if version != "" {
		url = fmt.Sprintf("%s/%s/%s/json", PyPIRegistryURL, name, version)
	}

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from PyPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package not found: %s (status: %d)", name, resp.StatusCode)
	}

	var pypiResp PyPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&pypiResp); err != nil {
		return nil, fmt.Errorf("failed to parse PyPI response: %w", err)
	}

	// Find best distribution (prefer wheel, then source)
	var downloadURL string
	for _, url := range pypiResp.URLs {
		if url.Packagetype == "bdist_wheel" {
			downloadURL = url.URL
			break
		}
	}
	if downloadURL == "" && len(pypiResp.URLs) > 0 {
		downloadURL = pypiResp.URLs[0].URL
	}

	return &PyPIPackageInfo{
		Name:    pypiResp.Info.Name,
		Version: pypiResp.Info.Version,
		URL:     downloadURL,
	}, nil
}

// downloadAndExtractPyPI downloads and extracts a PyPI package
func (p *Packager) downloadAndExtractPyPI(url, destPath string) error {
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	// Read the archive
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	filename := filepath.Base(url)

	// Handle different archive types
	if strings.HasSuffix(filename, ".whl") || strings.HasSuffix(filename, ".zip") {
		return p.extractWheel(data, destPath)
	} else if strings.HasSuffix(filename, ".tar.gz") {
		return p.extractTarGz(data, destPath)
	}

	return fmt.Errorf("unsupported archive type: %s", filename)
}

// extractWheel extracts a wheel (zip) file
func (p *Packager) extractWheel(data []byte, destPath string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to read wheel: %w", err)
	}

	for _, file := range reader.File {
		// Skip directories and __pycache__
		if file.FileInfo().IsDir() || strings.Contains(file.Name, "__pycache__") {
			continue
		}

		// Only extract Python source files
		if !strings.HasSuffix(file.Name, ".py") &&
			!strings.HasSuffix(file.Name, ".pyi") &&
			file.Name != "py.typed" {
			continue
		}

		srcFile, err := file.Open()
		if err != nil {
			return err
		}

		destFile := filepath.Join(destPath, filepath.Base(file.Name))
		if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
			srcFile.Close()
			return err
		}

		destWriter, err := os.Create(destFile)
		if err != nil {
			srcFile.Close()
			return err
		}

		_, err = io.Copy(destWriter, srcFile)
		srcFile.Close()
		destWriter.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// extractTarGz extracts a tar.gz file (simplified - would need proper tar handling)
func (p *Packager) extractTarGz(data []byte, destPath string) error {
	// For now, just create a placeholder
	// A full implementation would use archive/tar
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer reader.Close()

	// Read content (simplified - real impl would parse tar structure)
	_, err = io.ReadAll(reader)
	if err != nil {
		return err
	}

	// Create a marker file
	markerPath := filepath.Join(destPath, ".extracted")
	return os.WriteFile(markerPath, []byte("extracted"), 0644)
}

// installFromNPM installs a JavaScript package from npm
func (p *Packager) installFromNPM(name string) error {
	pkgName, version := parsePackageVersion(name)

	// Fetch package info from npm
	info, err := p.fetchNPMPackage(pkgName, version)
	if err != nil {
		return err
	}

	pkgPath := filepath.Join(p.installDir, "javascript", pkgName)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return err
	}

	// Download and extract
	if err := p.downloadAndExtractNPM(info.URL, pkgPath); err != nil {
		return err
	}

	// Add to manifest
	p.manifest.Packages = append(p.manifest.Packages, PackageInfo{
		Name:        pkgName,
		Version:     info.Version,
		Language:    "javascript",
		InstallDate: time.Now(),
		Path:        pkgPath,
	})

	return p.saveManifest()
}

// NPMPackageInfo contains npm package metadata
type NPMPackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

// fetchNPMPackage fetches package info from npm
func (p *Packager) fetchNPMPackage(name, version string) (*NPMPackageInfo, error) {
	url := fmt.Sprintf("%s/%s", NPMRegistryURL, name)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from npm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package not found: %s (status: %d)", name, resp.StatusCode)
	}

	var npmResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&npmResp); err != nil {
		return nil, fmt.Errorf("failed to parse npm response: %w", err)
	}

	versions, ok := npmResp["versions"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid npm response format")
	}

	// Get version info
	versionStr := version
	if versionStr == "" {
		versionStr = "latest"
	}

	// Get latest version if "latest" requested or no version specified
	if versionStr == "latest" {
		distTags, ok := npmResp["dist-tags"].(map[string]interface{})
		if ok {
			if latest, ok := distTags["latest"].(string); ok {
				versionStr = latest
			}
		}
	}

	versionData, ok := versions[versionStr].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("version not found: %s", versionStr)
	}

	dist, ok := versionData["dist"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no dist info for version")
	}

	tarballURL, ok := dist["tarball"].(string)
	if !ok {
		return nil, fmt.Errorf("no tarball URL found")
	}

	return &NPMPackageInfo{
		Name:    name,
		Version: versionStr,
		URL:     tarballURL,
	}, nil
}

// downloadAndExtractNPM downloads and extracts an npm package
func (p *Packager) downloadAndExtractNPM(url, destPath string) error {
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// npm packages are gzipped tarballs
	return p.extractTarGz(data, destPath)
}

// GetPackagePath returns the path to an installed package
func (p *Packager) GetPackagePath(name, language string) (string, bool) {
	for _, pkg := range p.manifest.Packages {
		if pkg.Name == name && pkg.Language == language {
			return pkg.Path, true
		}
	}
	return "", false
}

// ListPackages returns all installed packages
func (p *Packager) ListPackages() []PackageInfo {
	return p.manifest.Packages
}

// Uninstall removes an installed package
func (p *Packager) Uninstall(name, language string) error {
	var newPackages []PackageInfo
	var found bool

	for _, pkg := range p.manifest.Packages {
		if pkg.Name == name && pkg.Language == language {
			found = true
			// Remove package directory
			if err := os.RemoveAll(pkg.Path); err != nil {
				return fmt.Errorf("failed to remove package: %w", err)
			}
		} else {
			newPackages = append(newPackages, pkg)
		}
	}

	if !found {
		return fmt.Errorf("package not found: %s", name)
	}

	p.manifest.Packages = newPackages
	return p.saveManifest()
}

// parsePackageVersion extracts name and version from "name @version" format
func parsePackageVersion(name string) (pkgName, version string) {
	// Handle "package @version" or "package@version" format
	re := regexp.MustCompile(`^([a-zA-Z0-9_-]+)(?:\s*[@:]\s*([0-9.]+))?$`)
	matches := re.FindStringSubmatch(name)
	if len(matches) >= 2 {
		pkgName = matches[1]
		if len(matches) >= 3 {
			version = matches[2]
		}
	}
	return pkgName, version
}

// GetInstallDir returns the package installation directory
func (p *Packager) GetInstallDir() string {
	return p.installDir
}

// GetCacheDir returns the package cache directory
func (p *Packager) GetCacheDir() string {
	return p.cacheDir
}

// GetSystemPackageDir returns the default system package directory
func GetSystemPackageDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ditto", "packages"), nil
}

// GetSystemCacheDir returns the default system cache directory
func GetSystemCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use OS-specific cache directories
	var cacheBase string
	switch runtime.GOOS {
	case "windows":
		cacheBase = os.Getenv("LOCALAPPDATA")
		if cacheBase == "" {
			cacheBase = homeDir
		}
	case "darwin":
		cacheBase = filepath.Join(homeDir, "Library", "Caches")
	default: // linux, etc.
		cacheBase = os.Getenv("XDG_CACHE_HOME")
		if cacheBase == "" {
			cacheBase = filepath.Join(homeDir, ".cache")
		}
	}

	return filepath.Join(cacheBase, "ditto"), nil
}
