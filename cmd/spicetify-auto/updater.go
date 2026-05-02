package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type updateConfig struct {
	RepoOwner      string
	RepoName       string
	BinaryName     string
	CurrentVersion string
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
	Message string        `json:"message"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func checkAndUpdate(cfg updateConfig) (bool, error) {
	latest, assetURL, err := fetchLatestAsset(cfg)
	if err != nil {
		return false, err
	}
	if !isNewerVersion(latest, cfg.CurrentVersion) {
		return false, nil
	}

	downloadPath, err := downloadAsset(assetURL)
	if err != nil {
		return false, err
	}

	exePath, err := currentExecutable()
	if err != nil {
		return false, err
	}

	if err := startSelfReplace(downloadPath, exePath); err != nil {
		return false, err
	}

	return true, nil
}

func fetchLatestAsset(cfg updateConfig) (string, string, error) {
	release, err := fetchLatestRelease(cfg.RepoOwner, cfg.RepoName)
	if err != nil {
		return "", "", err
	}

	version := normalizeVersion(release.TagName)
	if version == "" {
		return "", "", errors.New("missing release tag")
	}

	assetName, err := buildAssetName(cfg.BinaryName, version)
	if err != nil {
		return "", "", err
	}

	for _, asset := range release.Assets {
		if asset.Name == assetName {
			return version, asset.BrowserDownloadURL, nil
		}
	}

	return version, "", fmt.Errorf("release asset not found: %s", assetName)
}

func fetchLatestRelease(owner, repo string) (githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	client := http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("User-Agent", "spicetify")

	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return githubRelease{}, err
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return githubRelease{}, err
	}

	if release.TagName == "" {
		return githubRelease{}, errors.New("GitHub response: " + release.Message)
	}

	return release, nil
}

func downloadAsset(url string) (string, error) {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	file, err := os.CreateTemp("", "spicetify-*"+suffix)
	if err != nil {
		return "", err
	}
	defer file.Close()

	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}

	if runtime.GOOS != "windows" {
		_ = os.Chmod(file.Name(), 0755)
	}

	return file.Name(), nil
}

func buildAssetName(binaryName, version string) (string, error) {
	arch, err := assetArch()
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s-%s-%s-%s", binaryName, version, runtime.GOOS, arch)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	return name, nil
}

func assetArch() (string, error) {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return "x64", nil
		case "386":
			return "x32", nil
		case "arm64":
			return "arm64", nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	default:
		switch runtime.GOARCH {
		case "amd64":
			return "amd64", nil
		case "arm64":
			return "arm64", nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	}
}

func normalizeVersion(tag string) string {
	return strings.TrimPrefix(tag, "v")
}

func isNewerVersion(latest, current string) bool {
	latest = normalizeVersion(latest)
	current = normalizeVersion(current)

	if current == "" || current == "dev" {
		return latest != ""
	}

	cmp := compareVersions(latest, current)
	if cmp == 0 {
		return false
	}
	if cmp > 0 {
		return true
	}
	return latest != current
}

func compareVersions(a, b string) int {
	aParts := parseVersion(a)
	bParts := parseVersion(b)
	max := len(aParts)
	if len(bParts) > max {
		max = len(bParts)
	}

	for i := 0; i < max; i++ {
		ai := 0
		bi := 0
		if i < len(aParts) {
			ai = aParts[i]
		}
		if i < len(bParts) {
			bi = bParts[i]
		}
		if ai > bi {
			return 1
		}
		if ai < bi {
			return -1
		}
	}

	return 0
}

func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n := 0
		for i := 0; i < len(part); i++ {
			c := part[i]
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		out = append(out, n)
	}
	return out
}
