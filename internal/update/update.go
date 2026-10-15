// Package update implements self-update from GitHub Releases.
//
// The flow is: query the latest release, compare versions, download the
// archive matching the current OS/arch, verify its SHA-256 against the
// release's checksums file, then atomically replace the running binary.
package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Asset is a downloadable release artifact.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// Release is the subset of the GitHub release payload we consume.
type Release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

// Client talks to the GitHub API (or a compatible mirror).
type Client struct {
	Repo    string // "owner/repo"
	BaseURL string // API base, defaults to https://api.github.com
	Token   string // optional; printed by env GITHUB_TOKEN / IFLOW_GITHUB_TOKEN
	HTTP    *http.Client
}

// NewClient builds a Client for repo, honoring IFLOW_UPDATE_API_URL and
// GITHUB_TOKEN.
func NewClient(repo string) *Client {
	base := os.Getenv("IFLOW_UPDATE_API_URL")
	if base == "" {
		base = "https://api.github.com"
	}
	token := os.Getenv("IFLOW_GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	return &Client{
		Repo:    repo,
		BaseURL: strings.TrimRight(base, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Latest fetches the most recent release.
func (c *Client) Latest() (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.BaseURL, c.Repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for %s", c.Repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api: %s", resp.Status)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// AssetName is the archive file name for a version on a given platform,
// matching the goreleaser configuration.
func AssetName(version, goos, goarch string) string {
	v := strings.TrimPrefix(version, "v")
	return fmt.Sprintf("iflow_%s_%s_%s.tar.gz", v, goos, goarch)
}

// Find returns the asset with the given name.
func (r *Release) Find(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// CompareSemver compares dotted versions, returning -1, 0 or 1. A leading "v"
// is ignored. Non-numeric suffixes are compared lexically.
func CompareSemver(a, b string) int {
	an := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bn := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(an) || i < len(bn); i++ {
		var av, bv string
		if i < len(an) {
			av = an[i]
		}
		if i < len(bn) {
			bv = bn[i]
		}
		if av == bv {
			continue
		}
		ai, aerr := parseNum(av)
		bi, berr := parseNum(bv)
		if aerr == nil && berr == nil {
			if ai < bi {
				return -1
			}
			return 1
		}
		if av < bv {
			return -1
		}
		return 1
	}
	return 0
}

func parseNum(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

// Check reports whether a newer release than current exists.
func (c *Client) Check(current string) (latest string, hasUpdate bool, rel *Release, err error) {
	rel, err = c.Latest()
	if err != nil {
		return "", false, nil, err
	}
	latest = strings.TrimPrefix(rel.TagName, "v")
	if current == "dev" || current == "" {
		return latest, true, rel, nil
	}
	return latest, CompareSemver(current, rel.TagName) < 0, rel, nil
}

// Apply downloads and installs the release's archive, replacing the running
// binary. It returns the path written.
func (c *Client) Apply(rel *Release) (string, error) {
	assetName := AssetName(rel.TagName, runtime.GOOS, runtime.GOARCH)
	asset, ok := rel.Find(assetName)
	if !ok {
		return "", fmt.Errorf("release %s has no asset %s", rel.TagName, assetName)
	}

	archive, err := c.download(asset.URL)
	if err != nil {
		return "", err
	}

	if sumAsset, ok := rel.Find("checksums.txt"); ok {
		sums, err := c.download(sumAsset.URL)
		if err != nil {
			return "", err
		}
		want, err := checksumFor(string(sums), assetName)
		if err != nil {
			return "", err
		}
		got := sha256Hex(archive)
		if !strings.EqualFold(want, got) {
			return "", fmt.Errorf("checksum mismatch for %s: want %s got %s", assetName, want, got)
		}
	}

	bin, err := extractBinary(archive, "iflow")
	if err != nil {
		return "", err
	}
	return replaceSelf(bin)
}

func (c *Client) download(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" && strings.Contains(url, c.BaseURL) {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func checksumFor(checksums, name string) (string, error) {
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum entry for %s", name)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// extractBinary returns the bytes of the named file from a .tar.gz archive.
func extractBinary(archive []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == name {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", name)
}

// replaceSelf atomically overwrites the running executable with newBytes.
func replaceSelf(newBytes []byte) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	tmp := filepath.Join(dir, "."+filepath.Base(exe)+".new")
	if err := os.WriteFile(tmp, newBytes, 0o755); err != nil {
		return "", err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return "", err
	}
	backup := exe + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(exe, backup); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("cannot replace running binary: %w", err)
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Rename(backup, exe) // restore
		return "", err
	}
	_ = os.Remove(backup)
	return exe, nil
}
